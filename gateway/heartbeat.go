package gateway

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type HeartbeatController struct {
	interval     time.Duration
	lastBeat     time.Time
	acknowledged atomic.Bool

	ticker   *time.Ticker
	stopChan chan struct{}

	mu       sync.Mutex
	shardID  uint16
	sendBeat func() error
}

func NewHeartbeatController(shardID uint16, sendBeatFunc func() error) *HeartbeatController {
	return &HeartbeatController{
		shardID:  shardID,
		sendBeat: sendBeatFunc,
	}
}

// Start a new heartbeat ticker loop. If already running, it restarts it.
func (hb *HeartbeatController) Start(ctx context.Context, interval time.Duration) {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	hb.stopLocked()

	hb.interval = interval
	hb.acknowledged.Store(true)
	hb.stopChan = make(chan struct{})

	jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
	hb.ticker = time.NewTicker(hb.interval + jitter)

	trace(hb.shardID, "Heartbeat", "Started ticker with interval %s (+%s of jitter)", hb.interval, jitter)

	go func() {
		for {
			select {
			case <-hb.ticker.C:
				hb.ManualBeat()
			case <-ctx.Done():
				trace(hb.shardID, "Heartbeat", "Context cancelled, stopping ticker.")
				hb.Stop()
				return
			case <-hb.stopChan:
				trace(hb.shardID, "Heartbeat", "Manual stop signal received.")
				return
			}
		}
	}()
}

// Sends a manual heartbeat immediately (used for both tick and DISCORD heartbeat opcodes)
func (hb *HeartbeatController) ManualBeat() {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if !hb.acknowledged.Load() {
		trace(hb.shardID, "Heartbeat", "Previous beat not acknowledged - connection may be zombified.")
		return
	}

	hb.acknowledged.Store(false)
	hb.lastBeat = time.Now()

	if err := hb.sendBeat(); err != nil {
		trace(hb.shardID, "Heartbeat", "Failed to send heartbeat: %v", err)
	}
}

// Marks current heartbeat as acknowledged (used when we get a HEARTBEAT_ACK)
func (hb *HeartbeatController) Acknowledge() {
	hb.acknowledged.Store(true)
}

// Returns whether the previous heartbeat was acknowledged. Healthy state = true.
func (hb *HeartbeatController) Healthy() bool {
	return !hb.acknowledged.Load()
}

// Returns the current latency, based on time since last acknowledged beat
func (hb *HeartbeatController) Latency() time.Duration {
	hb.mu.Lock()
	latency := time.Since(hb.lastBeat) - hb.interval
	hb.mu.Unlock()
	return latency
}

// Stops ticker loop and clears resources
func (hb *HeartbeatController) Stop() {
	hb.mu.Lock()
	hb.stopLocked()
	hb.mu.Unlock()
}

func (hb *HeartbeatController) stopLocked() {
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
