package contextmenu

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

type quoteCommand struct{}

func newQuoteCommand() *quoteCommand { return &quoteCommand{} }

func (c *quoteCommand) Definition() discord.Command {
	return discord.Command{
		Type: discord.CommandMessage,
		Name: "Quote",
	}
}

// handle reads the resolved target message.
func (c *quoteCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	target := discord.MessageID{Snowflake: itx.Data.TargetID}

	var message discord.Message
	if itx.Data.Resolved != nil {
		message = itx.Data.Resolved.Messages[target]
	}

	content := message.Content
	if content == "" {
		content = "(no text content)"
	}

	card := builder.Embed("Quoted message", content)
	card.Fields = []discord.EmbedField{
		builder.EmbedField("Author", message.Author.EffectiveName(), true),
	}

	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, true)
}
