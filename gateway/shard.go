package gateway

import (
	"context"
	"encoding/json"
	"log"
	"qord/discord"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Shard struct {
	mu               sync.RWMutex
	manager          *GatewayManager
	conn             *websocket.Conn
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
	Acknowledged atomic.Bool
	LastBeat     time.Time
	Interval     time.Duration
	Timer        *time.Timer
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
	ping := time.Since(shard.heartbeat.LastBeat) - shard.heartbeat.Interval
	shard.mu.RUnlock()
	return ping
}

func (shard *Shard) Close(resetSession bool) {
	//log.Printf("[Shard %02d] Requested shard shutdown!\n", shard.ID)
	shard.mu.Lock()

	shard.state = UNAVAILABLE_SHARD_STATE
	if shard.conn != nil {
		err := shard.conn.Close(websocket.StatusNormalClosure, "shutdown")
		if err != nil {
			log.Printf("[Shard %02d] Error closing WebSocket: %v\n", shard.ID, err)
		}
		shard.conn = nil
	}

	if resetSession {
		shard.lastSequence = 0
		shard.sessionID = ""
		shard.resumeGatewayURL = ""
	}

	shard.heartbeat.Acknowledged.Store(true)
	shard.heartbeat.LastBeat = time.Time{}
	if shard.heartbeat.Timer != nil {
		if !shard.heartbeat.Timer.Stop() {
			select {
			case <-shard.heartbeat.Timer.C:
			default:
			}
		}
		shard.heartbeat.Timer = nil
	}
	shard.heartbeat.Interval = 0

	shard.mu.Unlock()
}

func (shard *Shard) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Shard %02d] Context canceled, shutting down listener.\n", shard.ID)
			shard.Close(true)
			return
		default:
		}

		var payload EventPacket
		err := wsjson.Read(ctx, shard.conn, &payload)
		if err != nil {
			log.Printf("[Shard %02d] Read error: %v\n", shard.ID, err)

			exitCode := websocket.CloseStatus(err)
			log.Printf("[Shard %02d] Detected exit code: %d", shard.ID, exitCode)

			switch ExitCode(websocket.CloseStatus(err)) {
			case AUTHENTICATION_FAILED,
				INVALID_SHARD,
				SHARDING_REQUIRED,
				INVALID_API_VERSION,
				INVALID_INTENT,
				DISALLOWED_INTENT:
				log.Printf("[Shard %02d] Received %d fatal exit code, shard will shutdown as it's non-recoverable signal.", shard.ID, exitCode)
				shard.Close(true)
				return
			case INVALID_SEQ, SESSION_TIMED_OUT:
				log.Printf("[Shard %02d] Received %d exit code, shard will try to open new session.", shard.ID, exitCode)
				if err := shard.updateConnection(ctx, true, false); err != nil {
					log.Printf("[Shard %02d] Failed to update connection: %v", shard.ID, err)
				}
				continue
			default:
				log.Printf("[Shard %02d] Received %d exit code, shard will try to re-open connection & resume session.", shard.ID, exitCode)
				if err := shard.updateConnection(ctx, true, true); err != nil {
					log.Printf("[Shard %02d] Failed to update connection: %v", shard.ID, err)
				}
				continue
			}
		}

		if payload.Sequence != 0 {
			shard.lastSequence = payload.Sequence
			// log.Printf("[Shard %02d] Updated connection sequence to %d\n", shard.ID, payload.Sequence)
		}

		// log.Printf("[Shard %02d] Detected %s of latency from host to Discord Gateway.\n", shard.ID, shard.Latency().String())

		switch payload.Opcode {
		case DISPATCH_OPCODE:
			// log.Printf("[Shard %02d] Received event: %s\n", shard.ID, payload.Event)
			shard.dispatchEvent(ctx, payload)
		case HEARTBEAT_OPCODE:
			// log.Printf("[Shard %02d] Received ask for hearbeat\n", shard.ID)
			// log.Printf("[Shard %02d] [TRACKER] Trying to start hearbeat from heartbeat discord request opcode (acknowledged = %t)!\n", shard.ID, shard.heartbeat.Acknowledged.Load())
			shard.sendHeartbeat(ctx, false)
		case RECONNECT_OPCODE:
			log.Printf("[Shard %02d] Received reconnect request from Discord. Reconnecting within session!\n", shard.ID)
			if err := shard.updateConnection(ctx, true, false); err != nil {
				log.Printf("[Shard %02d] Failed to update connection: %v", shard.ID, err)
			}

			continue
		case INVALID_SESSION_OPCODE:
			var canResume bool
			if err := json.Unmarshal(payload.Data, &canResume); err != nil {
				log.Printf("[Shard %02d] Failed to parse invalid session flag\n", shard.ID)
			}

			if !canResume {
				log.Printf("[Shard %02d] Session invalidated and cannot resume. Re-identifying.\n", shard.ID)
				if err := shard.updateConnection(ctx, true, true); err != nil {
					log.Printf("[Shard %02d] Failed to update connection: %v", shard.ID, err)
				}
			} else {
				log.Printf("[Shard %02d] Session invalidated but can resume. Attempting resume...\n", shard.ID)
				if err := shard.updateConnection(ctx, true, false); err != nil {
					log.Printf("[Shard %02d] Failed to update connection: %v", shard.ID, err)
				}
			}

			continue
		case HELLO_OPCODE:
			shard.handleHello(ctx, payload.Data)
		case HEARTBEAT_ACK_OPCODE:
			// log.Printf("[Shard %02d] Heartbeat acknowledged\n", shard.ID)
			shard.heartbeat.Acknowledged.Store(true)
		}
	}
}

