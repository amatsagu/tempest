package gateway

import (
	"context"
	"encoding/json/v2"
	"errors"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	errReconnect = errors.New("gateway: reconnect requested")
	errNoAck     = errors.New("gateway: heartbeat not acknowledged")
	errBadHello  = errors.New("gateway: expected HELLO first")
)

const (
	readTimeout = 90 * time.Second
	writeWait   = 10 * time.Second
)

// Outbound is a raw gateway payload for Send and Broadcast: Op is the opcode and
// Data its payload, mirroring the wire envelope.
// https://docs.discord.com/developers/events/gateway#payload-structure
type Outbound struct {
	Data any    `json:"d,omitzero"`
	Op   Opcode `json:"op"`
}

// shard owns one gateway websocket connection and its session state.
type shard struct {
	g             *Gateway
	conn          *websocket.Conn
	sessionID     string
	resumeURL     string
	id            int
	heartbeatSent atomic.Int64
	latencyNanos  atomic.Int64
	seq           atomic.Int64
	state         atomic.Uint32
	mu            sync.Mutex
	identify      bool
	acked         atomic.Bool
	hasLatency    atomic.Bool
}

// setState records the shard's connection state for Shards.
func (s *shard) setState(state ShardState) { s.state.Store(uint32(state)) }

// info snapshots the shard's state and heartbeat latency.
func (s *shard) info() ShardInfo {
	info := ShardInfo{State: ShardState(s.state.Load())}
	if s.hasLatency.Load() {
		info.Latency = time.Duration(s.latencyNanos.Load())
	}
	return info
}

func newShard(g *Gateway, id int) *shard {
	return &shard{g: g, id: id}
}

// run keeps a shard connected across reconnects until the context is done.
func (s *shard) run(ctx context.Context) {
	attempt := 0

	for ctx.Err() == nil {
		err := s.connect(ctx)
		s.setState(ShardOffline)
		if ctx.Err() != nil {
			return
		}

		if errors.Is(err, errReconnect) {
			attempt = 0
		} else if err == nil {
			attempt = 0
		} else {
			attempt++
			s.g.reportError(s.id, err)
		}

		delay := backoff(attempt)
		if s.g.debug {
			s.g.debugf("shard %d reconnecting in %s (err=%v)", s.id, delay, err)
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *shard) connect(ctx context.Context) error {
	s.setState(ShardConnecting)

	dialer := s.g.dialer
	conn, _, err := dialer.DialContext(ctx, s.endpoint(), nil)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	// Close the socket when the context is done so a blocked read loop unblocks now.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	defer func() {
		s.mu.Lock()
		s.conn = nil
		s.mu.Unlock()
		_ = conn.Close()
	}()

	dec, err := newMessageDecoder(conn, s.g.cfg.Compression)
	if err != nil {
		return err
	}
	defer dec.close()

	var hello payload
	if err := dec.decode(&hello); err != nil {
		return err
	}

	if hello.Op != OpHello {
		return errBadHello
	}

	var helloData helloData
	if err := json.Unmarshal(hello.Data, &helloData); err != nil {
		return err
	}
	interval := time.Duration(helloData.HeartbeatInterval) * time.Millisecond
	if interval <= 0 {
		interval = 45 * time.Second
	}

	// Identify or resume before the heartbeat loop starts, so a heartbeat can never
	// reach Discord ahead of the handshake that opens the session.
	s.setState(ShardAuthenticating)

	if s.sessionID != "" && s.resumeURL != "" {
		if err := s.sendResume(); err != nil {
			return err
		}
	} else {
		s.g.identify.wait(ctx)
		if err := s.sendIdentify(); err != nil {
			return err
		}
	}

	s.acked.Store(true)

	hbCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.heartbeat(hbCtx, interval)

	return s.readLoop(conn, dec)
}

func (s *shard) readLoop(conn *websocket.Conn, dec *messageDecoder) error {
	for {
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))

		var p payload
		if err := dec.decode(&p); err != nil {
			return err
		}

		if p.Seq != nil {
			s.seq.Store(*p.Seq)
		}

		switch p.Op {
		case OpDispatch:
			s.handleDispatch(p)
		case OpHeartbeat:
			_ = s.sendHeartbeat()
		case OpHeartbeatACK:
			s.acked.Store(true)
			s.recordHeartbeatACK()
		case OpReconnect:
			return errReconnect
		case OpInvalidSession:
			var resumable bool
			_ = json.Unmarshal(p.Data, &resumable)
			if !resumable {
				s.sessionID = ""
			}
			return errReconnect
		}
	}
}

