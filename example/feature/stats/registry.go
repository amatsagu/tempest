package stats

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

func Register(client *tempest.Client, stats *Stats) ([]tempest.Command, error) {
	cmd := newStatsCommand(client, stats)

	definition := cmd.Definition()
	if err := client.RegisterCommand(definition); err != nil {
		return nil, fmt.Errorf("register /stats: %w", err)
	}

	return []tempest.Command{definition}, nil
}
