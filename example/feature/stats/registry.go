package stats

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires /stats. The Stats value is created in main so the same one can
// be handed to the client's lifecycle hooks; Register only needs the client (to
// time a REST call and read gateway latency) and that shared value.
func Register(client *tempest.Client, stats *Stats) ([]discord.Command, error) {
	cmd := newStatsCommand(client, stats)

	if err := client.Command("stats", cmd.handle); err != nil {
		return nil, fmt.Errorf("register /stats: %w", err)
	}

	return []discord.Command{cmd.Definition()}, nil
}
