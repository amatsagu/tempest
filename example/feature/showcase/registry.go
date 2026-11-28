package showcase

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

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
