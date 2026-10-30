// Package commands shows slash commands: options, autocomplete, embeds, ephemeral
// replies, and deferred follow-ups.
package commands

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires the slash command handlers and returns their definitions.
func Register(client *tempest.BaseClient) ([]tempest.Command, error) {
	ping := newPingCommand()
	echo := newEchoCommand()
	embed := newEmbedCommand()
	slow := newSlowCommand()
	color := newColorCommand()

	definitions := []tempest.Command{
		ping.Definition(),
		echo.Definition(),
		embed.Definition(),
		slow.Definition(),
		color.Definition(),
	}

	for _, definition := range definitions {
		if err := client.RegisterCommand(definition); err != nil {
			return nil, fmt.Errorf("register /%s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
