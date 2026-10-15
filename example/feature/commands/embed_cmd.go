package commands

import (
	"context"
	"strconv"
	"time"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

type embedCommand struct{}

func newEmbedCommand() *embedCommand { return &embedCommand{} }

func (c *embedCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "embed",
		Description: "Show a rich embed built with the builder package.",
	}
}

// handle builds the whole reply with builders instead of filling structs by
// hand, which is the style the builder package is meant to encourage.
func (c *embedCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	footer := builder.EmbedFooter("Tempest example")
	now := time.Now()

	card := builder.Embed("Tempest", "A low-allocation Discord runtime.")
	card.Color = 0x5865F2
	card.Footer = &footer
	card.Timestamp = &now
	card.Fields = []discord.EmbedField{
		builder.EmbedField("Guild", itx.GuildID.String(), true),
		builder.EmbedField("Channel", itx.ChannelID.String(), true),
		builder.EmbedField("Shard", strconv.Itoa(int(itx.ShardID)), true),
	}

	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false)
}
