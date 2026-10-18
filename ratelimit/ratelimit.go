// Package ratelimit is a low-overhead, sharded, header-driven rate limiter for HTTP
// APIs such as Discord's REST. It learns per-bucket state, enforces a process-wide
// ceiling, and evicts idle buckets opportunistically during Update.
package ratelimit

import (
	"context"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Canonicalized header names, computed once so lookups never allocate.
var (
	headerBucket     = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Bucket")
	headerLimit      = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Limit")
	headerRemaining  = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Remaining")
	headerResetAfter = textproto.CanonicalMIMEHeaderKey("X-RateLimit-Reset-After")
)

func headerValue(h http.Header, key string) string {
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Options tunes a Limiter. Zero values are replaced with sane defaults.
type Options struct {
	// Now overrides the clock (used in tests).
	Now func() time.Time
	// GlobalRate is the process-wide requests-per-second ceiling (Discord: 50).
	GlobalRate float64
	// Shards is the number of lock shards; rounded up to a power of two.
	Shards int
	// IdleTTL evicts a bucket untouched for this long; negative disables eviction.
	IdleTTL time.Duration
	// SweepInterval is the number of Updates per shard between amortized sweeps.
	SweepInterval int
	// MaxRoutes bounds the route-to-bucket cache; zero uses a default. Paths that
	// carry a unique token never repeat, so this keeps them from growing the cache.
	MaxRoutes int
	// MaxBuckets bounds the learned bucket state; zero uses a default. It is the
	// hard memory ceiling: past it, unseen routes are not tracked and simply rely on
	// the global ceiling until the sweep frees room.
	MaxBuckets int
}

const (
	defaultGlobalRate    = 50
	defaultShards        = 32
	defaultIdleTTL       = 10 * time.Minute
	defaultSweepInterval = 512

	// defaultMaxRoutes bounds the route-to-bucket cache. Paths that carry a unique
	// token (interaction callbacks, webhook executions) never repeat, so caching
	// them would grow without bound; the cache stops taking new keys at the cap and
	// the sweep evicts idle ones.
	defaultMaxRoutes = 4096

	// defaultMaxBuckets bounds the learned bucket state for the same reason.
	defaultMaxBuckets = 4096

	// resetFudge pads the learned window: Discord's reset times are approximate.
	resetFudge = 100 * time.Millisecond

	// retryFloor is the shortest wait when a bucket is exhausted but its window is
	// unknown, so a limiter can never spin at a few milliseconds per attempt.
	retryFloor = 250 * time.Millisecond
)

// Limiter coordinates a global ceiling plus per-route buckets.
type Limiter struct {
	now           func() time.Time
	global        *globalBucket
	routes        sync.Map // route key -> routeEntry
	shards        []shard
	idleTTL       time.Duration
	sweepInterval int
	maxRoutes     int64
	maxBuckets    int64
	mask          uint64
	routeCount    atomic.Int64
	bucketCount   atomic.Int64
}

type shard struct {
	buckets map[string]*bucket
	ops     int
	mu      sync.Mutex
}

// bucket is the learned state for one Discord rate-limit bucket (Unix nanoseconds).
// window and limit are learned from headers so an elapsed window can be reopened
// with the real allowance instead of letting every waiter fire at once.
type bucket struct {
	resetAt   int64
	lastUsed  int64
	window    int64
	remaining int
	limit     int
}

// routeEntry maps a route to its Discord bucket. It is published once and never
// mutated afterwards, so a reader needs no lock; a hot route only bumps the atomic
// lastUsed. bucket is precomputed so a refresh allocates nothing and never races.
type routeEntry struct {
	bucket   string
	id       string
	lastUsed atomic.Int64
}

// New builds a Limiter.
func New(opts Options) *Limiter {
	rate := opts.GlobalRate
	if rate <= 0 {
		rate = defaultGlobalRate
	}

	shards := opts.Shards
	if shards <= 0 {
		shards = defaultShards
	}
	shards = nextPowerOfTwo(shards)

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	idleTTL := opts.IdleTTL
	if idleTTL == 0 {
		idleTTL = defaultIdleTTL
	}

	sweepInterval := opts.SweepInterval
	if sweepInterval <= 0 {
		sweepInterval = defaultSweepInterval
	}

	maxRoutes := opts.MaxRoutes
	if maxRoutes <= 0 {
		maxRoutes = defaultMaxRoutes
	}

	maxBuckets := opts.MaxBuckets
	if maxBuckets <= 0 {
		maxBuckets = defaultMaxBuckets
	}

	l := &Limiter{
		now: now,
		global: &globalBucket{
			tokens:   rate,
			capacity: rate,
			rate:     rate,
			last:     now().UnixNano(),
		},
		shards:        make([]shard, shards),
		idleTTL:       idleTTL,
		sweepInterval: sweepInterval,
		maxRoutes:     int64(maxRoutes),
		maxBuckets:    int64(maxBuckets),
		mask:          uint64(shards - 1),
	}

	for i := range l.shards {
		l.shards[i].buckets = make(map[string]*bucket)
	}

	return l
}

// Pause suspends all requests for d (used on a global 429).
func (l *Limiter) Pause(d time.Duration) {
	if d <= 0 {
		return
	}
	l.global.pause(l.now().UnixNano() + d.Nanoseconds())
}

// Wait blocks until the route may proceed, honoring the global ceiling and its
// bucket. The global token is refunded if the bucket wait fails, so a cancelled
// request cannot spend the process-wide budget.
func (l *Limiter) Wait(ctx context.Context, key string) error {
	for {
		wait, ok := l.global.reserve(l.now().UnixNano())
		if ok {
			break
		}

		if err := sleep(ctx, wait); err != nil {
			return err
		}
	}

	if err := l.waitRoute(ctx, key); err != nil {
		l.global.refund()
		return err
	}
	return nil
}

// WaitRoute blocks until the route may proceed, honoring its bucket but not the
// global ceiling. Discord exempts interaction responses from the global limit, and
// a first response must go out within three seconds, so a global pause must not hold it.
func (l *Limiter) WaitRoute(ctx context.Context, key string) error {
	return l.waitRoute(ctx, key)
}

func (l *Limiter) waitRoute(ctx context.Context, key string) error {
	bucketKey := l.bucketKey(key)
	sh := l.shard(bucketKey)

	for {
		now := l.now().UnixNano()

		sh.mu.Lock()
		b := sh.buckets[bucketKey]
		if b == nil {
			// Unknown bucket: proceed optimistically; the first response teaches us.
			sh.mu.Unlock()
			return nil
		}

		b.lastUsed = now

		if now >= b.resetAt {
			// The window elapsed. Reopen it with the learned allowance and length, so
			// waiters do not all wake and fire at once and the state corrects itself
			// from the next response rather than from a burst.
			b.remaining = b.limit
			if b.remaining < 1 {
				b.remaining = 1
			}

			if b.window > 0 {
				b.resetAt = now + b.window
			} else {
				b.resetAt = now + retryFloor.Nanoseconds()
			}
		}

		if b.remaining > 0 {
			b.remaining--
			sh.mu.Unlock()
			return nil
		}

		wait := time.Duration(b.resetAt - now)
		sh.mu.Unlock()

		if wait < retryFloor {
			wait = retryFloor
		}

		if err := sleep(ctx, wait); err != nil {
			return err
		}
	}
}

// Update records rate-limit state learned from response headers for a route key,
// with an amortized eviction of idle buckets.
func (l *Limiter) Update(key string, header http.Header) {
	now := l.now().UnixNano()

	bucketKey := key
	if id := headerValue(header, headerBucket); id != "" {
		// Discord's bucket hash is not inclusive of the top-level resource, so two
		// channels share it; combine it with the normalized route, which carries the
		// major parameter, to keep their limits separate.
		bucketKey = l.rememberRoute(key, id, now)
	}

	remainingStr := headerValue(header, headerRemaining)
	if remainingStr == "" {
		return
	}

	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return
	}

	// Both are parsed only when present: strconv allocates a *NumError on failure,
	// which must not happen on the hot path.
	limit := 0
	if v := headerValue(header, headerLimit); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	var window int64
	if v := headerValue(header, headerResetAfter); v != "" {
		if secs, err := strconv.ParseFloat(v, 64); err == nil && secs > 0 {
			window = int64(secs*float64(time.Second)) + resetFudge.Nanoseconds()
		}
	}

	sh := l.shard(bucketKey)

	sh.mu.Lock()
	b := sh.buckets[bucketKey]
	if b == nil {
		if l.bucketCount.Load() >= l.maxBuckets {
			// Memory ceiling reached: leave this route to the global ceiling until the
			// sweep frees room, instead of growing without bound.
			sh.mu.Unlock()
			return
		}

		b = &bucket{}
		sh.buckets[bucketKey] = b
		l.bucketCount.Add(1)
	}

	if limit > 0 {
		b.limit = limit
	} else if remaining > b.limit {
		b.limit = remaining
	}

	if window > b.window {
		b.window = window
	}

	b.remaining = remaining
	if window > 0 {
		b.resetAt = now + window
	}
	b.lastUsed = now

	sh.ops++
	sweep := l.idleTTL > 0 && sh.ops >= l.sweepInterval
	if sweep {
		sh.ops = 0
	}
	sh.mu.Unlock()

	// The amortized sweep covers every shard and the route cache, so idle state is
	// reclaimed even if nobody calls Sweep, and the caps cannot become a permanent
	// ceiling after they are reached once.
	if sweep {
		l.sweepIdle(now, l.idleTTL.Nanoseconds())
	}
}

// Sweep removes buckets and route entries idle for idleAfter (or IdleTTL when zero)
// and returns how many. The limiter also runs this itself from Update on an
// amortized schedule, so calling it is only needed to force it early.
func (l *Limiter) Sweep(idleAfter time.Duration) int {
	if idleAfter <= 0 {
		idleAfter = l.idleTTL
	}

	if idleAfter <= 0 {
		return 0
	}
	return l.sweepIdle(l.now().UnixNano(), idleAfter.Nanoseconds())
}

func (l *Limiter) sweepIdle(now, idle int64) int {
	bucketsRemoved := 0
	for i := range l.shards {
		bucketsRemoved += l.shards[i].sweep(now, idle)
	}

	if bucketsRemoved > 0 {
		l.bucketCount.Add(-int64(bucketsRemoved))
	}

	removed := bucketsRemoved
	l.routes.Range(func(k, v any) bool {
		entry, ok := v.(*routeEntry)
		if ok && now-entry.lastUsed.Load() > idle {
			if l.routes.CompareAndDelete(k, entry) {
				l.routeCount.Add(-1)
				removed++

				// A route key maps to exactly one bucket, so retire both together;
				// otherwise a re-learned route would briefly shadow stale state.
				l.removeBucket(entry.bucket)
			}
		}
		return true
	})

	return removed
}

// removeBucket drops the learned state for a bucket key, if any.
func (l *Limiter) removeBucket(key string) {
	sh := l.shard(key)

	sh.mu.Lock()
	if _, ok := sh.buckets[key]; ok {
		delete(sh.buckets, key)
		l.bucketCount.Add(-1)
	}
	sh.mu.Unlock()
}

// rememberRoute returns the bucket key for a route, caching the mapping. The
// combined key is built once per route, so a hot route allocates nothing.
func (l *Limiter) rememberRoute(key, id string, now int64) string {
	if v, ok := l.routes.Load(key); ok {
		entry := v.(*routeEntry)
		if entry.id == id {
			entry.lastUsed.Store(now)
			return entry.bucket
		}

		// Bucket changed: replace the entry rather than mutating the published one.
		bucket := id + ":" + key
		l.routes.Store(key, newRouteEntry(id, bucket, now))
		return bucket
	}

	bucket := id + ":" + key
	if l.routeCount.Load() >= l.maxRoutes {
		return bucket
	}

	if _, loaded := l.routes.LoadOrStore(key, newRouteEntry(id, bucket, now)); !loaded {
		l.routeCount.Add(1)
	}
	return bucket
}

func newRouteEntry(id, bucket string, now int64) *routeEntry {
	entry := &routeEntry{bucket: bucket, id: id}
	entry.lastUsed.Store(now)
	return entry
}

// Size reports the number of tracked buckets and route mappings.
func (l *Limiter) Size() (buckets, routes int) {
	for i := range l.shards {
		l.shards[i].mu.Lock()
		buckets += len(l.shards[i].buckets)
		l.shards[i].mu.Unlock()
	}

	l.routes.Range(func(_, _ any) bool {
		routes++
		return true
	})

	return buckets, routes
}

func (s *shard) sweep(now, idle int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for key, b := range s.buckets {
		if now-b.lastUsed > idle {
			delete(s.buckets, key)
			removed++
		}
	}
	return removed
}

func (l *Limiter) bucketKey(key string) string {
	if v, ok := l.routes.Load(key); ok {
		if entry, ok := v.(*routeEntry); ok {
			return entry.bucket
		}
	}
	return key
}

func (l *Limiter) shard(key string) *shard {
	return &l.shards[fnv1a(key)&l.mask]
}

// RouteKey normalizes a method and path into a stable key. Numeric IDs become
// {id}, except the major parameter, the guild, channel or webhook ID that Discord
// rate limits separately; that is kept verbatim so two channels do not share state.
// Interaction and webhook tokens are collapsed because they never repeat.
// https://docs.discord.com/developers/topics/rate-limits
func RouteKey(method, path string) string {
	// Discord appends query parameters to some routes; they do not change the
	// bucket, so drop them.
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}

	var sb strings.Builder
	sb.Grow(len(method) + len(path) + 1)
	sb.WriteString(method)
	sb.WriteByte(' ')

	if strings.HasPrefix(path, "/") {
		sb.WriteByte('/')
	}

	top := ""
	segment := 0
	start := 0

	for i := 0; i <= len(path); i++ {
		if i != len(path) && path[i] != '/' {
			continue
		}

		seg := path[start:i]
		start = i + 1
		if seg == "" {
			continue
		}

		switch {
		case segment == 0:
			top = seg
			sb.WriteString(seg)
		case segment == 1 && isMajorResource(top) && isNumeric(seg):
			sb.WriteString(seg)
		case segment == 2 && (top == "interactions" || top == "webhooks"):
			sb.WriteString("{token}")
		case isNumeric(seg):
			sb.WriteString("{id}")
		default:
			sb.WriteString(seg)
		}

		segment++
		if i < len(path) {
			sb.WriteByte('/')
		}
	}

	return sb.String()
}

