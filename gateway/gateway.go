package gateway

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Config configures a Gateway. Fields are ordered to minimize struct padding.
type Config struct {
	// OnReady fires when a shard becomes READY or RESUMED.
	OnReady func(shard int)

	// Logf is the debug sink. When nil, logging is disabled.
	Logf func(format string, args ...any)

	// Presence is an optional initial presence.
	Presence *Presence

	// OnError receives connection errors. It is optional.
	OnError func(shard int, err error)

	// OnDispatch receives every DISPATCH (op 0) event in raw form; internally
	// handled session opcodes are never delivered here.
	OnDispatch func(shard int, event string, data jsontext.Value)

	// OnReshard fires when the gateway decides to re-shard, with the new count.
	OnReshard func(shards int)

	// Token is the bot token.
	Token string

	// Endpoint is the gateway base URL; defaults to wss://gateway.discord.gg.
	Endpoint string

	// MaxConcurrency is the identify rate-limit bucket count. Defaults to 1.
	MaxConcurrency int

	// ShardCount forces a shard count. When zero, a single shard is used.
	ShardCount int

	// Intents is the gateway intent bitfield.
	Intents uint32

	// Debug toggles debug logging; when false the log call sites are guarded.
	Debug bool

	// Compression enables optional payload compression.
	Compression Compression

	// AutoShard requests a re-shard when READY reports more guilds than the current
	// shard count can handle.
	AutoShard bool
}

// ShardState is a shard's connection state.
type ShardState uint8

const (
	ShardOffline ShardState = iota
	ShardConnecting
	ShardAuthenticating
	ShardOnline
)

func (s ShardState) String() string {
	switch s {
	case ShardConnecting:
		return "connecting"
	case ShardAuthenticating:
		return "authenticating"
	case ShardOnline:
		return "online"
	default:
		return "offline"
	}
}

// ShardInfo is one shard's state and heartbeat latency.
type ShardInfo struct {
	// Latency is the heartbeat round trip; zero until the first ACK.
	Latency time.Duration
	// State is the connection state.
	State ShardState
}

// Gateway runs one goroutine per shard and delivers dispatch events.
type Gateway struct {
	statusFn        func(shard int) *Presence
	done            chan struct{}
	dialer          *websocket.Dialer
	identify        *identifyLimiter
	reshardCh       chan int
	cancel          context.CancelFunc
	shards          []*shard
	cfg             Config
	reshardDelayMax time.Duration
	reshardDelayMin time.Duration
	mu              sync.Mutex
	requested       atomic.Bool
	started         atomic.Bool
	shardCount      atomic.Int32
	debug           bool
}

// New builds a Gateway. Call Run to connect.
func New(cfg Config) *Gateway {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 1
	}

	if cfg.ShardCount <= 0 {
		cfg.ShardCount = 1
	}

	g := &Gateway{
		cfg:       cfg,
		debug:     cfg.Debug && cfg.Logf != nil,
		reshardCh: make(chan int, 1),
		dialer: &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		},
	}
	g.shardCount.Store(int32(cfg.ShardCount))
	g.identify = newIdentifyLimiter(cfg.MaxConcurrency)
	g.reshardDelayMin = time.Second
	g.reshardDelayMax = 5 * time.Second

	return g
}

// ShardCount returns the number of shards the gateway currently runs.
func (g *Gateway) ShardCount() int { return int(g.shardCount.Load()) }

// Shards returns each running shard's state and heartbeat latency, indexed by
// shard ID. It is a snapshot and never blocks on a shard.
func (g *Gateway) Shards() []ShardInfo {
	g.mu.Lock()
	shards := g.shards
	g.mu.Unlock()

	infos := make([]ShardInfo, len(shards))
	for i, s := range shards {
		infos[i] = s.info()
	}
	return infos
}

// SetBotInfo applies Discord's recommended shard count, gateway endpoint and
// identify concurrency. Call it before Run; values left zero are ignored.
func (g *Gateway) SetBotInfo(shards int, endpoint string, maxConcurrency int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if shards > 0 {
		g.cfg.ShardCount = shards
		g.shardCount.Store(int32(shards))
	}

	if endpoint != "" {
		g.cfg.Endpoint = endpoint
	}

	if maxConcurrency > 0 {
		g.cfg.MaxConcurrency = maxConcurrency
		g.identify = newIdentifyLimiter(maxConcurrency)
	}
}

// UpdateStatus sets the presence shards broadcast, now and on every reconnect. fn
// receives the shard ID and may return nil to leave that shard unchanged; it may be
// called from several goroutines, so it must be safe to call concurrently.
func (g *Gateway) UpdateStatus(fn func(shard int) *Presence) error {
	if fn == nil {
		return errors.New("gateway: status function is nil")
	}

	g.mu.Lock()
	g.statusFn = fn
	shards := g.shards
	g.mu.Unlock()

	for i, s := range shards {
		presence := fn(i)
		if presence == nil {
			continue
		}

		if err := s.updatePresence(presence); err != nil {
			return err
		}
	}
	return nil
}

