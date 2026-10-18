// Package events shows raw gateway event handling and decoding an event payload.
package events

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"log/slog"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register subscribes a handler per event; registering is the subscription, and
// each handler receives the raw JSON payload.
func Register(client *tempest.Client) {
	client.Event("GUILD_CREATE", func(_ context.Context, shard int, _ string, data jsontext.Value) {
		var guild discord.GuildCreateData
		if err := json.Unmarshal(data, &guild); err != nil {
			slog.Error("guild create payload", "err", err)
			return
		}

		slog.Info("guild became available",
			"shard", shard,
			"guild", guild.ID,
			"name", guild.Name,
			"members", guild.MemberCount,
			"large", guild.Large,
			"community", guild.HasFeature("COMMUNITY"),
		)
	})

	client.Event("GUILD_DELETE", func(_ context.Context, shard int, _ string, data jsontext.Value) {
		var guild discord.UnavailableGuild
		if err := json.Unmarshal(data, &guild); err != nil {
			slog.Error("guild delete payload", "err", err)
			return
		}

		slog.Info("guild was removed", "shard", shard, "guild", guild.ID, "unavailable", guild.Unavailable)
	})
}
