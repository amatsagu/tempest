package ratelimit_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/amatsagu/tempest/ratelimit"
)

func TestRouteKey(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/channels/123/messages/456", "GET /channels/123/messages/{id}"},
		{"GET", "/guilds/456/members/789", "GET /guilds/456/members/{id}"},
		{"POST", "/webhooks/1/tok123", "POST /webhooks/1/{token}"},
		{"POST", "/interactions/1/tok123/callback", "POST /interactions/{id}/{token}/callback"},
		{"GET", "/users/789", "GET /users/{id}"},
		{"GET", "/guilds/1/members?limit=100", "GET /guilds/1/members"},
	}

	for _, tc := range cases {
		if got := ratelimit.RouteKey(tc.method, tc.path); got != tc.want {
			t.Errorf("RouteKey(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

// TestMajorParameterIsolated proves Discord's bucket hash (which excludes the
// top-level resource) plus the major parameter keep two channels' limits separate.
func TestMajorParameterIsolated(t *testing.T) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})

	one := ratelimit.RouteKey("POST", "/channels/1/messages")
	two := ratelimit.RouteKey("POST", "/channels/2/messages")
	if one == two {
		t.Fatalf("two channels collapsed to one key: %q", one)
	}

	// Discord reports the same bucket for both channels; only the major parameter
	// tells them apart. Channel 1 is exhausted, channel 2 has room.
	exhausted := http.Header{}
	exhausted.Set("X-RateLimit-Bucket", "shared")
	exhausted.Set("X-RateLimit-Remaining", "0")
	exhausted.Set("X-RateLimit-Reset-After", "3600")

	room := http.Header{}
	room.Set("X-RateLimit-Bucket", "shared")
	room.Set("X-RateLimit-Remaining", "5")
	room.Set("X-RateLimit-Reset-After", "3600")

	l.Update(one, exhausted)
	l.Update(two, room)

	if err := l.Wait(context.Background(), two); err != nil {
		t.Fatalf("Wait on channel 2: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := l.Wait(ctx, one); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait on exhausted channel 1 = %v, want deadline exceeded", err)
	}
}

func TestWaitAllowsWhenRemaining(t *testing.T) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	key := ratelimit.RouteKey("POST", "/channels/1/messages")

	header := http.Header{}
	header.Set("X-RateLimit-Bucket", "bucket-a")
	header.Set("X-RateLimit-Remaining", "5")
	header.Set("X-RateLimit-Reset-After", "1")
	l.Update(key, header)

	if err := l.Wait(context.Background(), key); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

func TestWaitCancelsWhenExhausted(t *testing.T) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	key := ratelimit.RouteKey("POST", "/channels/2/messages")

	header := http.Header{}
	header.Set("X-RateLimit-Bucket", "bucket-b")
	header.Set("X-RateLimit-Remaining", "0")
	header.Set("X-RateLimit-Reset-After", "3600")
	l.Update(key, header)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := l.Wait(ctx, key); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error = %v, want deadline exceeded", err)
	}
}

func TestGlobalPause(t *testing.T) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	l.Pause(40 * time.Millisecond)

	start := time.Now()
	if err := l.Wait(context.Background(), "GET /x"); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("expected global pause to delay, elapsed = %v", elapsed)
	}
}

// TestWaitRouteIgnoresGlobalPause proves WaitRoute skips the global ceiling while
// Wait still honors it.
func TestWaitRouteIgnoresGlobalPause(t *testing.T) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	l.Pause(time.Hour)

	key := "POST /interactions/{id}/tok/callback"

	start := time.Now()
	if err := l.WaitRoute(context.Background(), key); err != nil {
		t.Fatalf("WaitRoute: %v", err)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("WaitRoute was delayed by the global pause: %v", elapsed)
	}

	// The same key through Wait honors the pause, proving the difference.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx, key); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error = %v, want deadline exceeded", err)
	}
}

func limitedHeader(bucket string) http.Header {
	h := http.Header{}
	h.Set("X-RateLimit-Bucket", bucket)
	h.Set("X-RateLimit-Remaining", "5")
	h.Set("X-RateLimit-Reset-After", "1")
	return h
}

func TestSweepEvictsIdle(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(ratelimit.Options{
		GlobalRate: 1e9,
		IdleTTL:    time.Minute,
		Now:        func() time.Time { return now },
	})

	for i := 0; i < 40; i++ {
		l.Update(ratelimit.RouteKey("GET", fmt.Sprintf("/x/a%d", i)), limitedHeader(fmt.Sprintf("b%d", i)))
	}

	buckets, routes := l.Size()
	if buckets == 0 || routes == 0 {
		t.Fatalf("expected tracked entries, got buckets=%d routes=%d", buckets, routes)
	}

	now = now.Add(2 * time.Minute)

	if removed := l.Sweep(0); removed == 0 {
		t.Fatal("expected Sweep to evict idle entries")
	}

	buckets, routes = l.Size()
	if buckets != 0 || routes != 0 {
		t.Fatalf("after sweep buckets=%d routes=%d, want 0/0", buckets, routes)
	}
}

