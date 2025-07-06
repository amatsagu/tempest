package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// Represents single connection with discord gateway.
// Shards are fully controlled by GatewayManager - do not use them directly.
type shard struct {
	mu               sync.RWMutex
	manager          *GatewayManager // readonly after start
	conn             *wsConn
	hb               *heartbeat
	sessionID        string
	resumeGatewayURL string
	lastSequence     atomic.Uint32
	id               uint16 // readonly after start
	state            ShardState
}

type ShardState uint8

const (
	UNAVAILABLE_SHARD_STATE ShardState = iota
	CONNECTING_SHARD_STATE
	READY_SHARD_STATE
)

func (shard *shard) close(resetSession bool) error {
	shard.mu.Lock()

	shard.state = UNAVAILABLE_SHARD_STATE
	if shard.conn != nil {
		err := shard.conn.close()
		if err != nil {
			shard.mu.Unlock()
			return err
		}
		shard.conn = nil
	}

	if resetSession {
		shard.lastSequence.Store(0)
		shard.sessionID = ""
		shard.resumeGatewayURL = ""
	}

	shard.hb.reset()
	shard.mu.Unlock()
	return nil
}

// Closes current WS connection and starts new one with optional session resume.
func (shard *shard) reset(ctx context.Context, resetSession bool) error {
	err := shard.close(resetSession)
	if err != nil {
		trace(shard.id, "WebSocket", "Ran into issue while closing websocket connection - received error: %v", err)
	}

	shard.mu.Lock()
	shard.state = CONNECTING_SHARD_STATE
	time.Sleep(time.Second * 2) // Safety wait before reconnect (probably not needed but just in case)

	if shard.resumeGatewayURL == "" {
		trace(shard.id, "WebSocket", "Requested new websocket connection.")
		shard.conn, err = createwsConnection(ctx, discord.DISCORD_GATEWAY_URL)
	} else {
		trace(shard.id, "WebSocket", "Requested new websocket connection to %s (resuming session)", shard.resumeGatewayURL)
		shard.conn, err = createwsConnection(ctx, shard.resumeGatewayURL)
	}

	if err != nil {
		shard.mu.Unlock()
		trace(shard.id, "WebSocket", "Ran into issue while trying to open (dial) new websocket connection - received error: %v", err)
		return err
	}

	shard.mu.Unlock()
	return nil
}

func (shard *shard) sendHeartbeat(ctx context.Context) error {
	return shard.conn.sendPayload(ctx, HeartbeatEvent{Opcode: HEARTBEAT_OPCODE, Sequence: shard.lastSequence.Load()})
}

func (shard *shard) sendIdentify(ctx context.Context) error {
	return shard.conn.sendPayload(ctx, IdentifyEvent{
		Opcode: IDENTIFY_OPCODE,
		Data: IdentifyPayloadData{
			Token:          shard.manager.token,
			Intents:        0, // In this lib, we only care about interactions.
			ShardOrder:     [2]uint16{shard.id, shard.manager.shardCount},
			LargeThreshold: 50,
			Properties: IdentifyPayloadDataProperties{
				OS:      runtime.GOOS,
				Browser: discord.LIBRARY_NAME,
				Device:  discord.LIBRARY_NAME,
			},
		},
	})
}

func (shard *shard) sendResume(ctx context.Context) error {
	return shard.conn.sendPayload(ctx, ResumeEvent{
		Opcode: RESUME_OPCODE,
		Data: ResumeEventData{
			Token:     shard.manager.token,
			SessionID: shard.sessionID,
			Sequence:  shard.lastSequence.Load(),
		},
	})
}

func (shard *shard) handleHello(ctx context.Context, raw json.RawMessage) {
	var helloData HelloEventData
	err := json.Unmarshal(raw, &helloData)
	if err != nil {
		trace(shard.id, "Event", "Received hello opcode but failed to decode json payload - received error: %v (returning early, cannot identify without that data so will likely result in damaged session logic)", err)
		return
	}

	shard.hb.start(ctx, time.Duration(helloData.HeartbeatInterval)*time.Millisecond)
	shard.hb.manualBeat(ctx)
	shard.mu.RLock()

	if shard.sessionID == "" {
		err = shard.sendIdentify(ctx)
	} else {
		err = shard.sendResume(ctx)
	}

	if err != nil {
		trace(shard.id, "Event", "Failed to reply to hello opcode - received error: %v (identify? = %t)\n", err, shard.sessionID == "")
	}

	shard.mu.RUnlock()
}

