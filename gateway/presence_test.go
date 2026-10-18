package gateway

import (
	json "encoding/json/v2"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestPresenceFor covers resolution order: the status function wins, a nil
// result falls back to the configured presence, and a nil function is an error.
func TestPresenceFor(t *testing.T) {
	configured := &Presence{Status: StatusOnline, Activities: []Activity{{Name: "x", Type: ActivityCustom}}}
	g := New(Config{Presence: configured})

	if got := g.presenceFor(0); got != configured {
		t.Fatalf("presenceFor(0) = %+v, want the configured presence", got)
	}

	override := &Presence{Status: StatusIdle}
	if err := g.UpdateStatus(func(int) *Presence { return override }); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if got := g.presenceFor(0); got != override {
		t.Fatalf("presenceFor(0) = %+v, want the status function's presence", got)
	}

	if err := g.UpdateStatus(func(int) *Presence { return nil }); err != nil {
		t.Fatalf("UpdateStatus(nil result): %v", err)
	}

	if got := g.presenceFor(0); got != configured {
		t.Fatalf("presenceFor(0) with a nil function result = %+v, want the configured fallback", got)
	}

	if err := g.UpdateStatus(nil); err == nil {
		t.Fatal("UpdateStatus(nil) = nil error, want an error")
	}
}

// TestUpdateStatusSendsPresenceUpdate connects a shard to a fake gateway and
// proves UpdateStatus writes an op-3 frame carrying the new presence.
func TestUpdateStatusSendsPresenceUpdate(t *testing.T) {
	presenceCh := make(chan []byte, 1)

	endpoint := startFakeGateway(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`))

		for {
			op, data, err := readOp(conn)
			if err != nil {
				return
			}

			switch op {
			case OpIdentify:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`))
			case OpPresenceUpdate:
				select {
				case presenceCh <- data:
				default:
				}
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

	waitFor(t, func() bool {
		g.mu.Lock()
		defer g.mu.Unlock()
		return len(g.shards) == 1
	})

	waitFor(t, func() bool {
		col.mu.Lock()
		defer col.mu.Unlock()
		return col.ready
	})

	presence := &Presence{Status: StatusIdle, Activities: []Activity{{Name: "x", Type: ActivityCustom}}}
	if err := g.UpdateStatus(func(int) *Presence { return presence }); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	select {
	case data := <-presenceCh:
		var frame struct {
			D  Presence `json:"d"`
			Op int      `json:"op"`
		}

		if err := json.Unmarshal(data, &frame); err != nil {
			t.Fatalf("presence frame: %v", err)
		}

		if frame.Op != int(OpPresenceUpdate) {
			t.Fatalf("op = %d, want %d (PRESENCE_UPDATE)", frame.Op, OpPresenceUpdate)
		}

		if frame.D.Status != StatusIdle || len(frame.D.Activities) != 1 || frame.D.Activities[0].Name != "x" {
			t.Fatalf("presence = %+v, want idle with activity x", frame.D)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no PRESENCE_UPDATE frame arrived")
	}
}
