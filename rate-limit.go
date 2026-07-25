package tempest

import (
	"fmt"
	"io"
	"log"
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
	Remaining int
	Limit     int
	mu        sync.Mutex
}

type RateLimiterOptions struct {
	TraceLogger    *log.Logger
	SweepInterval  time.Duration // By default: 30 minutes
	SweepThreshold int           // By default: 2500 buckets
	Trace          bool
}

type RateLimiter struct {
	lastSweep      time.Time
	buckets        map[string]*Bucket // Bucket ID -> Bucket
	routeMapping   map[string]string  // Route (Method:Path) -> Bucket ID
	traceLogger    *log.Logger
	globalWait     atomic.Int64
	sweepInterval  time.Duration
	sweepThreshold int
	mu             sync.RWMutex
	trace          bool
}

func NewRateLimiter(opt RateLimiterOptions) *RateLimiter {
	sweepInterval := opt.SweepInterval
	if sweepInterval == 0 {
		sweepInterval = 30 * time.Minute
	}

	sweepThreshold := opt.SweepThreshold
	if sweepThreshold == 0 {
		sweepThreshold = 2500
	}

	trace := opt.Trace
	if !trace && opt.TraceLogger != nil {
		trace = opt.TraceLogger.Writer() != io.Discard
	}

	return &RateLimiter{
		buckets:        make(map[string]*Bucket),
		routeMapping:   make(map[string]string),
		lastSweep:      time.Now(),
		sweepInterval:  sweepInterval,
		sweepThreshold: sweepThreshold,
		traceLogger:    opt.TraceLogger,
		trace:          trace,
	}
}

func (rl *RateLimiter) tracef(format string, v ...any) {
	if !rl.trace {
		return
	}

	if rl.trace && rl.traceLogger != nil {
		rl.traceLogger.Printf("[(REST) LIMITER] "+format, v...)
	}
}

func (rl *RateLimiter) Wait(route string) func(headers http.Header) {
	globalWaitNano := rl.globalWait.Load()
	if globalWaitNano != 0 {
		globalWait := time.Unix(0, globalWaitNano)
		if time.Now().Before(globalWait) {
			wait := time.Until(globalWait)
			rl.tracef("Global rate limit hit! Waiting %s...", wait.Round(time.Millisecond))
			time.Sleep(wait)
		}
	}

	rl.mu.Lock()
	if time.Since(rl.lastSweep) > rl.sweepInterval {
		rl.lastSweep = time.Now()
		if len(rl.routeMapping) > rl.sweepThreshold {
			clear(rl.routeMapping)
		}

		now := time.Now()
		for id, b := range rl.buckets {
			if b.mu.TryLock() {
				if now.After(b.ResetAt) {
					delete(rl.buckets, id)
				}
				b.mu.Unlock()
			}
		}
	}

	bucketID, ok := rl.routeMapping[route]
	if !ok {
		bucketID = route
		rl.routeMapping[route] = bucketID
	}

	bucket, ok := rl.buckets[bucketID]
	if !ok {
		bucket = &Bucket{
			ID:        bucketID,
			Limit:     1,
			Remaining: 1,
		}
		rl.buckets[bucketID] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()

	if bucket.Remaining <= 0 {
		if time.Now().Before(bucket.ResetAt) {
			waitDuration := time.Until(bucket.ResetAt)
			bucket.mu.Unlock()
			rl.tracef("Rate limit hit on route \"%s\" (bucket ID: %s)! Waiting %s...", route, bucket.ID, waitDuration.Round(time.Millisecond))
			time.Sleep(waitDuration)
			return rl.Wait(route)
		}
	}

	bucket.Remaining--
	bucket.mu.Unlock()

	return func(headers http.Header) {
		if headers == nil {
			return
		}

		bucket.mu.Lock()
		defer bucket.mu.Unlock()

		retryAfterStr := headers.Get("Retry-After")
		if headers.Get("X-RateLimit-Global") == "true" {
			if retryAfterStr != "" {
				retryAfter, _ := strconv.ParseFloat(retryAfterStr, 64)
				rl.globalWait.Store(time.Now().Add(time.Duration(retryAfter*float64(time.Second)) + 250*time.Millisecond).UnixNano())

				rl.tracef("Received global rate limit! Retry after: %f", retryAfter)
			}
			return
		}

		bucketHeader := headers.Get("X-RateLimit-Bucket")
		if bucketHeader == "" {
			return
		}

		rl.mu.Lock()
		if bucket.ID != bucketHeader {
			rl.routeMapping[route] = bucketHeader
			if _, exists := rl.buckets[bucketHeader]; !exists {
				oldID := bucket.ID
				bucket.ID = bucketHeader
				rl.buckets[bucketHeader] = bucket
				delete(rl.buckets, oldID)
			}
		}
		rl.mu.Unlock()

		remainingStr := headers.Get("X-RateLimit-Remaining")
		limitStr := headers.Get("X-RateLimit-Limit")
		resetAfterStr := headers.Get("X-RateLimit-Reset-After")

		if limitStr != "" {
			limit, _ := strconv.Atoi(limitStr)
			bucket.Limit = limit
		}
		if remainingStr != "" {
			remaining, _ := strconv.Atoi(remainingStr)
			bucket.Remaining = remaining
		}
		if resetAfterStr != "" {
			resetAfter, _ := strconv.ParseFloat(resetAfterStr, 64)
			bucket.ResetAt = time.Now().Add(time.Duration(resetAfter*float64(time.Second)) + 100*time.Millisecond)
		}
	}
}

type rateLimitTransport struct {
	limiter        *RateLimiter
	innerTransport http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	route := fmt.Sprintf("%s:%s", req.Method, extractRoute(req.URL.Path))
	unlock := t.limiter.Wait(route)

	resp, err := t.innerTransport.RoundTrip(req)
	if err != nil {
		unlock(nil)
		return nil, err
	}

	unlock(resp.Header)
	return resp, nil
}

func extractRoute(path string) string {
	orig := path
	if strings.HasPrefix(path, "/api/v") {
		idx := strings.Index(path[6:], "/")
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

	idx1 := strings.Index(rest, "/")
	if idx1 == -1 {
		return path
	}

	idx2 := strings.Index(rest[idx1+1:], "/")
	if idx2 == -1 {
		return path
	}

	totalLen := len(major) + idx1 + 1 + idx2
	return path[:totalLen]
}
