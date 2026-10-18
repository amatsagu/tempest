package gateway

import (
	"testing"
	"time"
)

// TestShardLatencyUnknownBeforeMeasurement checks "not measured" before an ACK and
// for an out-of-range shard.
func TestShardLatencyUnknownBeforeMeasurement(t *testing.T) {
	g := New(Config{})
	g.shards = []*shard{newShard(g, 0)}

	if d, ok := g.ShardLatency(0); ok || d != 0 {
		t.Fatalf("ShardLatency(0) = %v, %v; want 0, false before any ACK", d, ok)
	}

	if _, ok := g.ShardLatency(1); ok {
		t.Fatal("ShardLatency(1) should be false for an out-of-range shard")
	}
}

// TestShardLatencyAfterHeartbeatACK checks a round trip is reported after HEARTBEAT_ACK.
func TestShardLatencyAfterHeartbeatACK(t *testing.T) {
	g := New(Config{})
	s := newShard(g, 0)
	g.shards = []*shard{s}

	if _, ok := g.ShardLatency(0); ok {
		t.Fatal("latency reported before any ACK")
	}

	// A heartbeat sent 5ms ago whose acknowledgement just arrived.
	s.heartbeatSent.Store(time.Now().Add(-5 * time.Millisecond).UnixNano())
	s.recordHeartbeatACK()

	d, ok := g.ShardLatency(0)
	if !ok {
		t.Fatal("ShardLatency(0) not measured after an ACK")
	}

	if d < 5*time.Millisecond {
		t.Fatalf("latency = %v, want at least 5ms", d)
	}
}
