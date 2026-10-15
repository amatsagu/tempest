// Package tags shows a slash command with subcommands, in-memory state, and autocomplete.
package tags

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the tag command and its autocomplete handler.
func Register(client *tempest.Client) ([]discord.Command, error) {
	tag := newTagCommand()

	if err := client.Command("tag", tag.handle); err != nil {
		return nil, fmt.Errorf("register /tag: %w", err)
	}

	if err := client.CommandAutocomplete("tag", tag.autocomplete); err != nil {
		return nil, fmt.Errorf("register /tag autocomplete: %w", err)
	}

	return []discord.Command{tag.Definition()}, nil
}
