package commands

import (
	"context"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type embedCommand struct{}

func newEmbedCommand() *embedCommand { return &embedCommand{} }

func (c *embedCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Name:        "embed",
			Description: "Show a rich embed built with plain Tempest types.",
		},
		SlashCommandHandler: c.handle,
	}
}

func (c *embedCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	footer := util.EmbedFooter("Tempest example")
	now := time.Now()

	card := util.Embed("Tempest", "A low-allocation Discord runtime.")
	card.Color = 0x5865F2
	card.Footer = &footer
	card.Timestamp = &now
	card.Fields = []discord.EmbedField{
		util.EmbedField("Guild", itx.GuildID.String(), true),
		util.EmbedField("Channel", itx.ChannelID.String(), true),
	}

	return itx.SendReply(ctx, discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false, nil)
}
