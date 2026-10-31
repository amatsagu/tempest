package tempest

import (
	"context"
	"io"
	"log"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Represents a Discord rate limit bucket.
type Bucket struct {
	ResetAt   time.Time
	ID        string
	Window    time.Duration
	Remaining int
	Limit     int
	mu        sync.Mutex
}

type RateLimiterOptions struct {
	TraceLogger    *log.Logger
	SweepInterval  time.Duration // By default: 30 minutes
	SweepThreshold int           // By default: 2500 buckets
	SweepEvery     uint64        // Sweep checks every N acquisitions. By default: 50.
	Trace          bool
}

type RateLimiter struct {
	start           time.Time
	routeMapping    map[string]string // Route (Method:Path) -> Bucket ID
	traceLogger     *log.Logger
	buckets         map[string]*Bucket // Bucket ID -> Bucket
	globalWait      atomic.Int64
	sweepCounter    atomic.Uint64
	lastSweep       atomic.Int64
	sweepInterval   time.Duration
	sweepThreshold  int
	sweepEvery      uint64
	bucketHighWater int
	mu              sync.RWMutex
	trace           bool
}

func NewRateLimiter(opt RateLimiterOptions) *RateLimiter {
	sweepInterval := opt.SweepInterval
	if sweepInterval == 0 {
		sweepInterval = RATE_LIMITER_SWEEP_INTERVAL
	}

	sweepThreshold := opt.SweepThreshold
	if sweepThreshold == 0 {
		sweepThreshold = RATE_LIMITER_SWEEP_THRESHOLD
	}

	sweepEvery := opt.SweepEvery
	if sweepEvery == 0 {
		sweepEvery = RATE_LIMITER_SWEEP_EVERY
	}

	trace := opt.Trace
	if !trace && opt.TraceLogger != nil {
		trace = opt.TraceLogger.Writer() != io.Discard
	}

	return &RateLimiter{
		start:          time.Now(),
		buckets:        make(map[string]*Bucket),
		routeMapping:   make(map[string]string),
		sweepInterval:  sweepInterval,
		sweepThreshold: sweepThreshold,
		sweepEvery:     sweepEvery,
		traceLogger:    opt.TraceLogger,
		trace:          trace,
	}
}

func (rl *RateLimiter) tracef(format string, v ...any) {
	if !rl.trace || rl.traceLogger == nil {
		return
	}

	rl.traceLogger.Printf("[(REST) LIMITER] "+format, v...)
}

// Acquires a rate limit slot for route. It stops waiting when ctx is
// canceled or its deadline expires. If the wait is interrupted, no permit is
// consumed and unlock is nil.
func (rl *RateLimiter) Wait(ctx context.Context, route string) (func(headers http.Header), error) {
	bucket, err := rl.acquire(ctx, route)
	if err != nil {
		return nil, err
	}

	return func(headers http.Header) {
		rl.release(bucket, route, headers)
	}, nil
}

// Reserves one request slot from the bucket associated with route.
// The returned bucket must be passed to release once the request completes.
func (rl *RateLimiter) acquire(ctx context.Context, route string) (*Bucket, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rl.recordAcquire()

	for {
		if rl.globalWait.Load() != 0 {
			if err := rl.waitGlobal(ctx); err != nil {
				return nil, err
			}
		}

		bucket := rl.bucketFor(route)
		bucket.mu.Lock()

		if bucket.Remaining > 0 {
			bucket.Remaining--
			bucket.mu.Unlock()
			return bucket, nil
		}

		now := time.Now()
		if now.Before(bucket.ResetAt) {
			bucketID := bucket.ID
			waitDuration := bucket.ResetAt.Sub(now)
			bucket.mu.Unlock()

			rl.tracef("Rate limit hit on route \"%s\" (bucket ID: %s)! Waiting %s...", route, bucketID, waitDuration.Round(time.Millisecond))
			if err := sleepContext(ctx, waitDuration); err != nil {
				return nil, err
			}

			continue
		}

		if bucket.Limit <= 0 {
			bucket.Limit = 1
		}

		window := bucket.Window
		if window <= 0 {
			window = time.Second
		}

		bucket.Remaining = bucket.Limit
		bucket.ResetAt = now.Add(window + RATE_LIMITER_RESET_TIME_FUDGE)
		bucket.Remaining--
		bucket.mu.Unlock()
		return bucket, nil
	}
}

// firstHeaderValue reads a canonical MIME header without the allocation
// performed by http.Header.Get when the lookup key is not already canonical.
func firstHeaderValue(headers http.Header, canonicalKey string) string {
	if values := headers[canonicalKey]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// Consumes the response headers and publishes any rate limit state
// learned from Discord. A nil header set means the request failed before a
// response was received.
func (rl *RateLimiter) release(bucket *Bucket, route string, headers http.Header) {
	if headers == nil {
		return
	}

	// net/http stores response headers under canonical MIME keys. Index the
	// map directly so hot-path header reads do not re-canonicalize every key.
	retryAfterStr := firstHeaderValue(headers, "Retry-After")
	if firstHeaderValue(headers, "X-Ratelimit-Global") == "true" {
		if retryAfterStr != "" {
			retryAfter, _ := strconv.ParseFloat(retryAfterStr, 64)
			rl.globalWait.Store(time.Now().Add(time.Duration(retryAfter*float64(time.Second)) + 250*time.Millisecond).UnixNano())

			rl.tracef("Received global rate limit! Retry after: %f", retryAfter)
		}

		return
	}

	bucketHeader := firstHeaderValue(headers, "X-Ratelimit-Bucket")
	if bucketHeader == "" {
		return
	}

	remainingStr := firstHeaderValue(headers, "X-Ratelimit-Remaining")
	limitStr := firstHeaderValue(headers, "X-Ratelimit-Limit")
	resetAfterStr := firstHeaderValue(headers, "X-Ratelimit-Reset-After")

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	rl.mu.Lock()
	if bucket.ID != bucketHeader {
		if _, exists := rl.buckets[bucketHeader]; !exists {
			oldID := bucket.ID
			bucket.ID = bucketHeader
			rl.buckets[bucketHeader] = bucket
			rl.routeMapping[route] = bucketHeader

			if current, ok := rl.buckets[oldID]; ok && current == bucket {
				delete(rl.buckets, oldID)
			}

			if len(rl.buckets) > rl.bucketHighWater {
				rl.bucketHighWater = len(rl.buckets)
			}
		}

		// If the bucket already exists, keep this route on its own bucket.
		// Remapping here would point subsequent requests at state that this
		// response does not update.
	}
	rl.mu.Unlock()

	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			bucket.Limit = limit
		}
	}
	if remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil && remaining >= 0 {
			bucket.Remaining = remaining
		}
	}
	if resetAfterStr != "" {
		if resetAfter, err := strconv.ParseFloat(resetAfterStr, 64); err == nil && resetAfter >= 0 {
			window := time.Duration(resetAfter * float64(time.Second))
			bucket.Window = window
			bucket.ResetAt = time.Now().Add(window + RATE_LIMITER_RESET_TIME_FUDGE)
		}
	}
}

