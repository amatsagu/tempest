package tags

import (
	"context"
	"fmt"
	"strings"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

// tagCommand answers the /tag subcommands. Its store is server-scoped state
// shared across every invocation.
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

// handle switches on the invoked subcommand, which Discord sends as the first
// entry of Data.Options, and hands off to one method per subcommand.
func (c *tagCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	if itx.GuildID.Snowflake == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "Tags only work in servers."}, true)
	}

	if len(itx.Data.Options) == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "Pick a subcommand."}, true)
	}

	sub := itx.Data.Options[0]
	switch sub.Name {
	case "add":
		return c.addTag(itx, sub)
	case "show":
		return c.showTag(itx, sub)
	case "remove":
		return c.removeTag(itx, sub)
	case "list":
		return c.listTags(itx)
	default:
		return itx.Reply(discord.ResponseMessageData{Content: "Unknown subcommand."}, true)
	}
}

func (c *tagCommand) addTag(itx *discord.CommandInteraction, sub discord.CommandInteractionOption) error {
	name := strings.TrimSpace(subString(sub, "name"))
	value := subString(sub, "value")

	if name == "" {
		return itx.Reply(discord.ResponseMessageData{Content: "A tag needs a name."}, true)
	}

	c.store.add(itx.GuildID, name, value)
	return itx.Reply(discord.ResponseMessageData{Content: "Saved tag " + name + "."}, true)
}

// showTag replies with the tag's embed. A missing tag is a genuine failure, so
// it is returned as an error and reported through Config.OnInteractionError
// rather than answered here.
func (c *tagCommand) showTag(itx *discord.CommandInteraction, sub discord.CommandInteractionOption) error {
	name := subString(sub, "name")

	value, ok := c.store.get(itx.GuildID, name)
	if !ok {
		return fmt.Errorf("tag %q not found", name)
	}

	card := builder.Embed(name, value)
	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false)
}

func (c *tagCommand) removeTag(itx *discord.CommandInteraction, sub discord.CommandInteractionOption) error {
	name := subString(sub, "name")

	if !c.store.remove(itx.GuildID, name) {
		return itx.Reply(discord.ResponseMessageData{Content: "No tag named " + name + "."}, true)
	}
	return itx.Reply(discord.ResponseMessageData{Content: "Removed tag " + name + "."}, true)
}

func (c *tagCommand) listTags(itx *discord.CommandInteraction) error {
	names := c.store.names(itx.GuildID)
	if len(names) == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "There are no tags in this server yet."}, true)
	}

	return itx.Reply(discord.ResponseMessageData{Content: "Tags: " + strings.Join(names, ", ")}, false)
}

// autocomplete suggests stored tag names, but only for the name option of the
// show and remove subcommands. Anything else returns no choices.
func (c *tagCommand) autocomplete(_ context.Context, itx *discord.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	typed, ok := focusedTagName(itx.Data)
	if !ok {
		return nil, nil
	}
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

// focusedTagName returns the text typed into the focused name option, but only
// when the invoked subcommand is show or remove.
func focusedTagName(data discord.CommandInteractionData) (string, bool) {
	if len(data.Options) == 0 {
		return "", false
	}

	sub := data.Options[0]
	if sub.Name != "show" && sub.Name != "remove" {
		return "", false
	}

	for _, opt := range sub.Options {
		if opt.Focused && opt.Name == "name" {
			typed, _ := opt.Value.(string)
			return typed, true
		}
	}
	return "", false
}

// subString reads a string option nested under a subcommand. Subcommand
// parameters live in the subcommand option's own Options, not at the top level.
func subString(sub discord.CommandInteractionOption, name string) string {
	for _, opt := range sub.Options {
		if opt.Name != name {
			continue
		}
		value, _ := opt.Value.(string)
		return value
	}
	return ""
}
