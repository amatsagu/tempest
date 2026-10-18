package gateway

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
)

// startFakeGateway starts a websocket server that runs fn on every connection and
// returns its endpoint for Config.
func startFakeGateway(t *testing.T, fn func(conn *websocket.Conn)) string {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		fn(conn)
	}))
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// startGateway runs a Gateway until the test ends, so the test can inspect it.
func startGateway(t *testing.T, cfg Config) *Gateway {
	t.Helper()

	g := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = g.Run(ctx)
		close(done)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("gateway did not stop after cancel")
		}
	})

	return g
}

// zstdStream emits Discord's zstd-stream compression: one context per connection,
// flushed per message, so a frame carries a slice of an endless stream.
type zstdStream struct {
	enc  *zstd.Encoder
	conn *websocket.Conn
	buf  bytes.Buffer
}

func newZstdStream(conn *websocket.Conn) (*zstdStream, error) {
	s := &zstdStream{conn: conn}
	enc, err := zstd.NewWriter(&s.buf)
	if err != nil {
		return nil, err
	}
	s.enc = enc
	return s, nil
}

// encode returns the bytes flushed for payload without sending; the caller frames them.
func (s *zstdStream) encode(payload string) []byte {
	start := s.buf.Len()
	_, _ = s.enc.Write([]byte(payload))
	_ = s.enc.Flush()

	out := make([]byte, s.buf.Len()-start)
	copy(out, s.buf.Bytes()[start:])
	return out
}

func (s *zstdStream) send(payload string) {
	_ = s.conn.WriteMessage(websocket.BinaryMessage, s.encode(payload))
}

func (s *zstdStream) close() { _ = s.enc.Close() }

// readOp reads one inbound gateway message and returns its opcode.
func readOp(conn *websocket.Conn) (Opcode, []byte, error) {
	_, data, err := conn.ReadMessage()
	if err != nil {
		return 0, nil, err
	}

	var envelope struct {
		Op Opcode `json:"op"`
	}

	if err := json.Unmarshal(data, &envelope); err != nil {
		return 0, data, err
	}
	return envelope.Op, data, nil
}