func (rl *RateLimiter) waitGlobal(ctx context.Context) error {
	for {
		globalWaitNano := rl.globalWait.Load()
		if globalWaitNano == 0 {
			return nil
		}

		waitDuration := time.Until(time.Unix(0, globalWaitNano))
		if waitDuration <= 0 {
			rl.globalWait.CompareAndSwap(globalWaitNano, 0)
			continue
		}

		rl.tracef("Global rate limit hit! Waiting %s...", waitDuration.Round(time.Millisecond))
		if err := sleepContext(ctx, waitDuration); err != nil {
			return err
		}
	}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (rl *RateLimiter) recordAcquire() {
	if rl.sweepCounter.Add(1) == rl.sweepEvery {
		rl.sweepCounter.Store(0)
		rl.sweepIfDue()
	}
}

func (rl *RateLimiter) sweepIfDue() {
	if time.Since(rl.start).Nanoseconds()-rl.lastSweep.Load() >= int64(rl.sweepInterval) {
		rl.sweep()
	}
}

// Returns the bucket for route (creating it when necessary).
func (rl *RateLimiter) bucketFor(route string) *Bucket {
	rl.mu.RLock()
	if bucketID, ok := rl.routeMapping[route]; ok {
		if bucket := rl.buckets[bucketID]; bucket != nil {
			rl.mu.RUnlock()
			return bucket
		}
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if bucketID, ok := rl.routeMapping[route]; ok {
		if bucket := rl.buckets[bucketID]; bucket != nil {
			return bucket
		}
		delete(rl.routeMapping, route)
	}

	// Route mapping may have been cleared by a sweep while its bucket is
	// still keyed by the route. Reuse that bucket instead of orphaning it.
	if bucket := rl.buckets[route]; bucket != nil {
		rl.routeMapping[route] = route
		return bucket
	}

	bucket := &Bucket{
		ID:        route,
		Limit:     1,
		Remaining: 1,
	}

	rl.routeMapping[route] = route
	rl.buckets[route] = bucket
	if len(rl.buckets) > rl.bucketHighWater {
		rl.bucketHighWater = len(rl.buckets)
	}

	return bucket
}

func (rl *RateLimiter) bucketForBytes(keyBytes []byte) *Bucket {
	rl.mu.RLock()
	if bucketID, ok := rl.routeMapping[string(keyBytes)]; ok {
		if bucket := rl.buckets[bucketID]; bucket != nil {
			rl.mu.RUnlock()
			return bucket
		}
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	route := string(keyBytes)
	if bucketID, ok := rl.routeMapping[route]; ok {
		if bucket := rl.buckets[bucketID]; bucket != nil {
			return bucket
		}
		delete(rl.routeMapping, route)
	}

	if bucket := rl.buckets[route]; bucket != nil {
		rl.routeMapping[route] = route
		return bucket
	}

	bucket := &Bucket{
		ID:        route,
		Limit:     1,
		Remaining: 1,
	}

	rl.routeMapping[route] = route
	rl.buckets[route] = bucket
	if len(rl.buckets) > rl.bucketHighWater {
		rl.bucketHighWater = len(rl.buckets)
	}

	return bucket
}

func (rl *RateLimiter) acquireRoute(ctx context.Context, method, routePath string) (*Bucket, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rl.recordAcquire()

	var buf [128]byte
	var keyBytes []byte
	totalLen := len(method) + 1 + len(routePath)
	if totalLen <= len(buf) {
		n := copy(buf[:], method)
		buf[n] = ':'
		copy(buf[n+1:], routePath)
		keyBytes = buf[:totalLen]
	} else {
		keyBytes = []byte(method + ":" + routePath)
	}

	for {
		if rl.globalWait.Load() != 0 {
			if err := rl.waitGlobal(ctx); err != nil {
				return nil, err
			}
		}

		bucket := rl.bucketForBytes(keyBytes)
		bucket.mu.Lock()

		if bucket.Remaining > 0 {
			bucket.Remaining--
			bucket.mu.Unlock()
			return bucket, nil
		}

		now := time.Now()
		if now.Before(bucket.ResetAt) {
			bucketID := bucket.ID
			waitDuration := bucket.ResetAt.Sub(now)
			bucket.mu.Unlock()

			if rl.trace {
				rl.tracef("Rate limit hit on route \"%s\" (bucket ID: %s)! Waiting %s...", string(keyBytes), bucketID, waitDuration.Round(time.Millisecond))
			}
			if err := sleepContext(ctx, waitDuration); err != nil {
				return nil, err
			}

			continue
		}

		if bucket.Limit <= 0 {
			bucket.Limit = 1
		}

		window := bucket.Window
		if window <= 0 {
			window = time.Second
		}

		bucket.Remaining = bucket.Limit
		bucket.ResetAt = now.Add(window + RATE_LIMITER_RESET_TIME_FUDGE)
		bucket.Remaining--
		bucket.mu.Unlock()
		return bucket, nil
	}
}

func (rl *RateLimiter) releaseRoute(bucket *Bucket, method, routePath string, headers http.Header) {
	if headers == nil {
		return
	}

	retryAfterStr := firstHeaderValue(headers, "Retry-After")
	if firstHeaderValue(headers, "X-Ratelimit-Global") == "true" {
		if retryAfterStr != "" {
			retryAfter, _ := strconv.ParseFloat(retryAfterStr, 64)
			rl.globalWait.Store(time.Now().Add(time.Duration(retryAfter*float64(time.Second)) + 250*time.Millisecond).UnixNano())

			rl.tracef("Received global rate limit! Retry after: %f", retryAfter)
		}

		return
	}

	bucketHeader := firstHeaderValue(headers, "X-Ratelimit-Bucket")
	if bucketHeader == "" {
		return
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	rl.mu.Lock()
	if bucket.ID != bucketHeader {
		if _, exists := rl.buckets[bucketHeader]; !exists {
			oldID := bucket.ID
			bucket.ID = bucketHeader
			rl.buckets[bucketHeader] = bucket
			route := method + ":" + routePath
			rl.routeMapping[route] = bucketHeader

			if current, ok := rl.buckets[oldID]; ok && current == bucket {
				delete(rl.buckets, oldID)
			}

			if len(rl.buckets) > rl.bucketHighWater {
				rl.bucketHighWater = len(rl.buckets)
			}
		}
	}
	rl.mu.Unlock()

	if limitStr := firstHeaderValue(headers, "X-Ratelimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			bucket.Limit = limit
		}
	}
	if remainingStr := firstHeaderValue(headers, "X-Ratelimit-Remaining"); remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil && remaining >= 0 {
			bucket.Remaining = remaining
		}
	}
	if resetAfterStr := firstHeaderValue(headers, "X-Ratelimit-Reset-After"); resetAfterStr != "" {
		if resetAfter, err := strconv.ParseFloat(resetAfterStr, 64); err == nil && resetAfter >= 0 {
			window := time.Duration(resetAfter * float64(time.Second))
			bucket.Window = window
			bucket.ResetAt = time.Now().Add(window + RATE_LIMITER_RESET_TIME_FUDGE)
		}
	}
}

func (rl *RateLimiter) sweep() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	sinceStart := now.Sub(rl.start).Nanoseconds()
	if sinceStart-rl.lastSweep.Load() < int64(rl.sweepInterval) {
		return
	}
	rl.lastSweep.Store(sinceStart)

	if len(rl.buckets) > rl.bucketHighWater {
		rl.bucketHighWater = len(rl.buckets)
	}

	deleted := 0
	for id, bucket := range rl.buckets {
		if bucket.mu.TryLock() {
			if now.After(bucket.ResetAt) {
				delete(rl.buckets, id)
				deleted++
			}
			bucket.mu.Unlock()
		}
	}

	if deleted > 0 {
		for route, bucketID := range rl.routeMapping {
			if _, ok := rl.buckets[bucketID]; !ok {
				delete(rl.routeMapping, route)
			}
		}
	}

	if len(rl.routeMapping) > rl.sweepThreshold {
		// Replacing the map (rather than clear) releases the old backing
		// storage once in-flight readers are done.
		rl.routeMapping = make(map[string]string)
	}

	if deleted > 0 && rl.bucketHighWater > 0 && len(rl.buckets) <= rl.bucketHighWater/2 {
		liveBuckets := make(map[string]*Bucket, len(rl.buckets))
		maps.Copy(liveBuckets, rl.buckets)
		rl.buckets = liveBuckets
		rl.bucketHighWater = len(liveBuckets)
	}
}

type rateLimitTransport struct {
	limiter        *RateLimiter
	innerTransport http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	method := req.Method
	routePath := extractRoute(req.URL.Path)
	bucket, err := t.limiter.acquireRoute(req.Context(), method, routePath)
	if err != nil {
		return nil, err
	}

	resp, err := t.innerTransport.RoundTrip(req)
	if err != nil {
		t.limiter.releaseRoute(bucket, method, routePath, nil)
		return nil, err
	}

	t.limiter.releaseRoute(bucket, method, routePath, resp.Header)
	return resp, nil
}

func extractRoute(path string) string {
	orig := path
	if strings.HasPrefix(path, "/api/v") {
		idx := strings.IndexByte(path[6:], '/')
		if idx != -1 {
			path = path[6+idx:]
		}
	}

	var rest string
	var major string
	if strings.HasPrefix(path, "/channels/") {
		major = "/channels/"
		rest = path[10:]
	} else if strings.HasPrefix(path, "/guilds/") {
		major = "/guilds/"
		rest = path[8:]
	} else if strings.HasPrefix(path, "/webhooks/") {
		major = "/webhooks/"
		rest = path[10:]
	} else {
		return orig
	}

	idx1 := strings.IndexByte(rest, '/')
	if idx1 == -1 {
		return path
	}

	idx2 := strings.IndexByte(rest[idx1+1:], '/')
	if idx2 == -1 {
		return path
	}

	totalLen := len(major) + idx1 + 1 + idx2
	return path[:totalLen]
}
