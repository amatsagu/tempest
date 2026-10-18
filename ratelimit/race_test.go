package ratelimit_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/amatsagu/tempest/ratelimit"
)

// TestRouteKeyStripsQuery proves query parameters do not fragment a route key.
func TestRouteKeyStripsQuery(t *testing.T) {
	got := ratelimit.RouteKey("GET", "/guilds/123/members?limit=100&after=456")
	want := "GET /guilds/123/members"
	if got != want {
		t.Fatalf("RouteKey = %q, want %q", got, want)
	}
}

// TestRouteKeyCollapsesTokens proves interaction and webhook tokens, which never
// repeat, do not each get their own route.
func TestRouteKeyCollapsesTokens(t *testing.T) {
	callback := ratelimit.RouteKey("POST", "/interactions/1/aaa/callback")
	if other := ratelimit.RouteKey("POST", "/interactions/2/bbb/callback"); callback != other {
		t.Fatalf("interaction callbacks differ: %q vs %q", callback, other)
	}

	webhook := ratelimit.RouteKey("POST", "/webhooks/1/aaa")
	if other := ratelimit.RouteKey("POST", "/webhooks/1/bbb"); webhook != other {
		t.Fatalf("webhook tokens differ: %q vs %q", webhook, other)
	}

	if webhook == ratelimit.RouteKey("POST", "/webhooks/2/aaa") {
		t.Fatal("different webhooks collapsed to one key")
	}
}

// TestConcurrentUpdateAndWait exercises the route cache from two goroutines, the
// way the REST client does; run under -race it catches unsafe sharing.
func TestConcurrentUpdateAndWait(t *testing.T) {
	l := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	key := ratelimit.RouteKey("POST", "/channels/1/messages")

	header := http.Header{}
	header.Set("X-RateLimit-Bucket", "bucket-a")
	header.Set("X-RateLimit-Remaining", "1000000")
	header.Set("X-RateLimit-Reset-After", "1")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			l.Update(key, header)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			if err := l.Wait(context.Background(), key); err != nil {
				t.Errorf("Wait: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}

// TestRouteCacheIsBounded proves a flood of unique routes cannot grow the cache
// past the cap, and that sweeping frees room again.
func TestRouteCacheIsBounded(t *testing.T) {
	const limit = 8

	now := time.Unix(0, 0)
	l := ratelimit.New(ratelimit.Options{
		GlobalRate: 1e9,
		MaxRoutes:  limit,
		IdleTTL:    time.Minute,
		Now:        func() time.Time { return now },
	})

	for i := 0; i < 100; i++ {
		key := ratelimit.RouteKey("POST", fmt.Sprintf("/channels/%d/messages", i))
		l.Update(key, limitedHeader(fmt.Sprintf("b%d", i)))
	}

	if _, routes := l.Size(); routes > limit {
		t.Fatalf("routes = %d, want at most %d", routes, limit)
	}

	now = now.Add(2 * time.Minute)
	l.Sweep(0)

	if _, routes := l.Size(); routes != 0 {
		t.Fatalf("routes after sweep = %d, want 0", routes)
	}

	l.Update(ratelimit.RouteKey("POST", "/channels/1/messages"), limitedHeader("fresh"))
	if _, routes := l.Size(); routes != 1 {
		t.Fatalf("routes after refill = %d, want 1", routes)
	}
}
