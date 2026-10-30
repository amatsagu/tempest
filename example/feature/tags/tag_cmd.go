package tags

import (
	"fmt"
	"strings"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// tagCommand answers the /tag subcommands, backed by server-scoped state.
type tagCommand struct {
	store *tagStore
}

func newTagCommand() *tagCommand {
	return &tagCommand{store: newTagStore()}
}

// Definition is the root /tag command. The admin group lives here; the flat
// subcommands are registered separately because master's router handles one
// subcommand level.
func (c *tagCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:        "tag",
		Description: "Save and recall short text snippets, per server.",
		Options: []tempest.CommandOption{
			{
				Type:        tempest.SUB_COMMAND_GROUP_OPTION_TYPE,
				Name:        "admin",
				Description: "Manage server tags.",
				Options: []tempest.CommandOption{
					{
						Type:        tempest.SUB_COMMAND_OPTION_TYPE,
						Name:        "clear",
						Description: "Clear all tags saved in this server.",
					},
				},
			},
		},
		SlashCommandHandler: util.CommandHandler(c.handleGroup),
	}
}

// handleGroup routes the one subcommand group master does not strip for us.
func (c *tagCommand) handleGroup(itx *tempest.CommandInteraction) error {
	if len(itx.Data.Options) == 0 {
		return itx.SendLinearReply("Pick a /tag subcommand.", true)
	}

	group := itx.Data.Options[0]
	if group.Name != "admin" || len(group.Options) == 0 {
		return itx.SendLinearReply("Unknown /tag subcommand.", true)
	}

	switch group.Options[0].Name {
	case "clear":
		return c.adminClear(itx)
	default:
		return itx.SendLinearReply("Unknown /tag admin subcommand.", true)
	}
}

// add saves a tag; the subcommand options arrive at the top level.
func (c *tagCommand) add(itx *tempest.CommandInteraction) error {
	if itx.GuildID == 0 {
		return itx.SendLinearReply("Tags only work in servers.", true)
	}

	name, _ := util.StringOption(itx, "name")
	name = strings.TrimSpace(name)
	value, _ := util.StringOption(itx, "value")

	if name == "" {
		return itx.SendLinearReply("A tag needs a name.", true)
	}

	c.store.add(itx.GuildID, name, value)
	return itx.SendLinearReply("Saved tag "+name+".", true)
}

// show replies with the tag's embed; a missing tag is returned as an error so the
// wrapper logs it.
func (c *tagCommand) show(itx *tempest.CommandInteraction) error {
	name, _ := util.StringOption(itx, "name")

	value, ok := c.store.get(itx.GuildID, name)
	if !ok {
		return fmt.Errorf("tag %q not found", name)
	}

	card := util.Embed(name, value)
	return itx.SendReply(tempest.ResponseMessageData{Embeds: []tempest.Embed{card}}, false, nil)
}

func (c *tagCommand) remove(itx *tempest.CommandInteraction) error {
	name, _ := util.StringOption(itx, "name")

	if !c.store.remove(itx.GuildID, name) {
		return itx.SendLinearReply("No tag named "+name+".", true)
	}
	return itx.SendLinearReply("Removed tag "+name+".", true)
}

func (c *tagCommand) list(itx *tempest.CommandInteraction) error {
	if itx.GuildID == 0 {
		return itx.SendLinearReply("Tags only work in servers.", true)
	}

	names := c.store.names(itx.GuildID)
	if len(names) == 0 {
		return itx.SendLinearReply("There are no tags in this server yet.", true)
	}

	return itx.SendLinearReply("Tags: "+strings.Join(names, ", "), false)
}

// adminClear handles the "tag admin clear" subcommand group route.
func (c *tagCommand) adminClear(itx *tempest.CommandInteraction) error {
	if itx.GuildID == 0 {
		return itx.SendLinearReply("Tags only work in servers.", true)
	}

	c.store.clear(itx.GuildID)
	return itx.SendLinearReply("Cleared all tags in this server.", true)
}

// autocomplete suggests stored tag names for the focused name option.
func (c *tagCommand) autocomplete(itx tempest.CommandInteraction) []tempest.CommandOptionChoice {
	typed := focusedString(itx)
	typed = strings.ToLower(typed)
	names := c.store.names(itx.GuildID)

	choices := make([]tempest.CommandOptionChoice, 0, min(len(names), 25))
	for _, name := range names {
		if !strings.HasPrefix(strings.ToLower(name), typed) {
			continue
		}

		choices = append(choices, tempest.CommandOptionChoice{Name: name, Value: name})
		if len(choices) == 25 {
			break
		}
	}
	return choices
}

func focusedString(itx tempest.CommandInteraction) string {
	for _, option := range itx.Data.Options {
		if !option.Focused {
			continue
		}

		value, _ := option.Value.(string)
		return value
	}
	return ""
}
