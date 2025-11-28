package tempest

import (
	"context"
	"encoding/json"
	"log"
	"runtime"
	"sync"
	"time"
)

type ShardState uint8

const (
	// Represents any state that cannot be considered ready
	// (offline, dead, zombie connection, just disconnected, etc.).
	OFFLINE_SHARD_STATE ShardState = iota
	CONNECTING_SHARD_STATE
	// State where Shard's socket is connected but still in process of identifying or resuming session.
	AUTHENTICATING_SHARD_STATE
	ONLINE_SHARD_STATE
)

func (s ShardState) String() string {
	switch s {
	case OFFLINE_SHARD_STATE:
		return "OFFLINE"
	case CONNECTING_SHARD_STATE:
		return "CONNECTING"
	case AUTHENTICATING_SHARD_STATE:
		return "AUTHENTICATING"
	case ONLINE_SHARD_STATE:
		return "ONLINE"
	default:
		return "UNKNOWN"
	}
}

// Shard represents a single connection to the Discord Gateway. It handles
// the full lifecycle of the connection, including identifying, heartbeating,
// and resuming. It is designed to be managed by a Manager.
type Shard struct {
	ID           uint16
	totalShards  uint16
	token        string
	intents      uint32
	socket       *socket
	traceLogger  *log.Logger // Inherited from the manager
	eventHandler func(shardID uint16, packet EventPacket)

	// State
	mu                  sync.RWMutex
	sessionID           string
	resumeGatewayURL    string
	lastSequence        uint32
	heartbeatInterval   time.Duration
	heartbeatAckMissing bool
	state               ShardState // New field to track the shard's state
}

// Creates a new Shard instance
// - shard by default will handle own session lifecycle (identify, heartbeat, session resume).
//
// All packets shard receives that are not related to connection lifecycle will be pushed to eventHandler function.
//
// Warning: Shards are intended to be used via Manager. If you don't know what you're doing - use manager instead.
func NewShard(
	id uint16,
	totalShards uint16,
	token string,
	intents uint32,
	traceLogger *log.Logger,
	eventHandler func(shardID uint16, packet EventPacket),
) *Shard {
	return &Shard{
		ID:           id,
		totalShards:  totalShards,
		token:        token,
		intents:      intents,
		socket:       &socket{},
		traceLogger:  traceLogger,
		eventHandler: eventHandler,
		// state:        ShardStateOffline,
	}
}

func (s *Shard) Status() ShardState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Start establishes a connection to the Discord Gateway and starts handling events.
// This is a blocking call that will manage the connection until the context is canceled.
func (s *Shard) Start(ctx context.Context, gatewayURL string) {
	s.tracef("Starting connection loop.")

	for {
		select {
		case <-ctx.Done():
			s.tracef("Context cancellation received. Exiting connection loop.")
			s.socket.close() // Explicitly close the socket.
			s.mu.Lock()
			s.state = OFFLINE_SHARD_STATE
			s.mu.Unlock()
			return
		default:
			s.socket.close()

			s.mu.Lock()
			s.state = CONNECTING_SHARD_STATE
			s.mu.Unlock()
			s.tracef("Changing state to %s.", s.state.String())

			targetURL := gatewayURL
			s.mu.RLock()
			if s.resumeGatewayURL != "" {
				targetURL = s.resumeGatewayURL
			}
			s.mu.RUnlock()

			s.tracef("Attempting to connect to %s", targetURL)
			if err := s.socket.connect(targetURL); err != nil {
				s.tracef("Connection failed: %v. Retrying in 5 seconds.", err)
				time.Sleep(5 * time.Second)
				continue
			}
			s.tracef("WebSocket connection established.")

			s.mu.Lock()
			s.state = AUTHENTICATING_SHARD_STATE
			s.mu.Unlock()
			s.tracef("Changing state to %s.", s.state.String())

			// Create a context for this connection lifecycle to manage the heartbeat loop.
			connCtx, cancelConn := context.WithCancel(ctx)

			go s.heartbeatLoop(connCtx)

			// Start the read loop. This is blocking.
			err := s.readLoop()

			// Stop the heartbeat loop immediately upon disconnection.
			cancelConn()

			s.mu.Lock()
			s.state = OFFLINE_SHARD_STATE
			s.mu.Unlock()
			s.tracef("Disconnected from gateway: %v", err)

			select {
			case <-ctx.Done():
				s.tracef("Context cancellation received. Not reconnecting.")
				return
			default:
				s.tracef("Reconnecting...")
			}
		}
	}
}

func (s *Shard) Close() {
	s.tracef("Closing shard connection.")
	s.socket.close()
	s.mu.Lock()
	s.state = OFFLINE_SHARD_STATE
	s.mu.Unlock()
}

func (s *Shard) Send(jsonPayload any) error {
	return s.socket.writeJSON(jsonPayload)
}

func (s *Shard) tracef(format string, v ...any) {
	s.traceLogger.Printf("[SHARD %d (%s)] "+format, append([]any{s.ID, s.Status()}, v...)...)
}

func (s *Shard) readLoop() error {
	for {
		var packet EventPacket
		if err := s.socket.readJSON(&packet); err != nil {
			return err
		}

		s.tracef("RECV Op: %d, Seq: %d, Event: %s", packet.Opcode, packet.Sequence, packet.Event)

		if packet.Sequence > 0 {
			s.mu.Lock()
			s.lastSequence = packet.Sequence
			s.mu.Unlock()
		}

		if err := s.handlePacket(packet); err != nil {
			s.tracef("HANDLE_EVENT Error: %v", err)
			return err
		}
	}
}