func (shard *Shard) dispatchEvent(ctx context.Context, payload EventPacket) {
	switch payload.Event {
	case READY_EVENT:
		var readyData ReadyEventData
		err := json.Unmarshal(payload.Data, &readyData)
		if err != nil {
			log.Printf("[Shard %02d] Failed to unpack ready event details:\n", shard.ID)
			log.Println(err)
		}

		shard.mu.Lock()
		shard.sessionID = readyData.SessionID
		shard.resumeGatewayURL = readyData.ResumeGatewayURL
		// shard.manager.User = readyData.User
		log.Printf("[Shard %02d] Changes state to active (operational)\n", shard.ID)
		shard.state = READY_SHARD_STATE
		shard.mu.Unlock()
	case RESUMED_EVENT:
		log.Printf("[Shard %02d] Successfully resumed session.\n", shard.ID)
	}
}

func (shard *Shard) handleHello(ctx context.Context, raw json.RawMessage) {
	var helloData HelloEventData
	err := json.Unmarshal(raw, &helloData)
	if err != nil {
		log.Printf("[Shard %02d] Failed to unpack heartbeat interval details:\n", shard.ID)
		log.Println(err)
	}

	shard.mu.Lock()
	shard.heartbeat.Interval = time.Duration(helloData.HeartbeatInterval)
	shard.heartbeat.Acknowledged.Store(true)
	shard.mu.Unlock()
	//log.Printf("[Shard %02d] [TRACKER] Trying to start hearbeat from Hello handler (acknowledged = %t)!\n", shard.ID, shard.heartbeat.Acknowledged.Load())
	shard.sendHeartbeat(ctx, true)

	shard.mu.RLock()
	if shard.sessionID == "" {
		err := wsjson.Write(ctx, shard.conn, IdentifyEvent{
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
			shard.mu.RUnlock()
			log.Printf("[Shard %02d] Failed to send back hello payload!\n", shard.ID)
			log.Println(err)
		}

		log.Printf("[Shard %02d] Identified on hello code.\n", shard.ID)
		shard.mu.RUnlock()
		return
	} else {
		err := wsjson.Write(ctx, shard.conn, ResumeEvent{
			Opcode: RESUME_OPCODE,
			Data: ResumeEventData{
				Token:     shard.manager.token,
				SessionID: shard.sessionID,
				Sequence:  shard.lastSequence,
			},
		})

		if err != nil {
			shard.mu.RUnlock()
			log.Printf("[Shard %02d] Failed to send back resume payload!\n", shard.ID)
			log.Println(err)
		}

		log.Printf("[Shard %02d] Resumed session with sID = %s.\n", shard.ID, shard.sessionID)
		shard.mu.RUnlock()
	}
}

func (shard *Shard) sendHeartbeat(ctx context.Context, autoLoop bool) {
	if !shard.heartbeat.Acknowledged.Load() {
		log.Printf("[Shard %02d] Failed to acknowledge previous heartbeat. Assuming zombified connection. Should reconnect!\n", shard.ID)
		if err := shard.updateConnection(ctx, true, true); err != nil {
			log.Printf("[Shard %02d] Failed to update connection: %v", shard.ID, err)
		}
		return
	}

	shard.mu.Lock()

	if autoLoop {
		//log.Printf("[Shard %02d] Requested automatic loop on heartbeat!\n", shard.ID)

		// Clean up existing timer if it exists
		if shard.heartbeat.Timer != nil {
			if !shard.heartbeat.Timer.Stop() {
				select {
				case <-shard.heartbeat.Timer.C:
				default:
				}
			}
			shard.heartbeat.Timer = nil
		}

		timer := time.NewTimer(shard.heartbeat.Interval)
		shard.heartbeat.Timer = timer

		go func(t *time.Timer) {
			select {
			case <-t.C:
				//log.Printf("[Shard %02d] [TRACKER] Trying to start hearbeat from heartbeat ticker loop (acknowledged = %t)!\n", shard.ID, shard.heartbeat.Acknowledged.Load())
				shard.sendHeartbeat(ctx, false)
			case <-ctx.Done():
			}
		}(timer)
	} else {
		shard.heartbeat.Acknowledged.Store(false)
		shard.heartbeat.LastBeat = time.Now()

		//log.Printf("[Shard %02d] Sending heartbeat! (last sequence = %d)\n", shard.ID, shard.lastSequence)
		err := wsjson.Write(ctx, shard.conn, HeartbeatEvent{Opcode: HEARTBEAT_OPCODE, Sequence: shard.lastSequence})
		if err != nil {
			log.Printf("[Shard %02d] Failed to send heartbeart response: %s\n", shard.ID, err.Error())
		}
	}

	shard.mu.Unlock()
}

func (shard *Shard) updateConnection(ctx context.Context, restart, resetSession bool) error {
	// log.Printf("[Shard %02d] Requested connection update! (restart = %t, reset session = %t)\n", shard.ID, restart, resetSession)

	if shard.State() != UNAVAILABLE_SHARD_STATE {
		shard.Close(resetSession)
		//time.Sleep(time.Second * 2) // Safety wait before reconnect (probably not needed but just in case)
	}

	if restart {
		var (
			conn *websocket.Conn
			err  error
		)

		shard.mu.Lock()
		shard.state = CONNECTING_SHARD_STATE
		shard.heartbeat.Acknowledged.Store(true)

		if shard.resumeGatewayURL == "" {
			conn, _, err = websocket.Dial(ctx, discord.DISCORD_GATEWAY_URL, nil)
		} else {
			conn, _, err = websocket.Dial(ctx, shard.resumeGatewayURL, nil)
		}

		if err != nil {
			shard.mu.Unlock()
			return err
		}

		shard.conn = conn
	}
	shard.mu.Unlock()

	return nil
}
