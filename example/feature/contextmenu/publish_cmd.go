package contextmenu

import (
	"context"
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

type publishCommand struct{}

func newPublishCommand() *publishCommand { return &publishCommand{} }

func (c *publishCommand) Definition() tempest.Command {
	return tempest.Command{
		Type:                discord.MessageCommandType,
		Name:                "Publish",
		SlashCommandHandler: c.handle,
	}
}

func (c *publishCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	if err := itx.Client.CrosspostMessage(ctx, itx.ChannelID, discord.MessageID{Snowflake: itx.Data.TargetID}); err != nil {
		return fmt.Errorf("crosspost message: %w", err)
	}

	return itx.SendLinearReply(ctx, "Message published to following channels.", true)
}
