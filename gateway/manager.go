package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"qord/api"
	"sync"
	"time"
)

// ShardManager is responsible for orchestrating multiple Shard connections to the
// Discord Gateway. It handles everything that is required for Bot to start receiving packets with event data.
type ShardManager struct {
	token        string
	traceLogger  *log.Logger
	eventHandler func(packet EventPacket)

	mu     sync.RWMutex
	shards map[uint16]*Shard
	wg     sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

// Creates a new gateway connection manager.
// Set trace to true to enable detailed logging for the manager & all shards under it control.
func NewShardManager(token string, trace bool, eventHandler func(packet EventPacket)) *ShardManager {
	m := &ShardManager{
		token:        token,
		shards:       make(map[uint16]*Shard),
		traceLogger:  log.New(io.Discard, "[QORD] ", log.LstdFlags),
		eventHandler: eventHandler,
	}

	if trace {
		m.traceLogger.SetOutput(os.Stdout)
		m.tracef("Shard Manager tracing enabled.")
	}

	if m.eventHandler == nil {
		m.eventHandler = func(p EventPacket) {
			m.tracef("Received %s event but there's no proper event handling setup in place.", p.Event)
		}
	}

	return m
}

func (m *ShardManager) tracef(format string, v ...interface{}) {
	m.traceLogger.Printf("[MANAGER] "+format, v...)
}

func (m *ShardManager) fetchGatewayBotInfo() (*GatewayBot, error) {
	req, err := http.NewRequest(http.MethodGet, api.DISCORD_API_URL+"/gateway/bot", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", api.CONTENT_TYPE_JSON)
	req.Header.Add("User-Agent", api.USER_AGENT)
	req.Header.Add("Authorization", "Bot "+m.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get gateway info: " + resp.Status)
	}

	var gatewayBot GatewayBot
	if err := json.NewDecoder(resp.Body).Decode(&gatewayBot); err != nil {
		return nil, err
	}

	return &gatewayBot, nil
}

// Start connects the manager to Discord. It fetches the gateway configuration,
// creates the necessary shards, and starts them in parallel. This is a
// blocking call that waits for all shards to complete their lifecycle.
//
// Note: Normally Manager will ask Discord API for recommended number of shards and use that.
// You can manually change that by setting forcedShardCount param to value larger than 0.
func (m *ShardManager) Start(ctx context.Context, intents uint32, forcedShardCount uint16) error {
	m.mu.Lock()
	if len(m.shards) != 0 {
		m.mu.Unlock()
		return errors.New("manager has already started")
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	m.tracef("Starting...")

	gBot, err := m.fetchGatewayBotInfo()
	if err != nil {
		m.Stop()
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
				case <-m.ctx.Done():
					m.tracef("Cancelled context while shards were still spawning.")
					return
				default:
					m.tracef("Spawning shard ID = %d in bucket ID = %d.", shardID, bucketID)

					shard := NewShard(shardID, gBot.ShardCount, m.token, intents, m.traceLogger, m.eventHandler)

					m.mu.Lock()
					m.shards[shardID] = shard
					m.wg.Add(1)
					m.mu.Unlock()

					go func() {
						defer m.wg.Done()
						shard.Start(m.ctx, gBot.URL+"/?v=10&encoding=json")
					}()

					// Discord requires a 5-second delay between each IDENTIFY per bucket
					m.tracef("Waiting 5s before spawn of next shard in bucket ID = %d.", bucketID)
					time.Sleep(5 * time.Second)
				}
			}
		}(bucket)
	}

	spawnWg.Wait() // Wait for all shards to be launched before proceeding.
	m.tracef("All shards have been launched.")
	m.wg.Wait() // This makes Start() a blocking function.
	m.tracef("All shards have stopped.")
	return nil
}

// Stop gracefully closes all shard connections.
func (m *ShardManager) Stop() {
	m.mu.RLock()
	if len(m.shards) == 0 {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	m.tracef("Stopping all shards...")
	m.cancel()  // Signal all shards to stop
	m.wg.Wait() // Wait for all shards to exit gracefully
	m.tracef("All shards have stopped.")
}

// Sends a payload to all online shards. This is useful for
// actions that affect the bot's global state, such as presence updates.
func (m *ShardManager) Broadcast(jsonStruct any) {
	m.tracef("[MANAGER] Broadcasting payload to all online shards!")
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, shard := range m.shards {
		// Send concurrently to avoid a slow shard blocking others.
		go func(s *Shard) {
			if err := s.Send(jsonStruct); err != nil {
				s.tracef("[BROADCAST] Error sending to shard ID = %d: %v", s.ID, err)
			}
		}(shard)
	}
}
