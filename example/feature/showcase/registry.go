// Package showcase renders a Components V2 message: containers, sections, text
// displays, separators, media galleries, and the flag that enables them.
package showcase

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

const acknowledgeID = "showcase:ack"

// Register wires the showcase command and its button handler.
func Register(client *tempest.Client) ([]discord.Command, error) {
	showcase := newShowcaseCommand()

	if err := client.Component(acknowledgeID, showcase.handleAcknowledge); err != nil {
		return nil, fmt.Errorf("register %s: %w", acknowledgeID, err)
	}
	if err := client.Command("showcase", showcase.handle); err != nil {
		return nil, fmt.Errorf("register /showcase: %w", err)
	}

	return []discord.Command{showcase.Definition()}, nil
}
