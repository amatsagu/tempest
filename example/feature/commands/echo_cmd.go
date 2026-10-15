package commands

import (
	"context"

	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type echoCommand struct{}

func newEchoCommand() *echoCommand { return &echoCommand{} }

func (c *echoCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "echo",
		Description: "Repeat a message back to you.",
		Options: []discord.CommandOption{
			{
				Type:        discord.OptionString,
				Name:        "text",
				Description: "What to repeat.",
				Required:    true,
				MaxLength:   util.Ptr(2000),
			},
			{
				Type:        discord.OptionBoolean,
				Name:        "ephemeral",
				Description: "Only you can see the reply.",
			},
		},
	}
}

// handle repeats the text, with the ephemeral flag on the first response.
func (c *echoCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	text, _ := itx.Data.StringOption("text")
	ephemeral, _ := itx.Data.BoolOption("ephemeral")

	return itx.Reply(discord.ResponseMessageData{Content: text}, ephemeral)
}
