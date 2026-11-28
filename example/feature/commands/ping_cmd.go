package commands

import (
	"context"

	"github.com/amatsagu/tempest"
)

type pingCommand struct{}

func newPingCommand() *pingCommand { return &pingCommand{} }

func (c *pingCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "ping",
		Description:         "Check that the bot is alive.",
		SlashCommandHandler: c.handle,
	}
}

func (c *pingCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	return itx.SendLinearReply(ctx, "Pong.", false)
}
