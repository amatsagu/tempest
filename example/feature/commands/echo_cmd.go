package commands

import (
	"errors"
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type echoCommand struct{}

func newEchoCommand() *echoCommand { return &echoCommand{} }

func (c *echoCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Name:        "echo",
			Description: "Repeat a message back to you.",
			Options: []discord.CommandOption{
				{
					Type:        discord.StringOptionType,
					Name:        "text",
					Description: "What to repeat.",
					Required:    true,
					MaxLength:   2000,
				},
				{
					Type:        discord.BooleanOptionType,
					Name:        "ephemeral",
					Description: "Only you can see the reply.",
				},
				{
					Type:        discord.BooleanOptionType,
					Name:        "dm",
					Description: "Send the echo directly as a private direct message.",
				},
				{
					Type:        discord.ChannelOptionType,
					Name:        "channel",
					Description: "Send the text as a plain linear message to another channel.",
				},
			},
		},
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle repeats the text. When dm or channel is set, it demonstrates the client's
// SendPrivateMessage and SendLinearMessage helpers.
func (c *echoCommand) handle(itx *tempest.CommandInteraction) error {
	text, _ := util.StringOption(itx, "text")
	ephemeral, _ := util.BoolOption(itx, "ephemeral")

	if targetChannel, ok := util.SnowflakeOption(itx, "channel"); ok {
		channelID := discord.ChannelID{Snowflake: targetChannel}
		if _, err := itx.BaseClient.SendLinearMessage(channelID, text); err != nil {
			return fmt.Errorf("send linear message: %w", err)
		}

		return itx.SendLinearReply(fmt.Sprintf("Echo sent to %s.", util.ChannelMention(channelID)), true)
	}

	if dm, _ := util.BoolOption(itx, "dm"); dm {
		user := itx.BaseUser()
		if user == nil {
			return errors.New("cannot determine user to send DM")
		}

		if _, err := itx.BaseClient.SendPrivateMessage(user.ID, discord.Message{Content: text}, nil); err != nil {
			return fmt.Errorf("send private message: %w", err)
		}

		return itx.SendLinearReply("Echo sent to your direct messages.", true)
	}

	return itx.SendReply(discord.ResponseMessageData{Content: text}, ephemeral, nil)
}
