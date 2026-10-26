package tempest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// ShardManager is responsible for orchestrating multiple Shard connections to the
// Discord Gateway. It handles everything that is required for Bot to start receiving packets with event data.
type ShardManager struct {
	traceLogger  *log.Logger
	eventHandler func(shardID uint16, packet EventPacket)
	onStart      func(context.Context)
	shards       map[uint16]*Shard
	token        string
	wg           sync.WaitGroup

	mu       sync.RWMutex
	compress bool
}

// Creates a new gateway connection manager.
// Set trace to true to enable detailed logging for the manager & all shards under its control.
// If tracing is enabled and logger is provided, it'll be used for all internal messages. If none is provided, the default Stdout logger will be used instead.
//
// Set zstdCompression to true to enable zstd-stream transport compression.
func NewShardManager(token string, trace bool, zstdCompression bool, eventHandler func(shardID uint16, packet EventPacket), logger *log.Logger) *ShardManager {
	m := &ShardManager{
		token:        token,
		shards:       make(map[uint16]*Shard),
		traceLogger:  logger,
		compress:     zstdCompression,
		eventHandler: eventHandler,
	}

	if m.traceLogger == nil {
		m.traceLogger = log.New(io.Discard, "[TEMPEST] ", log.LstdFlags)
	}

	if trace {
		w := m.traceLogger.Writer()
		if w == nil || w == io.Discard {
			m.traceLogger.SetOutput(os.Stdout)
		}
		m.tracef("Shard Manager tracing enabled.")
	}

	if m.eventHandler == nil {
		m.eventHandler = func(shardID uint16, packet EventPacket) {
			m.tracef("Received %s event from shard ID = %d but there's no proper event handling setup in place.", packet.Event, shardID)
		}
	}

	return m
}

// Start connects the manager to Discord. It fetches the gateway configuration,
// creates the necessary shards, and starts them in parallel. This is a
// blocking call that waits for all shards to complete their lifecycle.
//
// Canceling ctx stops the startup process and gracefully closes every active
// shard. Start returns nil after a context-triggered shutdown. Callers do not
// need a separate stop method; the context owns the connection lifecycle just
// like it does for an http.Server.
//
// Note: Normally Manager will ask Discord API for recommended number of shards and use that.
// You can manually change that by setting forcedShardCount param to value larger than 0.
// There's also an option to provide a callback invoked after all shards have been launched.
func (m *ShardManager) Start(ctx context.Context, intents uint32, forcedShardCount uint16, launchedCallbackFn func()) error {
	m.mu.Lock()
	if len(m.shards) != 0 {
		m.mu.Unlock()
		return errors.New("manager has already started")
	}
	m.mu.Unlock()

	if m.onStart != nil {
		m.onStart(ctx)
	}

	m.tracef("Starting...")

	gBot, err := m.fetchGatewayBotInfo(ctx)
	if err != nil {
		if ctx.Err() != nil {
			m.tracef("Context canceled before gateway startup completed.")
			return nil
		}
		return err
	}

	if forcedShardCount != 0 {
		gBot.ShardCount = forcedShardCount
	}

	m.tracef("Starting recommended number of shards (%d) in series of %d.", gBot.ShardCount, gBot.SessionStartLimit.MaxConcurrency)

	var spawnWg sync.WaitGroup
	for bucket := uint16(0); bucket < gBot.SessionStartLimit.MaxConcurrency; bucket++ {
		spawnWg.Add(1)

		go func(bucketID uint16) {
			defer spawnWg.Done()

			for shardID := bucketID; shardID < gBot.ShardCount; shardID += gBot.SessionStartLimit.MaxConcurrency {
				select {
				case <-ctx.Done():
					m.tracef("Cancelled context while shards were still spawning.")
					return
				default:
					m.tracef("Spawning shard ID = %d in bucket ID = %d.", shardID, bucketID)

					shard := NewShard(
						shardID,
						gBot.ShardCount,
						m.token,
						intents,
						m.compress,
						m.traceLogger,
						m.eventHandler,
					)

					m.mu.Lock()
					m.shards[shardID] = shard
					m.wg.Add(1)
					m.mu.Unlock()

					go func() {
						defer m.wg.Done()
						url := gBot.URL + "/?v=10&encoding=json"
						if m.compress {
							url += "&compress=zstd-stream"
						}
						shard.Start(ctx, url)
					}()

					// Discord requires a 5-second delay between each IDENTIFY per bucket.
					// Skip the delay when this bucket has no more shards to launch, and
					// use a timer instead of time.Sleep so cancellation is immediate.
					if shardID+gBot.SessionStartLimit.MaxConcurrency >= gBot.ShardCount {
						break
					}

					m.tracef("Waiting 5s before spawn of next shard in bucket ID = %d.", bucketID)
					timer := time.NewTimer(5 * time.Second)
					select {
					case <-ctx.Done():
						timer.Stop()
						m.tracef("Cancelled context while waiting to spawn the next shard.")
						return
					case <-timer.C:
					}
				}
			}
		}(bucket)
	}

	spawnWg.Wait() // Wait for all shards to be launched before proceeding.
	if ctx.Err() == nil {
		if launchedCallbackFn != nil {
			launchedCallbackFn()
		}
		m.tracef("All shards have been launched.")
	} else {
		m.tracef("Context canceled before all shards were launched.")
	}

	m.wg.Wait() // This makes Start() a blocking function.
	m.tracef("All shards have stopped.")
	return nil
}

