package contextmenu

import (
	"fmt"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

type publishCommand struct{}

func newPublishCommand() *publishCommand { return &publishCommand{} }

func (c *publishCommand) Definition() tempest.Command {
	return tempest.Command{
		Type:                tempest.MESSAGE_COMMAND_TYPE,
		Name:                "Publish",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle publishes (crossposts) an announcement message to following channels.
func (c *publishCommand) handle(itx *tempest.CommandInteraction) error {
	if err := itx.BaseClient.CrosspostMessage(itx.ChannelID, itx.Data.TargetID); err != nil {
		return fmt.Errorf("crosspost message: %w", err)
	}

	return itx.SendLinearReply("Message published to following channels.", true)
}
