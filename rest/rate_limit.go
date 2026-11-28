package rest

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"weak"
)

const (
	bucketRetryFloor = 250 * time.Millisecond
	evictSampleCount = 16
	initShardMapCap  = 32
	retryFloorNano   = int64(bucketRetryFloor)
)

// Pre-canonicalized HTTP header keys so hot-path lookups allocate zero bytes.
var (
	headerRetryAfter = textproto.CanonicalMIMEHeaderKey("Retry-After")
	headerVia        = textproto.CanonicalMIMEHeaderKey("Via")
	headerGlobal     = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Global")
	headerScope      = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Scope")
	headerBucket     = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Bucket")
	headerLimit      = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Limit")
	headerRemaining  = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Remaining")
	headerReset      = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Reset")
	headerResetAfter = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Reset-After")
)

// Learned Discord rate-limit window state for one route bucket.
type Bucket struct {
	ID        string
	ResetAt   int64
	LastUsed  int64
	Window    int64
	Remaining int32
	Limit     int32
	evicted   bool
}

// Configuration for a sharded Discord rate limiter.
type RateLimiterConfig struct {
	TraceLogger   *log.Logger
	Now           func() time.Time
	GlobalRate    float64       // Proactive req/s cap (<= 0 disables).
	IdleTTL       time.Duration // Reclaim idle buckets after this duration (< 0 disables). Default: 10m.
	ResetFudge    time.Duration // Safety padding on reset windows (< 0 disables). Default: 100ms.
	Shards        int           // Shard count (rounded up to power of two). Default: 64.
	SweepInterval int           // Updates between amortized shard sweeps. Default: 512.
	MaxBuckets    int           // Max tracked buckets across shards. Default: 65536.
	MaxRoutes     int           // Max cached route->bucket mappings across shards. Default: 65536.
	Trace         bool
}

func defaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Now:           time.Now,
		IdleTTL:       10 * time.Minute,
		ResetFudge:    100 * time.Millisecond,
		Shards:        64,
		SweepInterval: 512,
		MaxBuckets:    65536,
		MaxRoutes:     65536,
	}
}

type limiterShard struct {
	buckets      map[string]*Bucket // Keyed by "<bucketHeader>\x00<major>" to allow 0-alloc stack lookups.
	routes       map[string]*Bucket
	mu           sync.Mutex
	maxBuckets   int
	maxRoutes    int
	ops          int
	bucketEvicts uint32
	routeEvicts  uint32
	grown        bool
	_            [7]byte // Keep each shard on its own 64-byte cache line.
}

type autoSweepHandle struct {
	timer   *time.Timer
	mu      sync.Mutex
	stopped bool
}

// Tracks Discord per-route buckets and global pauses across cache-aligned shards.
type RateLimiter struct {
	global         *globalPacer
	now            func() time.Time
	traceLogger    *log.Logger
	sweeper        *autoSweepHandle
	shards         []limiterShard
	globalWait     atomic.Int64 // UnixNano deadline when a Discord global pause expires
	mask           uint64
	idleTTL        int64
	resetFudgeNano int64
	sweepInterval  int
	sweepArmed     atomic.Bool
	trace          bool
}

type globalPacer struct {
	tokens   float64
	capacity float64
	rate     float64
	lastNano int64
	mu       sync.Mutex
}

