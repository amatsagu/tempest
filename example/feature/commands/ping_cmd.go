package commands

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

type pingCommand struct{}

func newPingCommand() *pingCommand { return &pingCommand{} }

func (c *pingCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "ping",
		Description:         "Check that the bot is alive.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

func (c *pingCommand) handle(itx *tempest.CommandInteraction) error {
	return itx.SendLinearReply("Pong.", false)
}
