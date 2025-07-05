package gateway

import (
	"context"
	"sync"
	"time"
)

// RateLimiter controls rate-limited actions (e.g., identify calls) per bucket.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[int]*bucketQueue
	// identifyInterval is the minimal interval between identify calls per bucket.
	identifyInterval time.Duration
}

type bucketQueue struct {
	waitChans []chan struct{}
	lastSent  time.Time
}

// NewRateLimiter creates a new RateLimiter with given identify interval.
func NewRateLimiter(identifyInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets:          make(map[int]*bucketQueue),
		identifyInterval: identifyInterval,
	}
}

// Wait waits until the shard in bucketID is allowed to proceed with identify.
// Context allows cancellation/timeouts.
func (rl *RateLimiter) Wait(ctx context.Context, bucketID int) error {
	rl.mu.Lock()

	bq, ok := rl.buckets[bucketID]
	if !ok {
		bq = &bucketQueue{}
		rl.buckets[bucketID] = bq
	}

	// If enough time passed since last identify, allow immediately.
	now := time.Now()
	elapsed := now.Sub(bq.lastSent)
	if elapsed >= rl.identifyInterval && len(bq.waitChans) == 0 {
		bq.lastSent = now
		rl.mu.Unlock()
		return nil
	}

	// Otherwise, enqueue and wait for signal.
	waitChan := make(chan struct{})
	bq.waitChans = append(bq.waitChans, waitChan)
	rl.mu.Unlock()

	select {
	case <-ctx.Done():
		// Remove waitChan from queue on cancel
		rl.mu.Lock()
		defer rl.mu.Unlock()
		for i, ch := range bq.waitChans {
			if ch == waitChan {
				bq.waitChans = append(bq.waitChans[:i], bq.waitChans[i+1:]...)
				break
			}
		}
		return ctx.Err()
	case <-waitChan:
		return nil
	}
}

// Done notifies RateLimiter that the shard in bucketID finished identify call,
// so next waiting shard can proceed.
func (rl *RateLimiter) Done(bucketID int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bq, ok := rl.buckets[bucketID]
	if !ok {
		return
	}

	now := time.Now()
	bq.lastSent = now

	if len(bq.waitChans) == 0 {
		return
	}

	// Pop the first waiting shard and notify it to proceed.
	next := bq.waitChans[0]
	bq.waitChans = bq.waitChans[1:]

	// We want to delay the notify so that shards are spaced out by identifyInterval.
	delay := rl.identifyInterval
	time.AfterFunc(delay, func() {
		close(next)
	})
}
