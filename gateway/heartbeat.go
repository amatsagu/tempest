package gateway

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type heartbeat struct {
	interval     time.Duration
	lastBeat     time.Time
	acknowledged atomic.Bool

	ticker   *time.Ticker
	stopChan chan struct{}

	mu       sync.Mutex
	shardID  uint16
	sendBeat func(ctx context.Context) error
}

// Start a new heartbeat ticker loop. If already running, it restarts it.
func (hb *heartbeat) start(ctx context.Context, interval time.Duration) {
	hb.mu.Lock()
	hb.stopLocked()
	hb.interval = interval
	hb.acknowledged.Store(true)

	stopChan := make(chan struct{})
	jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
	ticker := time.NewTicker(hb.interval + jitter)

	hb.stopChan = stopChan
	hb.ticker = ticker
	hb.mu.Unlock()

	trace(hb.shardID, "Heartbeat", "Started ticker with interval %s (+%s of jitter)", hb.interval, jitter)

	go func(ctx context.Context, t *time.Ticker, done <-chan struct{}) {
		for {
			select {
			case <-t.C:
				hb.manualBeat(ctx)
			case <-ctx.Done():
				trace(hb.shardID, "Heartbeat", "Context cancelled, stopping ticker.")
				hb.stop()
				return
			case <-stopChan:
				trace(hb.shardID, "Heartbeat", "Manual stop signal received.")
				return
			}
		}
	}(ctx, ticker, stopChan)
}

// Sends a manual heartbeat immediately (used for both tick and DISCORD heartbeat opcodes)
func (hb *heartbeat) manualBeat(ctx context.Context) {
	hb.mu.Lock()

	if !hb.acknowledged.Load() {
		hb.mu.Unlock()
		trace(hb.shardID, "Heartbeat", "Previous beat not acknowledged - connection may be zombified.")
		return
	}

	hb.acknowledged.Store(false)
	hb.lastBeat = time.Now()

	if err := hb.sendBeat(ctx); err != nil {
		trace(hb.shardID, "Heartbeat", "Failed to send heartbeat: %v", err)
	}
	hb.mu.Unlock()
}

// Returns the current latency, based on time since last acknowledged beat
func (hb *heartbeat) latency() time.Duration {
	hb.mu.Lock()
	latency := time.Since(hb.lastBeat) - hb.interval
	hb.mu.Unlock()
	return latency
}

// Stops ticker loop and clears resources
func (hb *heartbeat) stop() {
	hb.mu.Lock()
	hb.stopLocked()
	hb.mu.Unlock()
}

// Hard resets all values of a heartbeat.
func (hb *heartbeat) reset() {
	hb.mu.Lock()
	hb.stopLocked()
	hb.acknowledged.Store(true)
	hb.lastBeat = time.Now()
	hb.mu.Unlock()
}

func (hb *heartbeat) stopLocked() {
	if hb.ticker != nil {
		hb.ticker.Stop()
		hb.ticker = nil
	}
	if hb.stopChan != nil {
		close(hb.stopChan)
		hb.stopChan = nil
	}
	hb.interval = 0
}