// Send writes one raw payload to a shard, for opcodes the library does not wrap,
// such as a voice state update or a member request.
func (g *Gateway) Send(shard int, payload Outbound) error {
	g.mu.Lock()
	shards := g.shards
	g.mu.Unlock()

	if shard < 0 || shard >= len(shards) {
		return fmt.Errorf("gateway: shard %d is not running", shard)
	}
	return shards[shard].write(payload)
}

// Broadcast writes one raw payload to every running shard.
func (g *Gateway) Broadcast(payload Outbound) error {
	g.mu.Lock()
	shards := g.shards
	g.mu.Unlock()

	for _, s := range shards {
		if err := s.write(payload); err != nil {
			return err
		}
	}
	return nil
}

// presenceFor resolves the presence a shard should identify with.
func (g *Gateway) presenceFor(shard int) *Presence {
	g.mu.Lock()
	fn := g.statusFn
	g.mu.Unlock()

	if fn != nil {
		if presence := fn(shard); presence != nil {
			return presence
		}
	}
	return g.cfg.Presence
}

// ShardLatency returns a shard's heartbeat round trip, false before the first ACK
// or for an out-of-range shard.
func (g *Gateway) ShardLatency(shard int) (time.Duration, bool) {
	g.mu.Lock()
	shards := g.shards
	g.mu.Unlock()

	if shard < 0 || shard >= len(shards) {
		return 0, false
	}
	return shards[shard].latency()
}

// Run connects all shards and blocks until ctx is cancelled, re-sharding as needed.
func (g *Gateway) Run(ctx context.Context) error {
	if g.cfg.Token == "" {
		return errors.New("gateway: token is required")
	}

	if !g.started.CompareAndSwap(false, true) {
		return errors.New("gateway: already running")
	}

	runCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel
	g.done = make(chan struct{})
	defer close(g.done)
	defer cancel()

	for {
		count := int(g.shardCount.Load())
		shards := make([]*shard, count)
		innerCtx, innerCancel := context.WithCancel(runCtx)

		var wg sync.WaitGroup
		for i := range shards {
			shards[i] = newShard(g, i)
			wg.Add(1)
			go func(s *shard) {
				defer wg.Done()
				s.run(innerCtx)
			}(shards[i])
		}

		g.mu.Lock()
		g.shards = shards
		g.mu.Unlock()

		select {
		case <-runCtx.Done():
			innerCancel()
			wg.Wait()
			return runCtx.Err()

		case needed := <-g.reshardCh:
			innerCancel()
			wg.Wait()

			g.shardCount.Store(int32(needed))
			g.requested.Store(false)
			if g.debug {
				g.debugf("resharding to %d shard(s)", needed)
			}

			if g.cfg.OnReshard != nil {
				g.cfg.OnReshard(needed)
			}

			// Discord asks for a random delay before reconnecting.
			span := g.reshardDelayMax - g.reshardDelayMin
			delay := g.reshardDelayMin
			if span > 0 {
				delay += time.Duration(rand.Float64() * float64(span))
			}
			timer := time.NewTimer(delay)
			select {
			case <-runCtx.Done():
				timer.Stop()
				return runCtx.Err()
			case <-timer.C:
			}
		}
	}
}

// Close stops every shard and waits for them, returning ctx.Err() if the wait is
// cut short. It is safe to call multiple times and from any goroutine.
func (g *Gateway) Close(ctx context.Context) error {
	if g.cancel != nil {
		g.cancel()
	}

	if g.done == nil {
		return nil
	}

	select {
	case <-g.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *Gateway) maybeReshard(guilds int) {
	if !g.cfg.AutoShard || !g.started.Load() {
		return
	}

	needed := RecommendedShardCount(guilds)
	if needed <= int(g.shardCount.Load()) {
		return
	}

	if !g.requested.CompareAndSwap(false, true) {
		return
	}

	select {
	case g.reshardCh <- needed:
	default:
	}
}

func (g *Gateway) markReady(shard int) {
	if g.cfg.OnReady != nil {
		g.cfg.OnReady(shard)
	}
}

func (g *Gateway) reportError(shard int, err error) {
	if g.cfg.OnError != nil && err != nil {
		g.cfg.OnError(shard, err)
	}
}

func (g *Gateway) debugf(format string, args ...any) {
	if g.debug {
		g.cfg.Logf(format, args...)
	}
}

// RecommendedShardCount returns Discord's recommended shard count (one per 1000 guilds).
func RecommendedShardCount(guilds int) int {
	if guilds <= 0 {
		return 1
	}
	shards := (guilds + 999) / 1000
	if shards < 1 {
		return 1
	}
	return shards
}

// identifyLimiter enforces Discord's identify rate limit of max per 5s window.
type identifyLimiter struct {
	last   time.Time
	tokens int
	max    int
	mu     sync.Mutex
}

func newIdentifyLimiter(max int) *identifyLimiter {
	return &identifyLimiter{max: max, tokens: max, last: time.Now()}
}

func (l *identifyLimiter) wait(ctx context.Context) {
	const window = 5 * time.Second

	for {
		l.mu.Lock()
		now := time.Now()
		if elapsed := now.Sub(l.last); elapsed >= window {
			l.tokens = l.max
			l.last = now
		}

		if l.tokens > 0 {
			l.tokens--
			l.mu.Unlock()
			return
		}
		wait := window - now.Sub(l.last)
		l.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