// Creates a sharded Discord rate limiter.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	def := defaultRateLimiterConfig()

	now := cfg.Now
	if now == nil {
		now = def.Now
	}

	shards := cfg.Shards
	if shards <= 0 {
		shards = def.Shards
	}
	shards = nextPowerOfTwo(shards)

	idleTTL := cfg.IdleTTL
	if idleTTL == 0 {
		idleTTL = def.IdleTTL
	} else if idleTTL < 0 {
		idleTTL = 0
	}

	resetFudge := cfg.ResetFudge
	if resetFudge == 0 {
		resetFudge = def.ResetFudge
	} else if resetFudge < 0 {
		resetFudge = 0
	}

	sweepInterval := cfg.SweepInterval
	if sweepInterval <= 0 {
		sweepInterval = def.SweepInterval
	}

	maxBuckets := cfg.MaxBuckets
	if maxBuckets <= 0 {
		maxBuckets = def.MaxBuckets
	}

	maxRoutes := cfg.MaxRoutes
	if maxRoutes <= 0 {
		maxRoutes = def.MaxRoutes
	}

	if cfg.Shards <= 0 {
		for shards > 1 && (shards > maxBuckets || shards > maxRoutes) {
			shards >>= 1
		}
	}

	perShardBuckets := max(maxBuckets/shards, 1)
	perShardRoutes := max(maxRoutes/shards, 1)

	traceLogger := cfg.TraceLogger
	trace := cfg.Trace
	if !trace && traceLogger != nil {
		trace = traceLogger.Writer() != io.Discard
	}
	if trace && traceLogger == nil {
		traceLogger = log.New(os.Stdout, "[TEMPEST] ", log.LstdFlags)
	}

	var pacer *globalPacer
	if cfg.GlobalRate > 0 {
		pacer = &globalPacer{
			tokens:   cfg.GlobalRate,
			capacity: cfg.GlobalRate,
			rate:     cfg.GlobalRate,
			lastNano: now().UnixNano(),
		}
	}

	var sweeper *autoSweepHandle
	if idleTTL > 0 {
		sweeper = &autoSweepHandle{}
	}

	rl := &RateLimiter{
		global:         pacer,
		now:            now,
		traceLogger:    traceLogger,
		sweeper:        sweeper,
		shards:         make([]limiterShard, shards),
		mask:           uint64(shards - 1),
		idleTTL:        idleTTL.Nanoseconds(),
		resetFudgeNano: resetFudge.Nanoseconds(),
		sweepInterval:  sweepInterval,
		trace:          trace,
	}

	initCap := min(perShardBuckets, initShardMapCap)
	for i := range rl.shards {
		rl.shards[i] = limiterShard{
			buckets:    make(map[string]*Bucket, initCap),
			routes:     make(map[string]*Bucket, initCap),
			maxBuckets: perShardBuckets,
			maxRoutes:  perShardRoutes,
		}
	}

	if sweeper != nil {
		runtime.AddCleanup(rl, func(h *autoSweepHandle) {
			h.mu.Lock()
			h.stopped = true
			if h.timer != nil {
				h.timer.Stop()
				h.timer = nil
			}
			h.mu.Unlock()
		}, sweeper)
	}

	return rl
}

func (rl *RateLimiter) tracef(format string, v ...any) {
	if !rl.trace || rl.traceLogger == nil {
		return
	}

	rl.traceLogger.Printf("[(REST) LIMITER] "+format, v...)
}

// Waits for a route rate-limit slot and returns a callback to update state from response headers.
func (rl *RateLimiter) Wait(ctx context.Context, method, path string) (func(http.Header), error) {
	var stackBuf [192]byte
	keyBytes, majorStart, majorEnd := appendRouteKey(stackBuf[:0], method, path)
	major := keyBytes[majorStart:majorEnd]

	if err := rl.acquireKey(ctx, keyBytes, major, false); err != nil {
		return nil, err
	}

	keyStr := string(keyBytes)
	majorStr := string(major)
	return func(headers http.Header) {
		rl.releaseKey([]byte(keyStr), []byte(majorStr), headers)
	}, nil
}

func (rl *RateLimiter) acquireKey(ctx context.Context, keyBytes, major []byte, globalExempt bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if !globalExempt {
		if err := rl.waitGlobal(ctx); err != nil {
			return err
		}

		if rl.global != nil {
			for {
				wait, ok := rl.global.reserve(rl.now().UnixNano())
				if ok {
					break
				}

				if err := sleepContext(ctx, wait); err != nil {
					return err
				}
			}
		}
	}

	sh := &rl.shards[fnv1aBytes(major)&rl.mask]

	for {
		now := rl.now().UnixNano()
		sh.mu.Lock()

		b := sh.routes[string(keyBytes)]
		if b == nil {
			// Optimistic pass on unseen routes avoids writing to shard maps before
			// Discord confirms bucket headers (or for one-off token routes).
			sh.mu.Unlock()
			return nil
		}

		if b.evicted {
			delete(sh.routes, string(keyBytes))
			sh.mu.Unlock()
			return nil
		}

		b.LastUsed = now

		if now >= b.ResetAt {
			limit := max(b.Limit, 1)
			window := b.Window
			if window <= 0 {
				window = int64(time.Second)
			}
			b.Remaining = limit
			b.ResetAt = now + window + rl.resetFudgeNano
		}

		if b.Remaining > 0 {
			b.Remaining--
			sh.mu.Unlock()
			return nil
		}

		wait := max(time.Duration(b.ResetAt-now), bucketRetryFloor)
		bucketID := b.ID
		sh.mu.Unlock()

		if rl.trace {
			rl.tracef("Rate limit hit on route %q (bucket ID: %s); waiting %s", string(keyBytes), bucketID, wait.Round(time.Millisecond))
		}

		if err := sleepContext(ctx, wait); err != nil {
			if rl.global != nil && !globalExempt {
				rl.global.refund()
			}
			return err
		}
	}
}