// TestGatewayHandshake drives the real handshake (HELLO, IDENTIFY, READY, dispatch,
// heartbeats) in plain, zstd-stream, and ignored-compression modes.
func TestGatewayHandshake(t *testing.T) {
	cases := []struct {
		name           string
		client         Compression
		serverCompress bool
	}{
		{name: "plain", client: CompressionNone},
		{name: "zstd", client: CompressionZstd, serverCompress: true},
		{name: "zstd server ignores compress", client: CompressionZstd},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu         sync.Mutex
				ops        []Opcode
				identifies int
				identify   identifyData
			)

			endpoint := startFakeGateway(t, func(conn *websocket.Conn) {
				var send func(string)
				if tc.serverCompress {
					stream, err := newZstdStream(conn)
					if err != nil {
						t.Errorf("zstd writer: %v", err)
						return
					}
					defer stream.close()
					send = stream.send
				} else {
					send = func(payload string) {
						_ = conn.WriteMessage(websocket.TextMessage, []byte(payload))
					}
				}

				send(`{"op":10,"d":{"heartbeat_interval":45000}}`)

				for {
					op, data, err := readOp(conn)
					if err != nil {
						return
					}

					mu.Lock()
					ops = append(ops, op)
					if op == OpIdentify {
						identifies++
						var envelope struct {
							D identifyData `json:"d"`
						}

						if err := json.Unmarshal(data, &envelope); err != nil {
							t.Errorf("identify is not valid JSON: %v", err)
						}
						identify = envelope.D
					}
					mu.Unlock()

					switch op {
					case OpIdentify:
						send(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`)
						send(`{"op":0,"s":2,"t":"MESSAGE_CREATE","d":{"id":"1"}}`)
					case OpHeartbeat:
						send(`{"op":11}`)
					}
				}
			})

			col := &collector{}
			startGateway(t, Config{
				Token:       "test-token",
				Endpoint:    endpoint,
				Intents:     uint32(IntentGuilds),
				ShardCount:  1,
				Compression: tc.client,
				OnReady:     col.onReady,
				OnDispatch:  col.dispatch,
			})

			waitFor(t, func() bool { return col.has("MESSAGE_CREATE") })

			col.mu.Lock()
			ready := col.ready
			col.mu.Unlock()
			if !ready {
				t.Error("OnReady was not called")
			}

			mu.Lock()
			defer mu.Unlock()

			if identifies != 1 {
				t.Errorf("identified %d times, want exactly 1; ops=%v", identifies, ops)
			}

			if len(ops) == 0 || ops[0] != OpIdentify {
				t.Errorf("first inbound op = %v, want IDENTIFY (2) first; ops=%v", ops, ops)
			}

			if identify.Token != "test-token" {
				t.Errorf("identify token = %q, want %q", identify.Token, "test-token")
			}

			if identify.Intents != uint32(IntentGuilds) {
				t.Errorf("identify intents = %d, want %d", identify.Intents, IntentGuilds)
			}

			if identify.Properties.OS == "" || identify.Properties.Browser == "" || identify.Properties.Device == "" {
				t.Errorf("identify properties incomplete: %+v", identify.Properties)
			}

			if identify.Shard != [2]int{0, 1} {
				t.Errorf("identify shard = %v, want [0 1]", identify.Shard)
			}
		})
	}
}

// TestHeartbeatNeverPrecedesIdentify pins the ordering invariant: the heartbeat
// loop starts only after IDENTIFY or RESUME has been written.
func TestHeartbeatNeverPrecedesIdentify(t *testing.T) {
	var (
		mu  sync.Mutex
		ops []Opcode
	)

	endpoint := startFakeGateway(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":10,"d":{"heartbeat_interval":1}}`))

		for {
			op, _, err := readOp(conn)
			if err != nil {
				return
			}

			mu.Lock()
			ops = append(ops, op)
			mu.Unlock()

			switch op {
			case OpIdentify:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`))
			case OpHeartbeat:
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":11}`))
			}
		}
	})

	col := &collector{}
	startGateway(t, Config{
		Token:      "test-token",
		Endpoint:   endpoint,
		Intents:    uint32(IntentGuilds),
		ShardCount: 1,
		OnReady:    col.onReady,
	})

	waitFor(t, func() bool {
		col.mu.Lock()
		defer col.mu.Unlock()
		return col.ready
	})

	// Wait for the 1ms heartbeat to fire so the ordering below is exercised.
	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, op := range ops {
			if op == OpHeartbeat {
				return true
			}
		}
		return false
	})

	mu.Lock()
	defer mu.Unlock()

	firstIdentify, firstHeartbeat := -1, -1
	for i, op := range ops {
		if op == OpIdentify && firstIdentify < 0 {
			firstIdentify = i
		}

		if op == OpHeartbeat && firstHeartbeat < 0 {
			firstHeartbeat = i
		}
	}

	if firstIdentify < 0 {
		t.Fatalf("no IDENTIFY received; ops=%v", ops)
	}

	if firstHeartbeat >= 0 && firstHeartbeat < firstIdentify {
		t.Fatalf("heartbeat at %d preceded identify at %d; ops=%v", firstHeartbeat, firstIdentify, ops)
	}
}

