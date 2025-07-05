package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"qord/discord"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

func trace(shardID uint16, category string, msg string, args ...any) {
	log.Printf("[Shard %02d] [%s] %s", shardID+1, category, fmt.Sprintf(msg+"\n", args...))
}

type Shard struct {
	mu               sync.RWMutex
	manager          *GatewayManager
	conn             *wsConn
	sessionID        string
	resumeGatewayURL string
	heartbeat        heartbeatState
	lastSequence     uint32
	ID               uint16 // readonly after creation
	state            ShardState
}

type ShardState uint8

const (
	UNAVAILABLE_SHARD_STATE ShardState = iota
	CONNECTING_SHARD_STATE
	READY_SHARD_STATE
)

type heartbeatState struct {
	acknowledged atomic.Bool
	lastBeat     time.Time
	interval     time.Duration
	ticker       *time.Ticker
	done         chan struct{}
}

func (shard *Shard) State() ShardState {
	shard.mu.RLock()
	state := shard.state
	shard.mu.RUnlock()
	return state
}

// Returns time elapsed since last heartbeat (aka ping).
func (shard *Shard) Latency() time.Duration {
	shard.mu.RLock()
	ping := time.Since(shard.heartbeat.lastBeat) - shard.heartbeat.interval
	shard.mu.RUnlock()
	return ping
}

func (shard *Shard) Close(resetSession bool) {
	shard.mu.Lock()

	shard.state = UNAVAILABLE_SHARD_STATE
	if shard.conn != nil {
		err := shard.conn.close()
		if err != nil {
			trace(shard.ID, "WebSocket", "Ran into issue while closing websocket connection - received error: %v", err)
		}
		shard.conn = nil
	}

	if resetSession {
		shard.lastSequence = 0
		shard.sessionID = ""
		shard.resumeGatewayURL = ""
	}

	shard.heartbeat.acknowledged.Store(true)
	shard.heartbeat.lastBeat = time.Time{}
	if shard.heartbeat.ticker != nil {
		close(shard.heartbeat.done)
		shard.heartbeat.ticker.Stop()
		shard.heartbeat.ticker = nil
		shard.heartbeat.done = nil
	}
	shard.heartbeat.interval = 0

	shard.mu.Unlock()
}

