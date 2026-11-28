package commands

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

type pingCommand struct{}

func newPingCommand() *pingCommand { return &pingCommand{} }

func (c *pingCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Name:        "ping",
			Description: "Check that the bot is alive.",
		},
		SlashCommandHandler: c.handle,
	}
}

func (c *pingCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	return itx.SendLinearReply(ctx, "Pong.", false)
}
