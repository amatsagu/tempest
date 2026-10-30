// Package events shows one raw gateway event handler.
//
// GatewayClient accepts a single CustomEventHandler, so this package returns one
// function with a switch over the events the example cares about. Add new cases
// there when a feature needs another gateway event.
package events

import (
	"encoding/json"
	"log"

	"github.com/amatsagu/tempest"
)

// NewHandler returns the CustomEventHandler passed to GatewayClientOptions.
// It runs on the shard's read goroutine, so keep each case quick.
func NewHandler(logger *log.Logger) func(shardID uint16, packet tempest.EventPacket) {
	if logger == nil {
		logger = log.Default()
	}

	return func(shardID uint16, packet tempest.EventPacket) {
		switch packet.Event {
		case tempest.GUILD_CREATE_EVENT:
			handleGuildCreate(logger, shardID, packet.Data)
		case tempest.GUILD_DELETE_EVENT:
			handleGuildDelete(logger, shardID, packet.Data)
		case tempest.ENTITLEMENT_CREATE_EVENT,
			tempest.ENTITLEMENT_UPDATE_EVENT,
			tempest.ENTITLEMENT_DELETE_EVENT:
			handleEntitlement(logger, shardID, packet)
		}
	}
}

func handleGuildCreate(logger *log.Logger, shardID uint16, data json.RawMessage) {
	var guild tempest.CreateGuildEventData
	if err := json.Unmarshal(data, &guild); err != nil {
		logger.Printf("decode GUILD_CREATE on shard %d: %v", shardID, err)
		return
	}

	logger.Printf(
		"GUILD_CREATE shard=%d guild=%s name=%q members=%d large=%t features=%v",
		shardID, guild.ID, guild.Name, guild.MemberCount, guild.Large, guild.Features,
	)
}

func handleGuildDelete(logger *log.Logger, shardID uint16, data json.RawMessage) {
	var guild tempest.UnavailableGuild
	if err := json.Unmarshal(data, &guild); err != nil {
		logger.Printf("decode GUILD_DELETE on shard %d: %v", shardID, err)
		return
	}

	logger.Printf("GUILD_DELETE shard=%d guild=%s unavailable=%t", shardID, guild.ID, guild.Unavailable)
}

// handleEntitlement logs the entitlement lifecycle. Fulfilment belongs here (or
// in a quick handoff to a worker): a real app grants the item on create/update
// and then consumes the entitlement if its SKU is consumable.
func handleEntitlement(logger *log.Logger, shardID uint16, packet tempest.EventPacket) {
	var entitlement tempest.Entitlement
	if err := json.Unmarshal(packet.Data, &entitlement); err != nil {
		logger.Printf("decode %s on shard %d: %v", packet.Event, shardID, err)
		return
	}

	logger.Printf(
		"%s shard=%d id=%s sku=%s type=%d user=%s guild=%s deleted=%t consumed=%t",
		packet.Event, shardID, entitlement.ID, entitlement.SkuID, entitlement.Type,
		entitlement.UserID, entitlement.GuildID, entitlement.Deleted, entitlement.Consumed,
	)
}
