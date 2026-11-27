package contextmenu

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type publishCommand struct{}

func newPublishCommand() *publishCommand { return &publishCommand{} }

func (c *publishCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Type: discord.MessageCommandType,
			Name: "Publish",
		},
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle publishes (crossposts) an announcement message to following channels.
func (c *publishCommand) handle(itx *tempest.CommandInteraction) error {
	if err := itx.BaseClient.CrosspostMessage(itx.ChannelID, discord.MessageID{Snowflake: itx.Data.TargetID}); err != nil {
		return fmt.Errorf("crosspost message: %w", err)
	}

	return itx.SendLinearReply("Message published to following channels.", true)
}