func (rl *RateLimiter) releaseKey(keyBytes, major []byte, headers http.Header) {
	if headers == nil {
		return
	}

	if firstHeaderValue(headers, headerGlobal) == "true" || firstHeaderValue(headers, headerScope) == "global" {
		if retryAfterStr := firstHeaderValue(headers, headerRetryAfter); retryAfterStr != "" {
			if secs, err := strconv.ParseFloat(retryAfterStr, 64); err == nil && secs > 0 {
				pauseUntil := rl.now().UnixNano() + int64(secs*float64(time.Second)) + int64(250*time.Millisecond)
				rl.setGlobalPause(pauseUntil)
				rl.tracef("Received global rate limit; pausing for %.3fs", secs)
			}
		}

		return
	}

	bucketHeader := firstHeaderValue(headers, headerBucket)
	remainingStr := firstHeaderValue(headers, headerRemaining)
	if bucketHeader == "" && remainingStr == "" {
		return
	}

	limit := 0
	if limitStr := firstHeaderValue(headers, headerLimit); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	remaining := -1
	if remainingStr != "" {
		if parsed, err := strconv.Atoi(remainingStr); err == nil && parsed >= 0 {
			remaining = parsed
		}
	}

	now := rl.now().UnixNano()

	var window int64
	hasWindow := false
	if resetAfterStr := firstHeaderValue(headers, headerResetAfter); resetAfterStr != "" {
		if secs, err := strconv.ParseFloat(resetAfterStr, 64); err == nil && secs >= 0 {
			window = int64(secs * float64(time.Second))
			hasWindow = true
		}
	} else if resetStr := firstHeaderValue(headers, headerReset); resetStr != "" {
		if resetSec, err := strconv.ParseFloat(resetStr, 64); err == nil && resetSec > 0 {
			window = max(int64(resetSec*float64(time.Second))-now, 0)
			hasWindow = true
		}
	}

	if bucketHeader == "" && remaining < 0 && !hasWindow {
		return
	}

	sh := &rl.shards[fnv1aBytes(major)&rl.mask]

	createdEntry := false

	sh.mu.Lock()

	b := sh.routes[string(keyBytes)]
	if b != nil && b.evicted {
		delete(sh.routes, string(keyBytes))
		b = nil
	}

	lookupID := bucketHeader
	// Fast path: route already points to its active bucket with the same X-RateLimit-Bucket hash.
	// Skips bucket map hashing/key formatting completely with zero heap allocations.
	if b == nil || (lookupID != "" && b.ID != lookupID) {
		var bkStack [128]byte
		if lookupID != "" {
			bkBytes := appendBucketMapKey(bkStack[:0], lookupID, major)
			if existing := sh.buckets[string(bkBytes)]; existing != nil && !existing.evicted {
				b = existing
			} else if b != nil && b.ID != lookupID {
				// Bucket hash migrated and target key doesn't exist yet; re-key existing bucket.
				oldKey := appendBucketMapKey(bkStack[:0], b.ID, major)
				delete(sh.buckets, string(oldKey))
				b.ID = lookupID
				bkBytes = appendBucketMapKey(bkStack[:0], lookupID, major)
				sh.buckets[string(bkBytes)] = b
			}
		}

		if b == nil {
			if len(sh.buckets) >= sh.maxBuckets {
				evictBucketBatchLocked(sh, now, rl.idleTTL)
			}

			if len(sh.buckets) < sh.maxBuckets {
				if lookupID == "" {
					lookupID = string(keyBytes)
				}
				bkBytes := appendBucketMapKey(bkStack[:0], lookupID, major)
				b = &Bucket{ID: lookupID}
				sh.buckets[string(bkBytes)] = b
				createdEntry = true
				if len(sh.buckets) > initShardMapCap {
					sh.grown = true
				}
			}
		}
	}

	if b == nil {
		sh.mu.Unlock()
		return
	}

	if sh.routes[string(keyBytes)] != b {
		if len(sh.routes) >= sh.maxRoutes {
			evictRouteBatchLocked(sh, now, rl.idleTTL)
		}
		if len(sh.routes) < sh.maxRoutes {
			sh.routes[string(keyBytes)] = b
			if len(sh.routes) > initShardMapCap {
				sh.grown = true
			}
		}
	}

	if limit > 0 {
		b.Limit = int32(limit)
	} else if remaining >= 0 && int32(remaining) > b.Limit {
		b.Limit = int32(remaining)
	}

	if remaining >= 0 {
		b.Remaining = int32(remaining)
	}

	if hasWindow {
		b.Window = window
		b.ResetAt = now + window + rl.resetFudgeNano
	}

	b.LastUsed = now

	sh.ops++
	if rl.idleTTL > 0 && sh.ops >= rl.sweepInterval {
		sh.ops = 0
		sweepShardLocked(sh, now, rl.idleTTL)
	}

	sh.mu.Unlock()

	if createdEntry && rl.idleTTL > 0 && !rl.sweepArmed.Load() {
		rl.armAutoSweep()
	}
}