func (s *shard) handleDispatch(p payload) {
	switch p.EventName {
	case EventReady:
		var ready readyData
		if err := json.Unmarshal(p.Data, &ready); err == nil {
			s.sessionID = ready.SessionID
			s.resumeURL = ready.ResumeGatewayURL
			s.g.maybeReshard(len(ready.Guilds))
		}
		s.identify = true
		s.setState(ShardOnline)
		s.g.markReady(s.id)
	case EventResumed:
		s.setState(ShardOnline)
		s.g.markReady(s.id)
	}

	if s.g.cfg.OnDispatch != nil {
		s.g.cfg.OnDispatch(s.id, p.EventName, p.Data)
	}
}

func (s *shard) heartbeat(ctx context.Context, interval time.Duration) {
	jitter := time.Duration(rand.Float64() * float64(interval))

	timer := time.NewTimer(jitter)
	select {
	case <-ctx.Done():
		timer.Stop()
		return
	case <-timer.C:
	}

	if err := s.sendHeartbeat(); err != nil {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.acked.Load() {
				s.g.reportError(s.id, errNoAck)
				s.forceClose()
				return
			}

			if err := s.sendHeartbeat(); err != nil {
				return
			}
		}
	}
}

func (s *shard) sendHeartbeat() error {
	s.acked.Store(false)

	// Stamp the send so HEARTBEAT_ACK can measure the round trip; both stamps are
	// atomics because any goroutine may read them via ShardLatency.
	s.heartbeatSent.Store(time.Now().UnixNano())
	return s.write(Outbound{Op: OpHeartbeat, Data: s.seq.Load()})
}

// recordHeartbeatACK stores the round trip of the heartbeat sendHeartbeat stamped.
func (s *shard) recordHeartbeatACK() {
	if sent := s.heartbeatSent.Load(); sent != 0 {
		s.latencyNanos.Store(time.Since(time.Unix(0, sent)).Nanoseconds())
		s.hasLatency.Store(true)
	}
}

func (s *shard) latency() (time.Duration, bool) {
	if !s.hasLatency.Load() {
		return 0, false
	}
	return time.Duration(s.latencyNanos.Load()), true
}

func (s *shard) sendIdentify() error {
	return s.write(Outbound{Op: OpIdentify, Data: identifyData{
		Token:   s.g.cfg.Token,
		Intents: s.g.cfg.Intents,
		Properties: identifyProps{
			OS:      "linux",
			Browser: "tempest",
			Device:  "tempest",
		},
		Shard:    [2]int{s.id, int(s.g.shardCount.Load())},
		Presence: s.g.presenceFor(s.id),
	}})
}

func (s *shard) updatePresence(presence *Presence) error {
	return s.write(Outbound{Op: OpPresenceUpdate, Data: presence})
}

func (s *shard) sendResume() error {
	return s.write(Outbound{Op: OpResume, Data: resumeData{
		Token:     s.g.cfg.Token,
		SessionID: s.sessionID,
		Seq:       s.seq.Load(),
	}})
}

func (s *shard) write(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return errReconnect
	}

	_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return s.conn.WriteMessage(websocket.TextMessage, data)
}

func (s *shard) forceClose() {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func (s *shard) endpoint() string {
	base := s.g.cfg.Endpoint
	if base == "" {
		base = "wss://gateway.discord.gg"
	}

	if s.sessionID != "" && s.resumeURL != "" {
		base = s.resumeURL
	}

	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}

	url := base + sep + "v=10&encoding=json"
	if q := s.g.cfg.Compression.queryValue(); q != "" {
		url += "&compress=" + q
	}
	return url
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	if attempt > 10 {
		attempt = 10
	}

	base := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
	if base > 30*time.Second {
		base = 30 * time.Second
	}

	jitter := time.Duration(rand.Float64() * float64(base) * 0.25)
	return base + jitter
}
