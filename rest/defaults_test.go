package rest

import (
	"testing"
	"time"
)

// TestDefaultsMatchUpstream pins the timeout, retry count and dial budget to the
// values Tempest uses, since a zero Options now means "use these".
func TestDefaultsMatchUpstream(t *testing.T) {
	c := New("t", Options{})

	if c.http.Timeout != 10*time.Second {
		t.Fatalf("timeout = %v, want 10s", c.http.Timeout)
	}

	if c.maxRetries != 5 {
		t.Fatalf("maxRetries = %d, want 5", c.maxRetries)
	}
}

// TestBackoffGrows proves the retry wait lengthens on each attempt rather than
// hammering a failing endpoint.
func TestBackoffGrows(t *testing.T) {
	if first := backoff(0); first != 250*time.Millisecond {
		t.Fatalf("backoff(0) = %v, want 250ms", first)
	}

	previous := time.Duration(0)
	for attempt := 0; attempt < defaultMaxRetries; attempt++ {
		wait := backoff(attempt)
		if wait <= previous {
			t.Fatalf("backoff(%d) = %v, want more than %v", attempt, wait, previous)
		}
		previous = wait
	}
}
