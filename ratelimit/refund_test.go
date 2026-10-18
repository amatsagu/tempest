package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// TestWaitRefundsGlobalOnCancel proves a request that never goes out, because its
// bucket wait was cancelled, does not spend process-wide budget.
func TestWaitRefundsGlobalOnCancel(t *testing.T) {
	l := New(Options{GlobalRate: 1})

	header := http.Header{}
	header.Set("X-RateLimit-Bucket", "b")
	header.Set("X-RateLimit-Remaining", "0")
	header.Set("X-RateLimit-Reset-After", "3600")

	key := RouteKey("POST", "/channels/1/messages")
	l.Update(key, header)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := l.Wait(ctx, key); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait = %v, want context.Canceled", err)
	}

	l.global.mu.Lock()
	tokens := l.global.tokens
	l.global.mu.Unlock()

	if tokens != 1 {
		t.Fatalf("global tokens = %v, want 1 after the reserved token was refunded", tokens)
	}
}
