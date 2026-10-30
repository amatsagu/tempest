// Package tags shows a slash command with subcommands, in-memory state, and
// autocomplete.
package tags

import (
	"fmt"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// Register wires one handler per /tag subcommand plus autocomplete for the
// subcommands that take a tag name.
func Register(client *tempest.BaseClient) ([]tempest.Command, error) {
	tag := newTagCommand()
	root := tag.Definition()

	if err := client.RegisterCommand(root); err != nil {
		return nil, fmt.Errorf("register /tag: %w", err)
	}

	add := tempest.Command{
		Name:        "add",
		Description: "Save a tag.",
		Options: []tempest.CommandOption{
			{Type: tempest.STRING_OPTION_TYPE, Name: "name", Description: "The tag name.", Required: true},
			{Type: tempest.STRING_OPTION_TYPE, Name: "value", Description: "The text to save.", Required: true},
		},
		SlashCommandHandler: util.CommandHandler(tag.add),
	}

	show := tempest.Command{
		Name:        "show",
		Description: "Show a tag.",
		Options: []tempest.CommandOption{
			{Type: tempest.STRING_OPTION_TYPE, Name: "name", Description: "The tag to show.", Required: true, AutoComplete: true},
		},
		AutoCompleteHandler: tag.autocomplete,
		SlashCommandHandler: util.CommandHandler(tag.show),
	}

	remove := tempest.Command{
		Name:        "remove",
		Description: "Delete a tag.",
		Options: []tempest.CommandOption{
			{Type: tempest.STRING_OPTION_TYPE, Name: "name", Description: "The tag to delete.", Required: true, AutoComplete: true},
		},
		AutoCompleteHandler: tag.autocomplete,
		SlashCommandHandler: util.CommandHandler(tag.remove),
	}

	list := tempest.Command{
		Name:                "list",
		Description:         "List the tags saved in this server.",
		SlashCommandHandler: util.CommandHandler(tag.list),
	}

	definitions := []tempest.Command{root, add, show, remove, list}
	for _, definition := range definitions[1:] {
		if err := client.RegisterSubCommand(definition, "tag"); err != nil {
			return nil, fmt.Errorf("register /tag %s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
