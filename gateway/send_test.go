package gateway

import (
	json "encoding/json/v2"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestSendWritesRawPayload proves Send writes an opcode the library does not wrap,
// which is the hook a voice or member-request feature needs.
func TestSendWritesRawPayload(t *testing.T) {
	frames := make(chan []byte, 1)

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
			case OpRequestGuildMembers:
				select {
				case frames <- data:
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
		col.mu.Lock()
		defer col.mu.Unlock()
		return col.ready
	})

	if err := g.Send(0, Outbound{Op: OpRequestGuildMembers, Data: map[string]any{"guild_id": "1"}}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case data := <-frames:
		var frame struct {
			D  map[string]string `json:"d"`
			Op int               `json:"op"`
		}

		if err := json.Unmarshal(data, &frame); err != nil {
			t.Fatalf("frame: %v", err)
		}

		if frame.Op != int(OpRequestGuildMembers) {
			t.Fatalf("op = %d, want %d (REQUEST_GUILD_MEMBERS)", frame.Op, OpRequestGuildMembers)
		}

		if frame.D["guild_id"] != "1" {
			t.Fatalf("payload = %v, want guild_id 1", frame.D)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no REQUEST_GUILD_MEMBERS frame arrived")
	}

	if err := g.Send(9, Outbound{Op: OpHeartbeat}); err == nil {
		t.Fatal("Send to a missing shard = nil error, want an error")
	}
}
