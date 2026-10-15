// Package tags shows a slash command with subcommands and small per-server
// state kept in memory, plus autocomplete for the name options.
package tags

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the tag command and its autocomplete handler. The command's
// store is created by its constructor, so every invocation shares one map.
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