// isMajorResource reports whether a top-level path segment is one Discord rate
// limits per resource.
func isMajorResource(segment string) bool {
	return segment == "channels" || segment == "guilds" || segment == "webhooks"
}

func isNumeric(s string) bool {
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

// globalBucket is a token bucket for the process-wide ceiling (Unix nanoseconds).
type globalBucket struct {
	tokens      float64
	capacity    float64
	rate        float64
	last        int64
	pausedUntil int64
	mu          sync.Mutex
}

// reserve consumes one token, returning false and a wait if none is available.
func (g *globalBucket) reserve(now int64) (time.Duration, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if now < g.pausedUntil {
		return time.Duration(g.pausedUntil - now), false
	}

	if elapsed := now - g.last; elapsed > 0 {
		g.tokens += float64(elapsed) / float64(time.Second) * g.rate
		if g.tokens > g.capacity {
			g.tokens = g.capacity
		}
		g.last = now
	}

	if g.tokens >= 1 {
		g.tokens--
		return 0, true
	}

	needed := (1 - g.tokens) / g.rate
	return time.Duration(needed * float64(time.Second)), false
}

func (g *globalBucket) pause(until int64) {
	g.mu.Lock()
	if until > g.pausedUntil {
		g.pausedUntil = until
	}
	g.mu.Unlock()
}

// refund returns a token that was reserved but not used, never exceeding capacity.
func (g *globalBucket) refund() {
	g.mu.Lock()
	if g.tokens < g.capacity {
		g.tokens++
	}
	g.mu.Unlock()
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func nextPowerOfTwo(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// fnv1a hashes a key into a shard index. Cheap and dependency-free.
func fnv1a(s string) uint64 {
	const (
		offset = 14695981039346656037
		prime  = 1099511628211
	)
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}
