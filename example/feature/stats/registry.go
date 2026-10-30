package stats

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires /stats, sharing the Stats created in main with the command hook.
func Register(client *tempest.BaseClient, gateway *tempest.GatewayClient, stats *Stats) ([]tempest.Command, error) {
	cmd := newStatsCommand(client, gateway, stats)

	definition := cmd.Definition()
	if err := client.RegisterCommand(definition); err != nil {
		return nil, fmt.Errorf("register /stats: %w", err)
	}

	return []tempest.Command{definition}, nil
}
