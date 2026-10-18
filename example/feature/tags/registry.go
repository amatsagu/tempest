// Package tags shows a slash command with subcommands, in-memory state, and autocomplete.
package tags

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires one handler per /tag subcommand plus autocomplete for the
// subcommands that take a tag name.
func Register(client *tempest.Client) ([]discord.Command, error) {
	tag := newTagCommand()

	routes := []struct {
		name   string
		handle tempest.CommandHandler
	}{
		{"tag add", tag.add},
		{"tag show", tag.show},
		{"tag remove", tag.remove},
		{"tag list", tag.list},
	}

	for _, route := range routes {
		if err := client.Command(route.name, route.handle); err != nil {
			return nil, fmt.Errorf("register /%s: %w", route.name, err)
		}
	}

	for _, name := range []string{"tag show", "tag remove"} {
		if err := client.CommandAutocomplete(name, tag.autocomplete); err != nil {
			return nil, fmt.Errorf("register /%s autocomplete: %w", name, err)
		}
	}

	return []discord.Command{tag.Definition()}, nil
}
