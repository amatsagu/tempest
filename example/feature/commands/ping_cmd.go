package commands

import (
	"context"

	"github.com/amatsagu/tempest/discord"
)

type pingCommand struct{}

func newPingCommand() *pingCommand { return &pingCommand{} }

func (c *pingCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "ping",
		Description: "Check that the bot is alive.",
	}
}

func (c *pingCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	return itx.Reply(discord.ResponseMessageData{Content: "Pong."}, false)
}
