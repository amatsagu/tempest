package tags

import (
	"context"
	"fmt"
	"strings"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

// tagCommand answers the /tag subcommands, backed by server-scoped state.
type tagCommand struct {
	store *tagStore
}

func newTagCommand() *tagCommand {
	return &tagCommand{store: newTagStore()}
}

func (c *tagCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "tag",
		Description: "Save and recall short text snippets, per server.",
		Options: []discord.CommandOption{
			{
				Type:        discord.OptionSubCommand,
				Name:        "add",
				Description: "Save a tag.",
				Options: []discord.CommandOption{
					{Type: discord.OptionString, Name: "name", Description: "The tag name.", Required: true},
					{Type: discord.OptionString, Name: "value", Description: "The text to save.", Required: true},
				},
			},
			{
				Type:        discord.OptionSubCommand,
				Name:        "show",
				Description: "Show a tag.",
				Options: []discord.CommandOption{
					{Type: discord.OptionString, Name: "name", Description: "The tag to show.", Required: true, Autocomplete: true},
				},
			},
			{
				Type:        discord.OptionSubCommand,
				Name:        "remove",
				Description: "Delete a tag.",
				Options: []discord.CommandOption{
					{Type: discord.OptionString, Name: "name", Description: "The tag to delete.", Required: true, Autocomplete: true},
				},
			},
			{
				Type:        discord.OptionSubCommand,
				Name:        "list",
				Description: "List the tags saved in this server.",
			},
		},
	}
}

// add saves a tag; the subcommand options arrive at the top level.
func (c *tagCommand) add(_ context.Context, itx *discord.CommandInteraction) error {
	if itx.GuildID.Snowflake == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "Tags only work in servers."}, true)
	}

	name, _ := itx.Data.StringOption("name")
	name = strings.TrimSpace(name)
	value, _ := itx.Data.StringOption("value")

	if name == "" {
		return itx.Reply(discord.ResponseMessageData{Content: "A tag needs a name."}, true)
	}

	c.store.add(itx.GuildID, name, value)
	return itx.Reply(discord.ResponseMessageData{Content: "Saved tag " + name + "."}, true)
}

// show replies with the tag's embed; a missing tag is returned as an error for
// Config.OnInteractionError rather than answered here.
func (c *tagCommand) show(_ context.Context, itx *discord.CommandInteraction) error {
	name, _ := itx.Data.StringOption("name")

	value, ok := c.store.get(itx.GuildID, name)
	if !ok {
		return fmt.Errorf("tag %q not found", name)
	}

	card := builder.Embed(name, value)
	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false)
}

func (c *tagCommand) remove(_ context.Context, itx *discord.CommandInteraction) error {
	name, _ := itx.Data.StringOption("name")

	if !c.store.remove(itx.GuildID, name) {
		return itx.Reply(discord.ResponseMessageData{Content: "No tag named " + name + "."}, true)
	}
	return itx.Reply(discord.ResponseMessageData{Content: "Removed tag " + name + "."}, true)
}

func (c *tagCommand) list(_ context.Context, itx *discord.CommandInteraction) error {
	if itx.GuildID.Snowflake == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "Tags only work in servers."}, true)
	}

	names := c.store.names(itx.GuildID)
	if len(names) == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "There are no tags in this server yet."}, true)
	}

	return itx.Reply(discord.ResponseMessageData{Content: "Tags: " + strings.Join(names, ", ")}, false)
}

// autocomplete suggests stored tag names for the focused name option.
func (c *tagCommand) autocomplete(_ context.Context, itx *discord.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	_, raw, ok := itx.Data.Focused()
	if !ok {
		return nil, nil
	}

	typed, _ := raw.(string)
	typed = strings.ToLower(typed)
	names := c.store.names(itx.GuildID)

	choices := make([]discord.CommandOptionChoice, 0, min(len(names), 25))
	for _, name := range names {
		if !strings.HasPrefix(strings.ToLower(name), typed) {
			continue
		}

		choices = append(choices, discord.CommandOptionChoice{Name: name, Value: name})
		if len(choices) == 25 {
			break
		}
	}
	return choices, nil
}
