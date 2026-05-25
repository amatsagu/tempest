package tempest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Represents a Discord rate limit bucket.
type Bucket struct {
	mu        sync.Mutex
	Remaining int
	Limit     int
	ResetAt   time.Time
}

type RateLimiterOptions struct {
	SweepInterval      time.Duration // By default: 30 minutes
	SweepThreshold     int           // By default: 2500 buckets
	ReactiveThrottling bool          // Whether to disable preemptive throttling. It's more correct but in some edge cases may exhaust endpoint rate limit sooner than it should. Default: true.
}

type RateLimiter struct {
	mu           sync.RWMutex
	buckets      map[string]*Bucket // Bucket ID -> Bucket
	routeMapping map[string]string  // Route (Method:Path) -> Bucket ID

	globalWait *time.Time
	globalMu   sync.RWMutex

	lastSweep      time.Time
	sweepInterval  time.Duration
	sweepThreshold int
	preemptive     bool
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

	return &RateLimiter{
		buckets:        make(map[string]*Bucket),
		routeMapping:   make(map[string]string),
		lastSweep:      time.Now(),
		sweepInterval:  sweepInterval,
		sweepThreshold: sweepThreshold,
		preemptive:     !opt.ReactiveThrottling,
	}
}

func (rl *RateLimiter) Wait(route string) {
	rl.globalMu.RLock()

	if rl.globalWait != nil {
		if time.Now().Before(*rl.globalWait) {
			wait := time.Until(*rl.globalWait)
			rl.globalMu.RUnlock()
			time.Sleep(wait)
		} else {
			rl.globalMu.RUnlock()
			rl.globalMu.Lock()
			rl.globalWait = nil
			rl.globalMu.Unlock()
		}
	} else {
		rl.globalMu.RUnlock()
	}

	rl.mu.RLock()
	bucketID, ok := rl.routeMapping[route]
	if !ok {
		rl.mu.RUnlock()
		return
	}

	bucket, ok := rl.buckets[bucketID]
	rl.mu.RUnlock()

	if !ok {
		return
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if bucket.Remaining <= 0 {
		waitDuration := time.Until(bucket.ResetAt)
		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}
	}

	if rl.preemptive {
		bucket.Remaining--
	}
}

func (rl *RateLimiter) Update(route string, headers http.Header) {
	if headers.Get("X-RateLimit-Global") == "true" {
		retryAfter, _ := strconv.ParseFloat(headers.Get("Retry-After"), 64)
		resetAt := time.Now().Add(time.Duration(retryAfter * float64(time.Second)))

		rl.globalMu.Lock()
		rl.globalWait = &resetAt
		rl.globalMu.Unlock()
		return
	}

	bucketID := headers.Get("X-RateLimit-Bucket")
	if bucketID == "" {
		return
	}

	remainingStr := headers.Get("X-RateLimit-Remaining")
	limitStr := headers.Get("X-RateLimit-Limit")
	resetAfterStr := headers.Get("X-RateLimit-Reset-After")

	remaining, _ := strconv.Atoi(remainingStr)
	limit, _ := strconv.Atoi(limitStr)
	resetAfter, _ := strconv.ParseFloat(resetAfterStr, 64)

	rl.mu.Lock()
	if time.Since(rl.lastSweep) > rl.sweepInterval {
		rl.lastSweep = time.Now()
		if len(rl.routeMapping) > rl.sweepThreshold {
			clear(rl.routeMapping)
		}

		now := time.Now()
		for id, b := range rl.buckets {
			b.mu.Lock()
			if now.After(b.ResetAt) {
				delete(rl.buckets, id)
			}
			b.mu.Unlock()
		}
	}

	rl.routeMapping[route] = bucketID
	bucket, ok := rl.buckets[bucketID]
	if !ok {
		bucket = &Bucket{}
		rl.buckets[bucketID] = bucket
	}

	rl.mu.Unlock()

	bucket.mu.Lock()
	bucket.Remaining = remaining
	bucket.Limit = limit
	bucket.ResetAt = time.Now().Add(time.Duration(resetAfter*float64(time.Second)) + 100*time.Millisecond)
	bucket.mu.Unlock()
}

type rateLimitTransport struct {
	limiter        *RateLimiter
	innerTransport http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	route := fmt.Sprintf("%s:%s", req.Method, extractRoute(req.URL.Path))
	t.limiter.Wait(route)

	resp, err := t.innerTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	t.limiter.Update(route, resp.Header)
	return resp, nil
}

func extractRoute(path string) string {
	if strings.HasPrefix(path, "/api/v") {
		idx := strings.Index(path[6:], "/")
		if idx != -1 {
			path = path[6+idx:]
		}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return path
	}

	// Major parameters: /channels/{id}, /guilds/{id}, /webhooks/{id}
	if parts[0] == "guilds" || parts[0] == "channels" || parts[0] == "webhooks" {
		// Identify the resource after the ID to group buckets (e.g., /guilds/{id}/roles)
		if len(parts) >= 3 {
			return "/" + parts[0] + "/" + parts[1] + "/" + parts[2]
		}
		return "/" + parts[0] + "/" + parts[1]
	}

	return path
}
