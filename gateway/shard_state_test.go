package gateway

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestShardStateString covers the four wire states plus an out-of-range value.
func TestShardStateString(t *testing.T) {
	cases := []struct {
		want  string
		state ShardState
	}{
		{"offline", ShardOffline},
		{"connecting", ShardConnecting},
		{"authenticating", ShardAuthenticating},
		{"online", ShardOnline},
		{"offline", ShardState(99)},
	}

	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("ShardState(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}

// TestShardInfoFreshShard checks a shard that has not connected reads ShardOffline
// with zero latency, through both Shards() and shard.info().
func TestShardInfoFreshShard(t *testing.T) {
	g := New(Config{})
	g.shards = []*shard{newShard(g, 0)}

	infos := g.Shards()
	if len(infos) != 1 {
		t.Fatalf("Shards() returned %d entries, want 1", len(infos))
	}

	if infos[0].State != ShardOffline {
		t.Errorf("fresh state = %v, want %v", infos[0].State, ShardOffline)
	}

	if infos[0].Latency != 0 {
		t.Errorf("fresh latency = %v, want 0", infos[0].Latency)
	}

	if info := g.shards[0].info(); info.State != ShardOffline || info.Latency != 0 {
		t.Errorf("info() = %+v, want offline with zero latency", info)
	}
}

// TestShardStateOnlineAfterReady connects a shard to a fake gateway and proves it
// is not online once IDENTIFY is sent but only becomes ShardOnline after READY.
func TestShardStateOnlineAfterReady(t *testing.T) {
	identified := make(chan struct{}, 1)
	readyGate := make(chan struct{})

	endpoint := startFakeGateway(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`))

		for {
			op, _, err := readOp(conn)
			if err != nil {
				return
			}

			if op == OpIdentify {
				select {
				case identified <- struct{}{}:
				default:
				}

				<-readyGate
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`))
			}
		}
	})

	col := &collector{}
	g := startGateway(t, Config{
		Token:      "test-token",
		Endpoint:   endpoint,
		ShardCount: 1,
		OnReady:    col.onReady,
	})

	waitFor(t, func() bool { return len(g.Shards()) == 1 })

	select {
	case <-identified:
	case <-time.After(2 * time.Second):
		t.Fatal("no IDENTIFY arrived")
	}

	if s := g.Shards()[0].State; s == ShardOnline {
		t.Fatalf("state = %v after IDENTIFY, want not online before READY", s)
	}

	close(readyGate)

	waitFor(t, func() bool {
		infos := g.Shards()
		return len(infos) == 1 && infos[0].State == ShardOnline
	})

	waitFor(t, func() bool {
		col.mu.Lock()
		defer col.mu.Unlock()
		return col.ready
	})
}
