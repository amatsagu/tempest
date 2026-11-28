package tags

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

func Register(client *tempest.Client) ([]tempest.Command, error) {
	tag := newTagCommand()
	root := tag.Definition()

	if err := client.RegisterCommand(root); err != nil {
		return nil, fmt.Errorf("register /tag: %w", err)
	}

	add := tempest.Command{
		Name:        "add",
		Description: "Save a tag.",
		Options: []discord.CommandOption{
			{Type: discord.StringOptionType, Name: "name", Description: "The tag name.", Required: true},
			{Type: discord.StringOptionType, Name: "value", Description: "The text to save.", Required: true},
		},
		SlashCommandHandler: tag.add,
	}

	show := tempest.Command{
		Name:        "show",
		Description: "Show a tag.",
		Options: []discord.CommandOption{
			{Type: discord.StringOptionType, Name: "name", Description: "The tag to show.", Required: true, AutoComplete: true},
		},
		AutoCompleteHandler: tag.autocomplete,
		SlashCommandHandler: tag.show,
	}

	remove := tempest.Command{
		Name:        "remove",
		Description: "Delete a tag.",
		Options: []discord.CommandOption{
			{Type: discord.StringOptionType, Name: "name", Description: "The tag to delete.", Required: true, AutoComplete: true},
		},
		AutoCompleteHandler: tag.autocomplete,
		SlashCommandHandler: tag.remove,
	}

	list := tempest.Command{
		Name:                "list",
		Description:         "List the tags saved in this server.",
		SlashCommandHandler: tag.list,
	}

	adminClear := tempest.Command{
		Name:                "clear",
		Description:         "Clear all tags saved in this server.",
		SlashCommandHandler: tag.adminClear,
	}

	definitions := []tempest.Command{root, add, show, remove, list}
	for _, definition := range definitions[1:] {
		if err := client.RegisterSubCommand(definition, "tag"); err != nil {
			return nil, fmt.Errorf("register /tag %s: %w", definition.Name, err)
		}
	}

	if err := client.RegisterSubCommand(adminClear, "tag", "admin"); err != nil {
		return nil, fmt.Errorf("register /tag admin clear: %w", err)
	}

	return append(definitions, adminClear), nil
}