func (shard *Shard) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			trace(shard.ID, "WebSocket", "Cancelled context, requested shutdown.")
			shard.Close(true)
			return
		default:
		}

		var payload EventPacket
		err := shard.conn.readJSON(ctx, &payload)
		if err != nil {
			// trace(shard.ID, "WebSocket", "Failed to read json payload - received error: %v", err)

			var exitCode ExitCode
			if code := websocket.CloseStatus(err); code == -1 {
				if errors.Is(err, io.EOF) {
					trace(shard.ID, "WebSocket", "Received EOF (likely silent disconnect). Not resumable.")
					exitCode = SESSION_TIMED_OUT
				} else {
					trace(shard.ID, "WebSocket", "Received unknown exit code (%d). Not resumable.", code)
					exitCode = -1
				}
			} else {
				exitCode = ExitCode(code)
			}

			switch exitCode {
			case AUTHENTICATION_FAILED,
				INVALID_SHARD,
				SHARDING_REQUIRED,
				INVALID_API_VERSION,
				INVALID_INTENT,
				DISALLOWED_INTENT:
				trace(shard.ID, "WebSocket", "Received fatal exit code (%d). Not resumable.", exitCode)
				shard.Close(true)
				return
			case INVALID_SEQ, SESSION_TIMED_OUT:
				trace(shard.ID, "WebSocket", "Received error code (%d) from expected codes - shard will try to resume session on new connection.", exitCode)
				if err := shard.updateConnection(ctx, true, false); err != nil {
					trace(shard.ID, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
				}
				continue
			default:
				trace(shard.ID, "WebSocket", "Received unknown, not typical discord gateway error code (%d) - shard will attempt to open new connection and start brand new session.", exitCode)
				if err := shard.updateConnection(ctx, true, true); err != nil {
					trace(shard.ID, "WebSocket", "Attempted to open new connection & create new session but failed - received error: %v", err)
				}
				continue
			}
		}

		if payload.Sequence != 0 {
			shard.lastSequence = payload.Sequence
		}

		switch payload.Opcode {
		case DISPATCH_OPCODE:
			trace(shard.ID, "Event", "Received dispatch opcode with %s event name.", payload.Event)
			shard.dispatchEvent(ctx, payload)
		case HEARTBEAT_OPCODE:
			trace(shard.ID, "Event", "Received heartbeat opcode request - sending manual heartbeat.")
			shard.sendHeartbeat(ctx, false)
		case RECONNECT_OPCODE:
			trace(shard.ID, "Event", "Received reconnect opcode request - closing & reopening websocket connection within same session.")
			if err := shard.updateConnection(ctx, true, false); err != nil {
				trace(shard.ID, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
			}

			continue
		case INVALID_SESSION_OPCODE:
			var canResume bool
			if err := json.Unmarshal(payload.Data, &canResume); err != nil {
				trace(shard.ID, "Event", "Received invalid session opcode request but failed to decode json payload - unable to verify whether session can be resumed! (assuming not, new session is needed)")
			}

			if canResume {
				trace(shard.ID, "Event", "Received invalid session opcode request - closing & reopening websocket connection within same session.")
				if err := shard.updateConnection(ctx, true, false); err != nil {
					trace(shard.ID, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
				}
			} else {
				trace(shard.ID, "Event", "Received invalid session opcode request - closing & reopening websocket connection & creating new session.")
				if err := shard.updateConnection(ctx, true, true); err != nil {
					trace(shard.ID, "WebSocket", "Attempted to open new connection & create new session but failed - received error: %v", err)
				}
			}

			continue
		case HELLO_OPCODE:
			trace(shard.ID, "Event", "Received hello opcode - shard will now try to identify/resume.")
			shard.handleIdentify(ctx, payload.Data)
		case HEARTBEAT_ACK_OPCODE:
			// trace(shard.ID, "Event", "Received heartbeat acknowledgement opcode.")
			shard.heartbeat.acknowledged.Store(true)
		default:
			trace(shard.ID, "Event", "Received %d opcode - there's no handler for this event!", payload.Opcode)
		}
	}
}

func (shard *Shard) dispatchEvent(ctx context.Context, payload EventPacket) {
	switch payload.Event {
	case READY_EVENT:
		var readyData ReadyEventData
		err := json.Unmarshal(payload.Data, &readyData)
		if err != nil {
			trace(shard.ID, "Event", "Received request to dispatch READY event but failed to decode json payload - received error: %v (returning early, will likely result in damaged session logic)", err)
			return
		}

		shard.mu.Lock()
		shard.sessionID = readyData.SessionID
		shard.resumeGatewayURL = readyData.ResumeGatewayURL
		// shard.manager.User = readyData.User
		trace(shard.ID, "State", "Successfully joined new session. Changed status to ready!")
		shard.state = READY_SHARD_STATE

		if shard.ID+1 == shard.manager.shardCount {
			trace(shard.ID, "State", "Seems like it's last shard that just got ready - assuming all shards are ready! (if it runs for first time)")
		}
		shard.mu.Unlock()
	case RESUMED_EVENT:
		shard.mu.Lock()
		shard.state = READY_SHARD_STATE
		shard.mu.Unlock()
		trace(shard.ID, "State", "Successfully resumed session. Changed status to ready!")
	}
}

func (shard *Shard) handleIdentify(ctx context.Context, raw json.RawMessage) {
	var helloData HelloEventData
	err := json.Unmarshal(raw, &helloData)
	if err != nil {
		trace(shard.ID, "Event", "Received hello opcode but failed to decode json payload - received error: %v (returning early, cannot identify without that data so will likely result in damaged session logic)", err)
		return
	}

	shard.mu.Lock()
	shard.heartbeat.interval = time.Duration(helloData.HeartbeatInterval) * time.Millisecond
	shard.heartbeat.acknowledged.Store(true)
	shard.mu.Unlock()
	trace(shard.ID, "Heartbeat", "Started identify operation - started automatic heartbeating!")
	shard.sendHeartbeat(ctx, true)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if shard.sessionID == "" {
		err := shard.conn.sendPayload(ctx, IdentifyEvent{
			Opcode: IDENTIFY_OPCODE,
			Data: IdentifyPayloadData{
				Token:          shard.manager.token,
				Intents:        0, // In this lib, we only care about interactions.
				ShardOrder:     [2]uint16{shard.ID, shard.manager.shardCount},
				LargeThreshold: 50,
				Properties: IdentifyPayloadDataProperties{
					OS:      runtime.GOOS,
					Browser: discord.LIBRARY_NAME,
					Device:  discord.LIBRARY_NAME,
				},
			},
		})

		if err != nil {
			trace(shard.ID, "Event", "Ran into issue when tried to send identify payload - received error: %v (will likely result in damaged session logic)", err)
			return
		}

		trace(shard.ID, "Event", "Successfully sent identify payload - shard should soon receive new session details.")
	} else {
		time.Sleep(time.Duration(rand.Intn(1250)) * time.Millisecond)

		err := shard.conn.sendPayload(ctx, ResumeEvent{
			Opcode: RESUME_OPCODE,
			Data: ResumeEventData{
				Token:     shard.manager.token,
				SessionID: shard.sessionID,
				Sequence:  shard.lastSequence,
			},
		})

		if err != nil {
			trace(shard.ID, "Event", "Ran into issue when tried to send resume payload - received error: %v (will likely result in damaged session logic)", err)
			return
		}

		trace(shard.ID, "Event", "Successfully sent resume payload - shard should soon receive confirmation.")
	}
}

func (shard *Shard) sendHeartbeat(ctx context.Context, autoLoop bool) {
	if !shard.heartbeat.acknowledged.Load() {
		trace(shard.ID, "Heartbeat", "Failed to acknowledge previous beat - assuming zombified connection. Shard will try to close & reopen websocket connection within current session.")
		if err := shard.updateConnection(ctx, true, true); err != nil {
			trace(shard.ID, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
		}
		return
	}

	shard.mu.Lock()

	if autoLoop {
		trace(shard.ID, "Heartbeat", "Creating new heartbeat timer! (received interval instruction to beat every %s)", shard.heartbeat.interval.String())

		// Clean up existing timer if it exists
		if shard.heartbeat.ticker != nil {
			close(shard.heartbeat.done)
			shard.heartbeat.ticker.Stop()
			shard.heartbeat.ticker = nil
			shard.heartbeat.done = nil
		}

		jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
		ticker := time.NewTicker(shard.heartbeat.interval + jitter)
		done := make(chan struct{})
		shard.heartbeat.ticker = ticker
		shard.heartbeat.done = done

		go func(t *time.Ticker, done <-chan struct{}) {
			for {
				select {
				case <-t.C:
					// trace(shard.ID, "Heartbeat", "Sending heartbeat from automatic heartbeat ticker!")
					shard.sendHeartbeat(ctx, false)
				case <-ctx.Done():
					trace(shard.ID, "Heartbeat", "Context cancelled, stopping heartbeat ticker.")
					t.Stop()
					return
				case <-done:
					trace(shard.ID, "Heartbeat", "Heartbeat manually stopped.")
					t.Stop()
					return
				}
			}
		}(ticker, done)
	} else {
		shard.heartbeat.acknowledged.Store(false)
		shard.heartbeat.lastBeat = time.Now()

		// trace(shard.ID, "Heartbeat", "Sending heartbeat!")
		err := shard.conn.sendPayload(ctx, HeartbeatEvent{Opcode: HEARTBEAT_OPCODE, Sequence: shard.lastSequence})
		if err != nil {
			trace(shard.ID, "Heartbeat", "Failed to send heartbeat payload - received error: %v", err)
		}
	}

	shard.mu.Unlock()
}

func (shard *Shard) updateConnection(ctx context.Context, restart, resetSession bool) error {
	if shard.State() != UNAVAILABLE_SHARD_STATE {
		shard.Close(resetSession)
		time.Sleep(time.Second * 2) // Safety wait before reconnect (probably not needed but just in case)
	}

	if restart {
		var err error

		shard.mu.Lock()
		shard.state = CONNECTING_SHARD_STATE
		shard.heartbeat.acknowledged.Store(true)

		if shard.resumeGatewayURL == "" {
			trace(shard.ID, "WebSocket", "Requested new websocket connection to %s (new session?)", discord.DISCORD_GATEWAY_URL)
			shard.conn, err = createwsConnection(ctx, discord.DISCORD_GATEWAY_URL)
		} else {
			trace(shard.ID, "WebSocket", "Requested new websocket connection to %s (resume existing session?)", shard.resumeGatewayURL)
			shard.conn, err = createwsConnection(ctx, shard.resumeGatewayURL)
		}

		if err != nil {
			shard.mu.Unlock()
			trace(shard.ID, "WebSocket", "Ran into issue while trying to open (dial) new websocket connection - received error: %v", err)
			return err
		}
	}

	shard.mu.Unlock()
	return nil
}
