package gateway

import (
	"context"
	"qord/discord"
	"time"

	"github.com/coder/websocket"
)

type GatewayManager struct {
	Token      string
	Intents    uint32
	ShardCount uint16
	Shards     []Shard
}

type ShardState uint8

const (
	UNAVAILABLE_SHARD_STATE ShardState = iota
	CONNECTING_SHARD_STATE
	OPERATIONAL_SHARD_STATE
)

type heartbeatState struct {
	Acknowledged bool
	LastBeat     time.Time
	Timer        *time.Timer
	Interval     time.Duration
}

func (manager *GatewayManager) StartSession(ctx context.Context) {
	manager.SpawnShard(ctx, 0)
}

// Spawns new Shard connection in a sequence and returns it's ID.
func (manager *GatewayManager) SpawnShard(ctx context.Context, ID uint16) error {
	conn, _, err := websocket.Dial(ctx, discord.DISCORD_GATEWAY_URL, nil)
	if err != nil {
		return err
	}

	shard := Shard{
		ID:    uint16(ID),
		State: UNAVAILABLE_SHARD_STATE,
		heartbeat: heartbeatState{
			Acknowledged: true,
		},
		manager: manager,
		conn:    conn,
	}

	manager.Shards = append(manager.Shards, shard)
	shard.listen()
	return nil
}
