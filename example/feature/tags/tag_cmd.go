package tags

import (
	"context"
	"fmt"
	"strings"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type tagCommand struct {
	store *tagStore
}

func newTagCommand() *tagCommand {
	return &tagCommand{store: newTagStore()}
}

func (c *tagCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:        "tag",
		Description: "Save and recall short text snippets, per server.",
	}
}

func (c *tagCommand) add(ctx context.Context, itx *tempest.CommandInteraction) error {
	if itx.GuildID.Snowflake == 0 {
		return itx.SendLinearReply(ctx, "Tags only work in servers.", true)
	}

	name, _ := util.StringOption(itx, "name")
	name = strings.TrimSpace(name)
	value, _ := util.StringOption(itx, "value")

	if name == "" {
		return itx.SendLinearReply(ctx, "A tag needs a name.", true)
	}

	c.store.add(itx.GuildID, name, value)
	return itx.SendLinearReply(ctx, "Saved tag "+name+".", true)
}

func (c *tagCommand) show(ctx context.Context, itx *tempest.CommandInteraction) error {
	name, _ := util.StringOption(itx, "name")

	value, ok := c.store.get(itx.GuildID, name)
	if !ok {
		return fmt.Errorf("tag %q not found", name)
	}

	card := util.Embed(name, value)
	return itx.SendReply(ctx, discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false, nil)
}

func (c *tagCommand) remove(ctx context.Context, itx *tempest.CommandInteraction) error {
	name, _ := util.StringOption(itx, "name")

	if !c.store.remove(itx.GuildID, name) {
		return itx.SendLinearReply(ctx, "No tag named "+name+".", true)
	}
	return itx.SendLinearReply(ctx, "Removed tag "+name+".", true)
}

func (c *tagCommand) list(ctx context.Context, itx *tempest.CommandInteraction) error {
	if itx.GuildID.Snowflake == 0 {
		return itx.SendLinearReply(ctx, "Tags only work in servers.", true)
	}

	names := c.store.names(itx.GuildID)
	if len(names) == 0 {
		return itx.SendLinearReply(ctx, "There are no tags in this server yet.", true)
	}

	return itx.SendLinearReply(ctx, "Tags: "+strings.Join(names, ", "), false)
}

func (c *tagCommand) adminClear(ctx context.Context, itx *tempest.CommandInteraction) error {
	if itx.GuildID.Snowflake == 0 {
		return itx.SendLinearReply(ctx, "Tags only work in servers.", true)
	}

	c.store.clear(itx.GuildID)
	return itx.SendLinearReply(ctx, "Cleared all tags in this server.", true)
}

func (c *tagCommand) autocomplete(_ context.Context, itx *tempest.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	typed := strings.ToLower(util.FocusedString(itx))
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
