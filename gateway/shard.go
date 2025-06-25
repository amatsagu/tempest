package gateway

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"qord/discord"
	"runtime"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Shard struct {
	ID           uint16
	State        ShardState
	SessionID    string
	LastSequence uint32
	heartbeat    heartbeatState
	manager      *GatewayManager
	conn         *websocket.Conn
}

func (shard *Shard) listen() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	for {
		var payload EventPacket
		err := wsjson.Read(ctx, shard.conn, &payload)
		if err != nil {
			log.Printf("[Shard %d] Read error: %v\n", shard.ID, err)
			return
		}

		if payload.Sequence != 0 {
			shard.LastSequence = payload.Sequence
		}

		log.Printf("[DEBUG PREVIEW] %+v\n", payload)

		switch payload.Opcode {
		case HELLO_OPCODE:
			shard.HandleHello(ctx, payload.Data)
		case HEARTBEAT_ACK_OPCODE:
			log.Printf("[Shard %d] Heartbeat acknowledged\n", shard.ID)
			shard.heartbeat.Acknowledged = true
		case DISPATCH_OPCODE:
			log.Printf("[Shard %d] Received event: %s\n", shard.ID, payload.Event)

		case RECONNECT_OPCODE:
			log.Println("Reconnect requested by Discord")
			// TODO: implement resume
		}
	}
}

func (shard *Shard) HandleHello(ctx context.Context, raw json.RawMessage) {
	log.Printf("[Shard %d] Received Hello OP code!\n", shard.ID)

	var helloData HelloEvent
	err := json.Unmarshal(raw, &helloData)
	if err != nil {
		log.Printf("[Shard %d] Failed to unpack heartbeat interval details:\n", shard.ID)
		log.Println(err)
	}

	shard.heartbeat.Interval = time.Duration(helloData.HeartbeatInterval) * time.Millisecond
	shard.heartbeat.Acknowledged = true
	shard.SendHeartbeat(ctx, true)

	if shard.SessionID == "" {
		err := wsjson.Write(ctx, shard.conn, IdentifyPayload{
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

func (shard *Shard) SendHeartbeat(ctx context.Context, autoLoop bool) {
	if !shard.heartbeat.Acknowledged {
		log.Printf("[Shard %d] Failed to acknowledge previous heartbeat. Assuming zombified connection. Should reconnect!\n", shard.ID)
		shard.Close()
		panic("TODO: add reconnect logic")
	}

	shard.heartbeat.Acknowledged = false
	shard.heartbeat.LastBeat = time.Now()

	log.Printf("[Shard %d] Sending heartbeat! (last sequence = %d)\n", shard.ID, shard.LastSequence)
	wsjson.Write(ctx, shard.conn, EventPacket{Opcode: HEARTBEAT_OPCODE, Data: binary.BigEndian.AppendUint32(nil, shard.LastSequence)})
	if autoLoop {
		log.Printf("[Shard %d] Requested automatic loop on heartbeat!\n", shard.ID)
		shard.heartbeat.Timer = time.AfterFunc(shard.heartbeat.Interval, func() { shard.SendHeartbeat(ctx, false) })
	}
}

func (shard *Shard) Close() {
	log.Printf("[Shard %d] Requested shard shutdown!\n", shard.ID)
	shard.State = UNAVAILABLE_SHARD_STATE
	shard.conn.Close(websocket.StatusNormalClosure, "shutdown")
}