func (rl *RateLimiter) armAutoSweep() {
	if rl.idleTTL <= 0 || rl.sweeper == nil {
		return
	}

	if !rl.sweepArmed.CompareAndSwap(false, true) {
		return
	}

	delay := time.Duration(rl.idleTTL) + max(time.Duration(rl.idleTTL)/10, 5*time.Millisecond)
	rl.scheduleSweep(weak.Make(rl), delay)
}

func (rl *RateLimiter) scheduleSweep(wp weak.Pointer[RateLimiter], delay time.Duration) {
	h := rl.sweeper
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return
	}

	if h.timer != nil {
		h.timer.Stop()
	}

	h.timer = time.AfterFunc(delay, func() {
		cur := wp.Value()
		if cur == nil {
			return
		}

		cur.Sweep(0)
		buckets, routes := cur.Size()
		if buckets == 0 && routes == 0 {
			cur.sweepArmed.Store(false)

			buckets, routes = cur.Size()
			if buckets == 0 && routes == 0 {
				return
			}

			cur.armAutoSweep()
			return
		}

		nextDelay := max(time.Duration(cur.idleTTL)/2, 20*time.Millisecond)
		cur.scheduleSweep(wp, nextDelay)
	})
}

// Evicts idle buckets and stale routes across all shards and returns the removed count.
func (rl *RateLimiter) Sweep(idleAfter time.Duration) int {
	idleNano := idleAfter.Nanoseconds()
	if idleNano <= 0 {
		idleNano = rl.idleTTL
	}
	if idleNano <= 0 {
		return 0
	}

	now := rl.now().UnixNano()
	removed := 0
	for i := range rl.shards {
		sh := &rl.shards[i]
		sh.mu.Lock()
		removed += sweepShardLocked(sh, now, idleNano)
		sh.mu.Unlock()
	}

	return removed
}

// Returns active tracked bucket and route counts across all shards.
func (rl *RateLimiter) Size() (buckets, routes int) {
	for i := range rl.shards {
		sh := &rl.shards[i]
		sh.mu.Lock()
		buckets += len(sh.buckets)
		routes += len(sh.routes)
		sh.mu.Unlock()
	}

	return buckets, routes
}

func sweepShardLocked(sh *limiterShard, now, idleNano int64) int {
	removed := 0
	for k, b := range sh.buckets {
		if now-b.LastUsed > idleNano && now >= b.ResetAt {
			b.evicted = true
			delete(sh.buckets, k)
			removed++
		}
	}

	if removed > 0 {
		sh.bucketEvicts += uint32(removed)

		for rk, b := range sh.routes {
			if b.evicted || now-b.LastUsed > idleNano {
				delete(sh.routes, rk)
				sh.routeEvicts++
			}
		}
	}

	if sh.grown && len(sh.buckets) == 0 && len(sh.routes) == 0 {
		sh.buckets = make(map[string]*Bucket, min(sh.maxBuckets, initShardMapCap))
		sh.routes = make(map[string]*Bucket, min(sh.maxRoutes, initShardMapCap))
		sh.bucketEvicts = 0
		sh.routeEvicts = 0
		sh.grown = false
		return removed
	}

	if sh.grown && removed > 0 && len(sh.buckets) <= initShardMapCap && len(sh.routes) <= initShardMapCap {
		compactShardMapsLocked(sh)
		sh.grown = false
	}

	return removed
}

