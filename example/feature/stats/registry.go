package stats

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires /stats, sharing the Stats created in main with the hooks.
func Register(client *tempest.Client, stats *Stats) ([]discord.Command, error) {
	cmd := newStatsCommand(client, stats)

	if err := client.Command("stats", cmd.handle); err != nil {
		return nil, fmt.Errorf("register /stats: %w", err)
	}

	return []discord.Command{cmd.Definition()}, nil
}