// TestZstdStreamSplitAcrossFrames proves the decoder handles one zstd stream whose
// payloads are split across several websocket frames.
func TestZstdStreamSplitAcrossFrames(t *testing.T) {
	endpoint := startFakeGateway(t, func(conn *websocket.Conn) {
		stream, err := newZstdStream(conn)
		if err != nil {
			t.Errorf("zstd writer: %v", err)
			return
		}
		defer stream.close()

		_ = conn.WriteMessage(websocket.BinaryMessage, stream.encode(`{"op":10,"d":{"heartbeat_interval":45000}}`))

		if op, _, err := readOp(conn); err != nil || op != OpIdentify {
			t.Errorf("first op = %d, err = %v; want IDENTIFY", op, err)
			return
		}

		// READY, its compressed bytes split across three frames.
		ready := stream.encode(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`)
		for _, chunk := range [][]byte{ready[:1], ready[1 : len(ready)/2], ready[len(ready)/2:]} {
			_ = conn.WriteMessage(websocket.BinaryMessage, chunk)
		}

		// A dispatch event split across two frames.
		message := stream.encode(`{"op":0,"s":2,"t":"MESSAGE_CREATE","d":{"id":"1"}}`)
		_ = conn.WriteMessage(websocket.BinaryMessage, message[:len(message)/2])
		_ = conn.WriteMessage(websocket.BinaryMessage, message[len(message)/2:])

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	col := &collector{}
	startGateway(t, Config{
		Token:       "test-token",
		Endpoint:    endpoint,
		Intents:     uint32(IntentGuilds),
		ShardCount:  1,
		Compression: CompressionZstd,
		OnReady:     col.onReady,
		OnDispatch:  col.dispatch,
	})

	waitFor(t, func() bool { return col.has("READY") && col.has("MESSAGE_CREATE") })
}

// TestZstdConcatenatedFrames covers one frame carrying several concatenated zstd frames.
func TestZstdConcatenatedFrames(t *testing.T) {
	endpoint := startFakeGateway(t, func(conn *websocket.Conn) {
		enc, err := zstd.NewWriter(nil)
		if err != nil {
			t.Errorf("zstd writer: %v", err)
			return
		}
		defer func() { _ = enc.Close() }()

		frame := func(payload string) []byte { return enc.EncodeAll([]byte(payload), nil) }

		_ = conn.WriteMessage(websocket.BinaryMessage, frame(`{"op":10,"d":{"heartbeat_interval":45000}}`))

		if op, _, err := readOp(conn); err != nil || op != OpIdentify {
			t.Errorf("first op = %d, err = %v; want IDENTIFY", op, err)
			return
		}

		_ = conn.WriteMessage(websocket.BinaryMessage, frame(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`))

		// Two whole zstd frames concatenated into a single websocket message.
		concat := append(frame(`{"op":0,"s":2,"t":"MESSAGE_CREATE","d":{"id":"1"}}`), frame(`{"op":0,"s":3,"t":"MESSAGE_CREATE","d":{"id":"2"}}`)...)
		_ = conn.WriteMessage(websocket.BinaryMessage, concat)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	col := &collector{}
	startGateway(t, Config{
		Token:       "test-token",
		Endpoint:    endpoint,
		Intents:     uint32(IntentGuilds),
		ShardCount:  1,
		Compression: CompressionZstd,
		OnReady:     col.onReady,
		OnDispatch:  col.dispatch,
	})

	waitFor(t, func() bool { return col.has("MESSAGE_CREATE") })

	col.mu.Lock()
	count := len(col.events)
	events := append([]string(nil), col.events...)
	col.mu.Unlock()
	if count != 3 {
		t.Fatalf("dispatched %v, want READY plus two MESSAGE_CREATE", events)
	}
}

// TestDecodeAllFailsOnZstdStreamChunk documents the bug this fix removes: a chunk of
// a continuous zstd stream is not a complete frame, so DecodeAll fails.
func TestDecodeAllFailsOnZstdStreamChunk(t *testing.T) {
	stream, err := newZstdStream(nil)
	if err != nil {
		t.Fatalf("zstd writer: %v", err)
	}
	defer stream.close()

	chunk := stream.encode(`{"op":10,"d":{"heartbeat_interval":45000}}`)

	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("zstd reader: %v", err)
	}
	defer dec.Close()

	if _, err := dec.DecodeAll(chunk, nil); err == nil {
		t.Fatal("DecodeAll accepted a partial zstd-stream chunk; it must require a whole frame")
	}
}
