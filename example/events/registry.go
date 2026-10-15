// Package events shows raw gateway event handling: subscribing to named events
// and reading an event's verbatim payload.
package events

import (
	"context"
	"encoding/json/jsontext"
	"log/slog"

	"github.com/amatsagu/tempest"
)

// Names lists the extra gateway events this bot subscribes to. They are passed
// to Config.ExtraEvents before the client is built.
var Names = []string{"GUILD_CREATE", "GUILD_DELETE"}

// Register subscribes a handler per event. Handlers receive the raw JSON payload
// as jsontext.Value; decode it yourself only if you need its contents.
func Register(client *tempest.Client) {
	client.Event("GUILD_CREATE", func(_ context.Context, shard int, _ string, data jsontext.Value) {
		slog.Info("guild became available", "shard", shard, "bytes", len(data))
	})

	client.Event("GUILD_DELETE", func(_ context.Context, shard int, _ string, _ jsontext.Value) {
		slog.Info("guild was removed", "shard", shard)
	})
}