func appendBucketMapKey(dst []byte, hash string, major []byte) []byte {
	dst = append(dst, hash...)
	dst = append(dst, 0)
	return append(dst, major...)
}

// Rebuilds shard maps after heavy eviction churn so Go's SwissTable directories cannot
// accumulate split sub-tables and dangling evicted routes over millions of churn cycles.
func compactShardMapsLocked(sh *limiterShard) {
	bucketCap := max(len(sh.buckets), min(sh.maxBuckets, initShardMapCap))
	freshBuckets := make(map[string]*Bucket, bucketCap)
	for k, b := range sh.buckets {
		if !b.evicted {
			freshBuckets[k] = b
		}
	}
	sh.buckets = freshBuckets
	sh.bucketEvicts = 0

	compactRoutesLocked(sh)
}

func compactRoutesLocked(sh *limiterShard) {
	routeCap := max(len(sh.routes), min(sh.maxRoutes, initShardMapCap))
	freshRoutes := make(map[string]*Bucket, routeCap)
	for k, b := range sh.routes {
		if !b.evicted {
			freshRoutes[k] = b
		}
	}
	sh.routes = freshRoutes
	sh.routeEvicts = 0
}

// Evicts one expired or oldest-sampled bucket in O(1) bounded work.
func evictBucketBatchLocked(sh *limiterShard, now, idleNano int64) {
	var (
		victimKey   string
		victim      *Bucket
		oldestUsed  int64 = 1<<63 - 1
		expiredSeen       = false
		checked           = 0
	)

	for k, b := range sh.buckets {
		if now >= b.ResetAt && (idleNano <= 0 || now-b.LastUsed >= idleNano) {
			b.evicted = true
			delete(sh.buckets, k)
			expiredSeen = true
			break
		}

		if b.LastUsed < oldestUsed {
			oldestUsed = b.LastUsed
			victimKey = k
			victim = b
		}

		checked++
		if checked >= evictSampleCount {
			break
		}
	}

	if !expiredSeen && victim != nil {
		victim.evicted = true
		delete(sh.buckets, victimKey)
	}

	sh.bucketEvicts++
	if sh.bucketEvicts >= uint32(max(sh.maxBuckets*2, initShardMapCap*2)) {
		compactShardMapsLocked(sh)
	}
}

func evictRouteBatchLocked(sh *limiterShard, now, idleNano int64) {
	checked := 0
	evicted := false
	var fallbackKey string
	var oldestUsed int64 = 1<<63 - 1

	for k, b := range sh.routes {
		if b.evicted || (idleNano > 0 && now-b.LastUsed > idleNano) {
			delete(sh.routes, k)
			evicted = true
			break
		}

		if b.LastUsed < oldestUsed {
			oldestUsed = b.LastUsed
			fallbackKey = k
		}

		checked++
		if checked >= evictSampleCount {
			break
		}
	}

	if !evicted && fallbackKey != "" {
		delete(sh.routes, fallbackKey)
		evicted = true
	}

	if evicted {
		sh.routeEvicts++
		if sh.routeEvicts >= uint32(max(sh.maxRoutes*2, initShardMapCap*2)) {
			compactRoutesLocked(sh)
		}
	}
}

func (rl *RateLimiter) setGlobalPause(untilNano int64) {
	for {
		cur := rl.globalWait.Load()
		if untilNano <= cur {
			return
		}
		if rl.globalWait.CompareAndSwap(cur, untilNano) {
			return
		}
	}
}

func (rl *RateLimiter) waitGlobal(ctx context.Context) error {
	for {
		until := rl.globalWait.Load()
		if until == 0 {
			return nil
		}

		wait := time.Duration(until - rl.now().UnixNano())
		if wait <= 0 {
			rl.globalWait.CompareAndSwap(until, 0)
			continue
		}

		rl.tracef("Global rate limit active; waiting %s", wait.Round(time.Millisecond))
		if err := sleepContext(ctx, wait); err != nil {
			return err
		}
	}
}

