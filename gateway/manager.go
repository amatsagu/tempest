package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"qord/discord"
	"qord/other"
	"sync"
	"time"
)

type GatewayManager struct {
	token                string
	shards               []*Shard
	shardCount           uint16
	started              bool
	shardStateUpdateHook func(ID uint16, state ShardState)
	shardErrorHook       func(ID uint16, err error)
}

type GatewayManagerOptions struct {
	Token                string
	ShardCount           uint16 // Set to 0 if you prefer auto sharding. Each shard can handle max 2500 servers but it's recommended to keep ~70% max load on each shard (about 1750 servers).
	ShardStateUpdateHook func(ID uint16, state ShardState)
	ShardErrorHook       func(ID uint16, err error)
}

func NewGatewayManager(opt GatewayManagerOptions) GatewayManager {
	_, err := other.ExtractUserIDFromToken(opt.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	if opt.ShardCount > 1500 {
		panic("tried to create absurd amount of shards, please set realistic number")
	}

	return GatewayManager{
		token:                opt.Token,
		shardCount:           opt.ShardCount,
		shardStateUpdateHook: opt.ShardStateUpdateHook,
		shardErrorHook:       opt.ShardErrorHook,
	}
}

func (manager *GatewayManager) StartSession(ctx context.Context) error {
	if manager.started {
		return errors.New("already started session - any changes are impossible until it's fully stopped")
	}
	manager.started = true

	log.Println("[Manager] Starting session... Starting shards...")

	rData, err := manager.calculateShardCount()
	if err != nil {
		return err
	}

	if manager.shardCount == 0 {
		manager.shardCount = rData.ShardCount
	}

	manager.shards = make([]*Shard, manager.shardCount)

	if rData.SessionStartLimit.MaxConcurrency == 0 {
		rData.SessionStartLimit.MaxConcurrency = 1
	}

	var wg sync.WaitGroup

	for i := uint16(0); i < manager.shardCount; i += rData.SessionStartLimit.MaxConcurrency {
		batchSize := rData.SessionStartLimit.MaxConcurrency
		if manager.shardCount-i < rData.SessionStartLimit.MaxConcurrency {
			batchSize = manager.shardCount - i
		}

		for j := uint16(0); j < batchSize; j++ {
			shardID := i + j
			wg.Add(1)
			go func(id uint16) {
				defer wg.Done()
				manager.spawnShard(ctx, id)
			}(shardID)
		}

		if i+batchSize < manager.shardCount {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
	}

	log.Println("[Manager] Launched all shards.")
	wg.Wait()
	return nil
}

func (manager *GatewayManager) calculateShardCount() (GatewayBot, error) {
	data := GatewayBot{}
	request, err := http.NewRequest("GET", discord.DISCORD_API_URL+"/gateway/bot", nil)
	if err != nil {
		return data, errors.New("failed to create request to calculate recommended shard count: " + err.Error())
	}

	request.Header.Add("Content-Type", discord.CONTENT_TYPE_JSON)
	request.Header.Add("User-Agent", discord.USER_AGENT)
	request.Header.Add("Authorization", "Bot "+manager.token)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return data, errors.New("failed to make request to calculate recommended shard count: " + err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return data, fmt.Errorf("failed to access recommended shard count data from discord, received %d status code", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return data, errors.New("failed to decode discord gateway response: " + err.Error())
	}

	if data.ShardCount == 0 {
		return data, errors.New("looks like discord returned invalid payload where number of recommended shards is not defined")
	}

	return data, nil
}

// Spawns new Shard connection in a sequence and returns it's ID.
func (manager *GatewayManager) spawnShard(ctx context.Context, ID uint16) error {
	shard := Shard{
		ID:      ID,
		manager: manager,
	}

	err := shard.updateConnection(ctx, true, false)
	if err != nil {
		return err
	}

	manager.shards[ID] = &shard
	shard.listen(ctx)
	return nil
}