func (shard *shard) dispatchEvent(payload EventPacket) {
	switch payload.Event {
	case READY_EVENT:
		var readyData ReadyEventData
		err := json.Unmarshal(payload.Data, &readyData)
		if err != nil {
			trace(shard.id, "Event", "Received request to dispatch READY event but failed to decode json payload - received error: %v (returning early, will likely result in damaged session logic)", err)
			return
		}

		shard.mu.Lock()
		shard.sessionID = readyData.SessionID
		shard.resumeGatewayURL = readyData.ResumeGatewayURL
		trace(shard.id, "Event", "Successfully joined new session. Changed status to ready!")
		shard.state = READY_SHARD_STATE
		shard.mu.Unlock()
	case RESUMED_EVENT:
		shard.mu.Lock()
		shard.state = READY_SHARD_STATE
		shard.mu.Unlock()
		trace(shard.id, "Event", "Successfully resumed session. Changed status to ready!")
	default:
		trace(shard.id, "Event", "Received new, known event: %s - there's no handler for this event!\n%+v\n", payload.Event, payload)
	}
}

// Scans & handles incoming payloads in infinite loop - cancel context to exit early.
func (shard *shard) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			trace(shard.id, "WebSocket", "Cancelled context, requested shutdown.")
			shard.close(true)
			return
		default:
		}

		var payload EventPacket
		err := shard.conn.readJSON(ctx, &payload)
		if err != nil {
			// trace(shard.id, "WebSocket", "Failed to read json payload - received error: %v", err)

			var exitCode ExitCode
			if code := websocket.CloseStatus(err); code == -1 {
				exitCode = SESSION_TIMED_OUT
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
				trace(shard.id, "WebSocket", "Received fatal exit code (%d). Not resumable.", exitCode)
				shard.close(true)
				return
			case INVALID_SEQ, SESSION_TIMED_OUT:
				trace(shard.id, "WebSocket", "Received error code (%d) from expected codes - shard will try to resume session on new connection.", exitCode)
				if err := shard.reset(ctx, false); err != nil {
					trace(shard.id, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
				}
				continue
			default:
				trace(shard.id, "WebSocket", "Received unknown, not typical discord gateway error code (%d) - shard will attempt to open new connection and start brand new session.", exitCode)
				if err := shard.reset(ctx, true); err != nil {
					trace(shard.id, "WebSocket", "Attempted to open new connection & create new session but failed - received error: %v", err)
				}
				continue
			}
		}

		if payload.Sequence != 0 {
			shard.lastSequence.Store(payload.Sequence)
		}

		switch payload.Opcode {
		case DISPATCH_OPCODE:
			trace(shard.id, "Event", "Received dispatch opcode with %s event name.", payload.Event)
			shard.dispatchEvent(payload)
		case HEARTBEAT_OPCODE:
			trace(shard.id, "Event", "Received heartbeat opcode request - sending manual heartbeat.")
			shard.hb.manualBeat(ctx)
		case RECONNECT_OPCODE:
			trace(shard.id, "Event", "Received reconnect opcode request - closing & reopening websocket connection within same session.")
			if err := shard.reset(ctx, false); err != nil {
				trace(shard.id, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
			}

			continue
		case INVALID_SESSION_OPCODE:
			var canResume bool
			if err := json.Unmarshal(payload.Data, &canResume); err != nil {
				trace(shard.id, "Event", "Received invalid session opcode request but failed to decode json payload - unable to verify whether session can be resumed! (assuming not, new session is needed)")
			}

			if canResume {
				trace(shard.id, "Event", "Received invalid session opcode request - closing & reopening websocket connection within same session.")
				if err := shard.reset(ctx, false); err != nil {
					trace(shard.id, "WebSocket", "Attempted to open new connection within same session but failed - received error: %v", err)
				}
			} else {
				trace(shard.id, "Event", "Received invalid session opcode request - closing & reopening websocket connection & creating new session.")
				if err := shard.reset(ctx, true); err != nil {
					trace(shard.id, "WebSocket", "Attempted to open new connection & create new session but failed - received error: %v", err)
				}
			}

			continue
		case HELLO_OPCODE:
			trace(shard.id, "Event", "Received hello opcode - shard will now try to identify/resume.")
			shard.handleHello(ctx, payload.Data)
		case HEARTBEAT_ACK_OPCODE:
			// trace(shard.id, "Event", "Received heartbeat ack code.")
			shard.hb.acknowledged.Store(true)
		default:
			trace(shard.id, "Event", "Received %d opcode - there's no handler for this event!", payload.Opcode)
		}
	}
}