func (m *ShardManager) Status() map[uint16]ShardState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.shards) == 0 {
		return nil
	}

	res := make(map[uint16]ShardState, len(m.shards))
	for _, s := range m.shards {
		res[s.ID] = s.Status()
	}

	return res
}

func (m *ShardManager) Send(shardID uint16, jsonStruct any) {
	m.tracef("Trying to send payload to shard ID = %d!", shardID)
	m.mu.RLock()
	defer m.mu.RUnlock()

	if int(shardID) >= len(m.shards) {
		m.tracef("Tried sending payload via invalid shard (ID = %d) - such shard does not exist.", shardID)
		return
	}

	shard, ok := m.shards[shardID]
	if !ok || shard.Status() != ONLINE_SHARD_STATE {
		m.tracef("Failed to send payload to shard ID = %d: session status is not \"ONLINE\".", shardID)
		return
	}

	go func(s *Shard) {
		if err := s.Send(jsonStruct); err != nil {
			s.tracef("Error sending payload to shard ID = %d: %v", s.ID, err)
		}
	}(shard)
}

// Sends a payload to all online shards. This is useful for
// actions that affect the bot's global state, such as presence updates.
func (m *ShardManager) Broadcast(jsonStruct any) {
	m.tracef("Broadcasting payload to all online shards!")
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, shard := range m.shards {
		if shard.Status() != ONLINE_SHARD_STATE {
			m.tracef("Error broadcasting payload to shard ID = %d: session status is not \"ONLINE\".", shard.ID)
			continue
		}

		// Send concurrently to avoid a slow shard blocking others.
		go func(s *Shard) {
			if err := s.Send(jsonStruct); err != nil {
				m.tracef("Error broadcasting payload to shard ID = %d: %v", s.ID, err)
			}
		}(shard)
	}
}

// Allows to update status (presence) of the bot.
// Have generator function return nil to skip updating status for a specific shard.
func (m *ShardManager) UpdateStatus(statusGeneratorFn func(shardID uint16) *UpdatePresenceEvent) {
	m.tracef("Setting status for selected online shards!")
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, shard := range m.shards {
		if shard.Status() != ONLINE_SHARD_STATE {
			continue
		}

		payload := statusGeneratorFn(shard.ID)
		if payload == nil {
			continue
		}

		if payload.Opcode == 0 {
			payload.Opcode = PRESENCE_UPDATE_OPCODE
		}

		go func(s *Shard, p *UpdatePresenceEvent) {
			if err := s.UpdatePresence(p); err != nil {
				m.tracef("Failed to update status for shard %d: %v", s.ID, err)
			}
		}(shard, payload)
	}
}

// Returns shard status and ping value (calculated based on shard heartbeat) for each started shard.
func (m *ShardManager) AllShardDetails() ([]ShardState, []time.Duration) {
	m.tracef("Requested for ping value from all shards.")
	m.mu.RLock()
	defer m.mu.RUnlock()

	resState := make([]ShardState, len(m.shards))
	resPing := make([]time.Duration, len(m.shards))
	for _, shard := range m.shards {
		resState[shard.ID] = shard.Status()
		resPing[shard.ID] = shard.Ping()
	}
	return resState, resPing
}

// Returns shard status and ping value (calculated based on shard heartbeat).
func (m *ShardManager) ShardDetails(shardID uint16) (ShardState, time.Duration, error) {
	m.tracef("Requested shard state & ping value from %d shard.", shardID)
	m.mu.RLock()
	defer m.mu.RUnlock()

	if int(shardID) > len(m.shards)-1 {
		m.tracef("Requested invalid shard ID - request for %d shard details will return error.", shardID)
		return OFFLINE_SHARD_STATE, 0, errors.New("invalid shard ID")
	}

	s := m.shards[shardID]
	return s.Status(), s.Ping(), nil
}

func (m *ShardManager) tracef(format string, v ...any) {
	m.traceLogger.Printf("[(GATEWAY) SHARD MANAGER] "+format, v...)
}

func (m *ShardManager) fetchGatewayBotInfo(ctx context.Context) (*GatewayBot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DiscordAPIBaseURL()+"/gateway/bot", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", CONTENT_TYPE_JSON)
	req.Header.Add("User-Agent", USER_AGENT)
	req.Header.Add("Authorization", "Bot "+m.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			m.tracef("Failed to close response body: %v.", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get gateway info: " + resp.Status)
	}

	var gatewayBot GatewayBot
	if err := json.NewDecoder(resp.Body).Decode(&gatewayBot); err != nil {
		return nil, err
	}

	return &gatewayBot, nil
}