func (s *Shard) handleDispatchEvent(p EventPacket) error {
	switch p.Event {
	case READY_EVENT:
		var ready ReadyEventData
		if err := json.Unmarshal(p.Data, &ready); err != nil {
			return err
		}

		s.mu.Lock()
		s.sessionID = ready.SessionID
		s.resumeGatewayURL = ready.ResumeGatewayURL
		s.state = ONLINE_SHARD_STATE
		s.mu.Unlock()
		s.tracef("Successfully started new session with ID = %s.", ready.SessionID)
	case RESUMED_EVENT:
		s.mu.Lock()
		s.state = ONLINE_SHARD_STATE
		s.mu.Unlock()
		s.tracef("Successfully resumed session.")
	default:
		s.tracef("Received unknown dispatch event: %s (pushing to provided event handler)", p.Event)
		go s.eventHandler(s.ID, p)
	}

	return nil
}

func (s *Shard) handlePacket(p EventPacket) error {
	switch p.Opcode {
	case DISPATCH_OPCODE:
		s.handleDispatchEvent(p)
	case HELLO_OPCODE:
		var hello HelloEventData
		if err := json.Unmarshal(p.Data, &hello); err != nil {
			return err
		}
		s.mu.Lock()
		s.heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
		s.heartbeatAckMissing = false
		s.mu.Unlock()
		s.tracef("HELLO Heartbeat interval set to %s.", s.heartbeatInterval)

		return s.identifyOrResume()
	case HEARTBEAT_ACK_OPCODE:
		s.mu.Lock()
		s.heartbeatAckMissing = false
		s.mu.Unlock()
		s.tracef("HEARTBEAT ACK received.")
	case HEARTBEAT_OPCODE:
		s.tracef("HEARTBEAT Server requested heartbeat.")
		return s.sendHeartbeat()
	case RECONNECT_OPCODE:
		s.tracef("RECONNECT Server requested reconnect. Closing connection to reconnect.")

		// Add a small delay to throttle reconnect attempts
		time.Sleep(1 * time.Second)

		return s.socket.close()
	case INVALID_SESSION_OPCODE:
		var resume bool
		if err := json.Unmarshal(p.Data, &resume); err != nil {
			return err
		}

		if !resume {
			s.mu.Lock()
			s.lastSequence = 0
			s.sessionID = ""
			s.resumeGatewayURL = ""
			s.mu.Unlock()
		}

		// Add a small delay to throttle reconnect attempts
		time.Sleep(1 * time.Second)

		return s.socket.close()
	default:
		s.tracef("Received unknown Opcode: %d", p.Opcode)
	}

	return nil
}

func (s *Shard) identifyOrResume() error {
	s.mu.RLock()
	sessionID := s.sessionID
	s.mu.RUnlock()

	if sessionID == "" {
		return s.sendIdentify()
	}

	return s.sendResume()
}

func (s *Shard) sendIdentify() error {
	s.tracef("IDENTIFY as a new session.")
	payload := IdentifyEvent{
		Opcode: IDENTIFY_OPCODE,
		Data: IdentifyPayloadData{
			Token:      s.token,
			Intents:    s.intents,
			ShardOrder: [2]uint16{s.ID, s.totalShards},
			Properties: IdentifyPayloadDataProperties{
				OS:      runtime.GOOS,
				Browser: "qord",
				Device:  "qord",
			},
		},
	}

	return s.socket.writeJSON(payload)
}

func (s *Shard) sendResume() error {
	s.mu.RLock()
	sessionID := s.sessionID
	seq := s.lastSequence
	s.mu.RUnlock()

	s.tracef("RESUME session ID = %s with sequence = %d.", sessionID, seq)

	payload := ResumeEvent{
		Opcode: RESUME_OPCODE,
		Data: ResumeEventData{
			Token:     s.token,
			SessionID: sessionID,
			Sequence:  seq,
		},
	}

	return s.socket.writeJSON(payload)
}

func (s *Shard) heartbeatLoop(ctx context.Context) {
	s.mu.RLock()
	interval := s.heartbeatInterval
	s.mu.RUnlock()

	if interval == 0 {
		s.tracef("Invalid heartbeat interval (0). Aborting heartbeat loop.")
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.tracef("Starting heartbeat loop.")
	for {
		s.mu.Lock()
		if s.heartbeatAckMissing {
			s.mu.Unlock()
			s.tracef("Zombied connection detected. Reconnecting!")

			// Close the socket to force the main readLoop to exit.
			// This will cause the Start loop to trigger a reconnect.
			s.socket.close()
			return
		}
		s.heartbeatAckMissing = true
		s.mu.Unlock()

		if err := s.sendHeartbeat(); err != nil {
			s.tracef("Failed to send heartbeat: %v", err)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			s.tracef("Context cancellation received. Exiting heartbeat loop.")
			return
		}
	}
}

func (s *Shard) sendHeartbeat() error {
	s.mu.RLock()
	seq := s.lastSequence
	s.mu.RUnlock()

	s.tracef("Sending heartbeat with sequence = %d.", seq)

	payload := HeartbeatEvent{
		Opcode:   HEARTBEAT_OPCODE,
		Sequence: seq,
	}

	return s.socket.writeJSON(payload)
}
