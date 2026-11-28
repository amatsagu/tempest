// Package showcase renders a Components V2 message.
package showcase

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires the showcase command and its button handler.
func Register(client *tempest.Client) ([]tempest.Command, error) {
	showcase := newShowcaseCommand()

	if err := client.RegisterComponent(acknowledgeID, showcase.handleAcknowledge); err != nil {
		return nil, fmt.Errorf("register %s: %w", acknowledgeID, err)
	}

	definition := showcase.Definition()
	if err := client.RegisterCommand(definition); err != nil {
		return nil, fmt.Errorf("register /showcase: %w", err)
	}

	return []tempest.Command{definition}, nil
}