func TestAmortizedSweepShrinksBuckets(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(ratelimit.Options{
		GlobalRate:    1e9,
		Shards:        1,
		IdleTTL:       time.Minute,
		SweepInterval: 4,
		Now:           func() time.Time { return now },
	})

	l.Update("/a", limitedHeader("a"))
	l.Update("/b", limitedHeader("b"))

	now = now.Add(2 * time.Minute)

	// Touching /a enough times triggers the amortized sweep of its shard.
	for i := 0; i < 4; i++ {
		l.Update("/a", limitedHeader("a"))
	}

	buckets, _ := l.Size()
	if buckets != 1 {
		t.Fatalf("buckets = %d, want 1 (only the recently used bucket)", buckets)
	}
}

// TestExpiredWindowKeepsLearnedAllowance proves an elapsed window reopens with the
// bucket's learned allowance, then holds, instead of letting every waiter reset and
// fire at once (which is what produced a burst after a window lapsed).
func TestExpiredWindowKeepsLearnedAllowance(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9, Now: func() time.Time { return now }})

	key := ratelimit.RouteKey("POST", "/channels/1/messages")

	header := http.Header{}
	header.Set("X-RateLimit-Bucket", "learn")
	header.Set("X-RateLimit-Limit", "2")
	header.Set("X-RateLimit-Remaining", "0")
	header.Set("X-RateLimit-Reset-After", "1")
	l.Update(key, header)

	now = now.Add(2 * time.Second)

	if err := l.Wait(context.Background(), key); err != nil {
		t.Fatalf("first Wait: %v", err)
	}

	if err := l.Wait(context.Background(), key); err != nil {
		t.Fatalf("second Wait: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := l.Wait(ctx, key); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("third Wait = %v, want deadline exceeded once the allowance is spent", err)
	}
}

// TestBucketStateIsBounded proves learned bucket state cannot grow without bound.
func TestBucketStateIsBounded(t *testing.T) {
	const limit = 8

	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9, MaxBuckets: limit})

	for i := 0; i < 100; i++ {
		key := ratelimit.RouteKey("POST", fmt.Sprintf("/channels/%d/messages", i))
		l.Update(key, limitedHeader(fmt.Sprintf("b%d", i)))
	}

	if buckets, _ := l.Size(); buckets > limit {
		t.Fatalf("buckets = %d, want at most %d", buckets, limit)
	}
}

// TestAmortizedSweepReclaimsRoutes proves the limiter's own sweep, run from Update
// with nobody calling Sweep, frees the route cache, so MaxRoutes is a bound rather
// than a permanent ceiling once it is reached.
func TestAmortizedSweepReclaimsRoutes(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(ratelimit.Options{
		GlobalRate:    1e9,
		MaxRoutes:     2,
		SweepInterval: 1,
		IdleTTL:       time.Minute,
		Now:           func() time.Time { return now },
	})

	route := func(channel int) string {
		return ratelimit.RouteKey("POST", fmt.Sprintf("/channels/%d/messages", channel))
	}

	l.Update(route(1), limitedHeader("shared"))
	l.Update(route(2), limitedHeader("shared"))

	if _, routes := l.Size(); routes != 2 {
		t.Fatalf("routes = %d, want 2", routes)
	}

	// At the cap a third route cannot be cached.
	l.Update(route(3), limitedHeader("shared"))
	if _, routes := l.Size(); routes != 2 {
		t.Fatalf("at cap routes = %d, want 2", routes)
	}

	// Let the cache go idle; touching one route triggers the amortized sweep.
	now = now.Add(2 * time.Minute)
	l.Update(route(1), limitedHeader("shared"))

	if _, routes := l.Size(); routes != 1 {
		t.Fatalf("after sweep routes = %d, want 1 (idle routes reclaimed)", routes)
	}

	// Room is free again, so a new route is cached.
	l.Update(route(4), limitedHeader("shared"))
	if _, routes := l.Size(); routes != 2 {
		t.Fatalf("after refill routes = %d, want 2", routes)
	}
}

func BenchmarkUpdate(b *testing.B) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	key := ratelimit.RouteKey("POST", "/channels/1/messages")
	header := limitedHeader("b")

	b.ReportAllocs()
	for b.Loop() {
		l.Update(key, header)
	}
}
