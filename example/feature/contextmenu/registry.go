// Package contextmenu shows the USER and MESSAGE context menu command types.
package contextmenu

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires the context menu handlers; the target arrives as Data.TargetID
// and Data.Resolved.
func Register(client *tempest.BaseClient) ([]tempest.Command, error) {
	userInfo := newUserInfoCommand()
	quote := newQuoteCommand()
	publish := newPublishCommand()

	definitions := []tempest.Command{
		userInfo.Definition(),
		quote.Definition(),
		publish.Definition(),
	}

	for _, definition := range definitions {
		if err := client.RegisterCommand(definition); err != nil {
			return nil, fmt.Errorf("register %s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