func (g *globalPacer) reserve(now int64) (time.Duration, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if elapsed := now - g.lastNano; elapsed > 0 {
		g.tokens += float64(elapsed) / float64(time.Second) * g.rate
		if g.tokens > g.capacity {
			g.tokens = g.capacity
		}
		g.lastNano = now
	}

	if g.tokens >= 1 {
		g.tokens--
		return 0, true
	}

	needed := (1 - g.tokens) / g.rate
	return time.Duration(needed * float64(time.Second)), false
}

func (g *globalPacer) refund() {
	g.mu.Lock()
	if g.tokens < g.capacity {
		g.tokens++
	}
	g.mu.Unlock()
}

func firstHeaderValue(headers http.Header, canonicalKey string) string {
	if values := headers[canonicalKey]; len(values) > 0 {
		return values[0]
	}

	return ""
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}

	// Fast-fail before arming a timer when the remaining context deadline is shorter than the required sleep.
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < d {
		return context.DeadlineExceeded
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type rateLimitTransport struct {
	limiter        *RateLimiter
	innerTransport http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var stackBuf [192]byte
	keyBytes, majorStart, majorEnd := appendRouteKey(stackBuf[:0], req.Method, req.URL.Path)

	// Interaction callback paths are exempt from Discord's global limit and must
	// never stall on global pauses.
	globalExempt := strings.Contains(req.URL.Path, "/interactions/")
	major := keyBytes[majorStart:majorEnd]

	if err := t.limiter.acquireKey(req.Context(), keyBytes, major, globalExempt); err != nil {
		return nil, err
	}

	resp, err := t.innerTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if !globalExempt {
		t.limiter.releaseKey(keyBytes, major, resp.Header)
	}

	return resp, nil
}

// Normalizes an HTTP method and Discord API path into a canonical bucket route.
func RouteKey(method, rawPath string) string {
	var stackBuf [192]byte
	out, _, _ := appendRouteKey(stackBuf[:0], method, rawPath)
	return string(out)
}

func appendRouteKey(dst []byte, method, rawPath string) (out []byte, majorStart, majorEnd int) {
	path := rawPath
	if q := strings.IndexByte(path, '?'); q >= 0 {
		path = path[:q]
	}

	if strings.HasPrefix(path, "/api/v") {
		if restStart := strings.IndexByte(path[6:], '/'); restStart >= 0 {
			path = path[6+restStart:]
		}
	}

	dst = append(dst, method...)
	dst = append(dst, ' ')
	if strings.HasPrefix(path, "/") {
		dst = append(dst, '/')
	}

	var (
		top           string
		prevSeg       string
		segment       int
		hasMajorParam bool
		start         int
	)

	routePrefixStart := len(dst)

	for i := 0; i <= len(path); i++ {
		if i != len(path) && path[i] != '/' {
			continue
		}

		seg := path[start:i]
		start = i + 1
		if seg == "" {
			continue
		}

		if segment > 0 {
			dst = append(dst, '/')
		}

		switch {
		case segment == 0:
			top = seg
			dst = append(dst, seg...)
			hasMajorParam = top == "channels" || top == "guilds" || top == "webhooks"
			if hasMajorParam {
				majorStart = len(dst) - len(seg)
				majorEnd = len(dst)
			}
		case segment == 1 && hasMajorParam && isASCIIDigits(seg):
			dst = append(dst, seg...)
			majorEnd = len(dst)
		case segment == 2 && (top == "interactions" || top == "webhooks"):
			dst = append(dst, "{token}"...)
		case prevSeg == "reactions":
			dst = append(dst, "{emoji}"...)
		case isASCIIDigits(seg):
			dst = append(dst, "{id}"...)
		default:
			dst = append(dst, seg...)
		}

		prevSeg = seg
		segment++
	}

	if !hasMajorParam {
		majorStart = routePrefixStart
		majorEnd = len(dst)
	}

	return dst, majorStart, majorEnd
}

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}

func nextPowerOfTwo(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func fnv1aBytes(b []byte) uint64 {
	const (
		offset = 14695981039346656037
		prime  = 1099511628211
	)

	h := uint64(offset)
	for _, c := range b {
		h ^= uint64(c)
		h *= prime
	}

	return h
}
