package commands

import (
	"strconv"
	"time"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

type embedCommand struct{}

func newEmbedCommand() *embedCommand { return &embedCommand{} }

func (c *embedCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "embed",
		Description:         "Show a rich embed built with plain Tempest types.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

func (c *embedCommand) handle(itx *tempest.CommandInteraction) error {
	footer := util.EmbedFooter("Tempest example")
	now := time.Now()

	card := util.Embed("Tempest", "A low-allocation Discord runtime.")
	card.Color = 0x5865F2
	card.Footer = &footer
	card.Timestamp = &now
	card.Fields = []tempest.EmbedField{
		util.EmbedField("Guild", itx.GuildID.String(), true),
		util.EmbedField("Channel", itx.ChannelID.String(), true),
		util.EmbedField("Shard", strconv.Itoa(int(itx.ShardID)), true),
	}

	return itx.SendReply(tempest.ResponseMessageData{Embeds: []tempest.Embed{card}}, false, nil)
}
