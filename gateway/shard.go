package gateway

import (
	"context"
	"encoding/json"
	"log"
	"qord/discord"
	"runtime"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Shard struct {
	ID               uint16
	State            ShardState
	sessionID        string
	resumeGatewayURL string
	lastSequence     uint32
	heartbeat        heartbeatState
	manager          *GatewayManager
	conn             *websocket.Conn
}

func (shard *Shard) listen(ctx context.Context) {
	for {
		var payload EventPacket
		err := wsjson.Read(ctx, shard.conn, &payload)
		if err != nil {
			log.Printf("[Shard %d] Read error: %v\n", shard.ID, err)
			return
		}

		if payload.Sequence != 0 {
			shard.lastSequence = payload.Sequence
			log.Printf("[Shard %d] Updated connection sequence to %d\n", shard.ID, payload.Sequence)
		}

		switch payload.Opcode {
		case HELLO_OPCODE:
			shard.handleHello(ctx, payload.Data)
		case HEARTBEAT_OPCODE:
			log.Printf("[Shard %d] Received ask for hearbeat\n", shard.ID)
			shard.sendHeartbeat(ctx, false)
		case HEARTBEAT_ACK_OPCODE:
			log.Printf("[Shard %d] Heartbeat acknowledged\n", shard.ID)
			shard.heartbeat.Acknowledged = true
		case DISPATCH_OPCODE:
			log.Printf("[Shard %d] Received event: %s\n", shard.ID, payload.Event)
			shard.dispatchEvent(ctx, payload)
		case RECONNECT_OPCODE:
			log.Println("Reconnect requested by Discord")
			// TODO: implement resume
		}
	}
}

func (shard *Shard) dispatchEvent(ctx context.Context, payload EventPacket) {
	switch payload.Event {
	case READY_EVENT:
		var readyData ReadyEventData
		err := json.Unmarshal(payload.Data, &readyData)
		if err != nil {
			log.Printf("[Shard %d] Failed to unpack ready event details:\n", shard.ID)
			log.Println(err)
		}

		shard.sessionID = readyData.SessionID
		shard.resumeGatewayURL = readyData.ResumeGatewayURL
		shard.manager.User = readyData.User
		log.Printf("[Shard %d] Changes state to active (operational)\n", shard.ID)
		shard.State = ACTIVE_SHARD_STATE
	}
}

func (shard *Shard) handleHello(ctx context.Context, raw json.RawMessage) {
	log.Printf("[Shard %d] Received Hello OP code!\n", shard.ID)

	var helloData HelloEventData
	err := json.Unmarshal(raw, &helloData)
	if err != nil {
		log.Printf("[Shard %d] Failed to unpack heartbeat interval details:\n", shard.ID)
		log.Println(err)
	}

	shard.heartbeat.Interval = time.Duration(helloData.HeartbeatInterval)
	log.Printf("[Shard %d] Received heartbeat interval to %dms\n", shard.ID, shard.heartbeat.Interval)
	shard.heartbeat.Acknowledged = true
	shard.sendHeartbeat(ctx, true)

	if shard.sessionID == "" {
		err := wsjson.Write(ctx, shard.conn, IdentifyEvent{
			Opcode: IDENTIFY_OPCODE,
			Data: IdentifyPayloadData{
				Token:          shard.manager.Token,
				Intents:        shard.manager.Intents,
				ShardOrder:     [2]uint16{shard.ID, shard.manager.ShardCount},
				LargeThreshold: 50,
				Properties: IdentifyPayloadDataProperties{
					OS:      runtime.GOOS,
					Browser: discord.LIBRARY_NAME,
					Device:  discord.LIBRARY_NAME,
				},
			},
		})

		if err != nil {
			log.Printf("[Shard %d] Failed to send back hello payload!\n", shard.ID)
			log.Println(err)
		}
	} else {
		log.Printf("[Shard %d] Hello OP code is resume request!\n", shard.ID)
	}
}

func (shard *Shard) sendHeartbeat(ctx context.Context, autoLoop bool) {
	if !shard.heartbeat.Acknowledged {
		log.Printf("[Shard %d] Failed to acknowledge previous heartbeat. Assuming zombified connection. Should reconnect!\n", shard.ID)
		// shard.Close()
		// panic("TODO: add reconnect logic")
	}

	shard.heartbeat.Acknowledged = false
	shard.heartbeat.LastBeat = time.Now()

	log.Printf("[Shard %d] Sending heartbeat! (last sequence = %d)\n", shard.ID, shard.lastSequence)
	err := wsjson.Write(ctx, shard.conn, HeartbeatEvent{Opcode: HEARTBEAT_OPCODE, Sequence: shard.lastSequence})
	if err != nil {
		panic(err)
	}

	if autoLoop {
		log.Printf("[Shard %d] Requested automatic loop on heartbeat!\n", shard.ID)
		shard.heartbeat.Timer = time.AfterFunc(shard.heartbeat.Interval, func() { shard.sendHeartbeat(ctx, false) })
	}
}

func (shard *Shard) Close() {
	log.Printf("[Shard %d] Requested shard shutdown!\n", shard.ID)
	shard.State = UNAVAILABLE_SHARD_STATE
	shard.conn.Close(websocket.StatusNormalClosure, "shutdown")
}
