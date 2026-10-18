package gateway

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRecommendedShardCount(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, 1000: 1, 1001: 2, 5000: 5, 15000: 15, 100000: 100}
	for guilds, want := range cases {
		if got := RecommendedShardCount(guilds); got != want {
			t.Errorf("RecommendedShardCount(%d) = %d, want %d", guilds, got, want)
		}
	}
}

func TestBackoffGrows(t *testing.T) {
	// Early attempts are small, later ones larger, and all stay bounded.
	if d := backoff(1); d > 260*time.Millisecond {
		t.Errorf("backoff(1) = %v, want <= 260ms", d)
	}

	if d := backoff(5); d < 3200*time.Millisecond {
		t.Errorf("backoff(5) = %v, want >= 3.2s", d)
	}

	if d := backoff(50); d > 40*time.Second {
		t.Errorf("backoff(50) = %v, want <= 40s", d)
	}
}

func TestDebugLoggingZeroCostWhenDisabled(t *testing.T) {
	g := New(Config{}) // Debug false, Logf nil

	allocs := testing.AllocsPerRun(200, func() {
		if g.debug {
			g.debugf("shard %d %v", 1, "x")
		}
	})

	if allocs != 0 {
		t.Fatalf("disabled debug logging allocated %v per call, want 0", allocs)
	}
}

type collector struct {
	events []string
	mu     sync.Mutex
	ready  bool
}

func (c *collector) dispatch(_ int, event string, _ jsontext.Value) {
	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()
}

func (c *collector) onReady(int) {
	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
}

func (c *collector) has(event string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e == event {
			return true
		}
	}
	return false
}

// fakeGateway runs a minimal Discord gateway: HELLO, READY, one dispatch, heartbeats.
func fakeGateway(t *testing.T, compress bool) *httptest.Server {
	t.Helper()

	upgrader := websocket.Upgrader{}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		var stream *zstdStream
		if compress {
			stream, _ = newZstdStream(conn)
			if stream != nil {
				defer stream.close()
			}
		}

		send := func(payload string) {
			if stream != nil {
				stream.send(payload)
				return
			}
			_ = conn.WriteMessage(websocket.TextMessage, []byte(payload))
		}

		send(`{"op":10,"d":{"heartbeat_interval":45000}}`)

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var envelope struct {
				Op int `json:"op"`
			}
			_ = json.Unmarshal(data, &envelope)

			switch envelope.Op {
			case 2:
				send(`{"op":0,"s":1,"t":"READY","d":{"session_id":"abc","resume_gateway_url":"ws://localhost"}}`)
				send(`{"op":0,"s":2,"t":"MESSAGE_CREATE","d":{"id":"1"}}`)
			case 1:
				send(`{"op":11}`)
			}
		}
	}))
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func TestGatewayDispatchAndReady(t *testing.T) {
	runGatewayCase(t, false)
}

func TestGatewayDispatchZstd(t *testing.T) {
	runGatewayCase(t, true)
}

func runGatewayCase(t *testing.T, compress bool) {
	t.Helper()

	server := fakeGateway(t, compress)
	defer server.Close()

	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")

	col := &collector{}
	g := New(Config{
		Token:       "test-token",
		Endpoint:    endpoint,
		Compression: compressionFor(compress),
		OnDispatch:  col.dispatch,
		OnReady:     col.onReady,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = g.Run(ctx)
		close(done)
	}()

	waitFor(t, func() bool { return col.has("MESSAGE_CREATE") })

	col.mu.Lock()
	ready := col.ready
	col.mu.Unlock()
	if !ready {
		t.Error("OnReady was not called")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("gateway did not shut down after cancel")
	}
}

func compressionFor(compress bool) Compression {
	if compress {
		return CompressionZstd
	}
	return CompressionNone
}

func TestGatewayAutoReshard(t *testing.T) {
	upgrader := websocket.Upgrader{}
	guilds := strings.TrimSuffix(strings.Repeat("{},", 1500), ",")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":10,"d":{"heartbeat_interval":45000}}`))

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var envelope struct {
				Op int `json:"op"`
			}
			_ = json.Unmarshal(data, &envelope)

			if envelope.Op == 2 {
				ready := `{"op":0,"s":1,"t":"READY","d":{"session_id":"x","resume_gateway_url":"ws://localhost","guilds":[` + guilds + `]}}`
				_ = conn.WriteMessage(websocket.TextMessage, []byte(ready))
			}
		}
	}))
	defer server.Close()

	var resharded atomic.Int32
	col := &collector{}

	g := New(Config{
		Token:      "test-token",
		Endpoint:   "ws" + strings.TrimPrefix(server.URL, "http"),
		ShardCount: 1,
		AutoShard:  true,
		OnDispatch: col.dispatch,
		OnReady:    col.onReady,
		OnReshard:  func(shards int) { resharded.Store(int32(shards)) },
	})
	g.reshardDelayMin = 0
	g.reshardDelayMax = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = g.Run(ctx)
		close(done)
	}()

	waitFor(t, func() bool { return resharded.Load() == 2 })
	waitFor(t, func() bool { return g.ShardCount() == 2 })

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("gateway did not shut down after cancel")
	}
}
