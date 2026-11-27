package contextmenu

import (
	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type quoteCommand struct{}

func newQuoteCommand() *quoteCommand { return &quoteCommand{} }

func (c *quoteCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Type: discord.MessageCommandType,
			Name: "Quote",
		},
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle reads the resolved target message.
func (c *quoteCommand) handle(itx *tempest.CommandInteraction) error {
	message := itx.Data.Resolved.ResolveMessage(discord.MessageID{Snowflake: itx.Data.TargetID})

	content := message.Content
	if content == "" {
		content = "(no text content)"
	}

	author := ""
	if message.Author != nil {
		author = util.EffectiveName(*message.Author)
	}

	card := util.Embed("Quoted message", content)
	card.Fields = []discord.EmbedField{
		util.EmbedField("Author", author, true),
	}

	return itx.SendReply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, true, nil)
}
