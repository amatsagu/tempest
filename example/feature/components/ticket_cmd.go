package components

import (
	"context"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type ticketCommand struct {
	client *tempest.Client
}

func newTicketCommand(client *tempest.Client) *ticketCommand {
	return &ticketCommand{client: client}
}

func (c *ticketCommand) Definition() tempest.Command {
	return tempest.Command{
		Command: discord.Command{
			Name:        "ticket",
			Description: "Post a button whose handler exists only for that one button.",
		},
		SlashCommandHandler: c.handle,
	}
}

// handle registers a runtime component handler keyed by this interaction's ID and
// unregisters it as soon as the owner clicks it or the 2-minute window expires.
func (c *ticketCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	id := "ticket:" + itx.ID.String()
	row := util.ActionRow(util.Button(id, "Create ticket", discord.SuccessButtonStyle))

	user := itx.BaseUser()

	var timer *time.Timer
	if err := c.client.RegisterComponent(id, func(clickCtx context.Context, click *tempest.ComponentInteraction) error {
		if user != nil {
			clickUser := click.BaseUser()
			if clickUser == nil || clickUser.ID != user.ID {
				return click.Acknowledge(clickCtx)
			}
		}

		if timer != nil {
			timer.Stop()
		}
		c.client.UnregisterComponent(id)

		return click.AcknowledgeWithUpdate(clickCtx, discord.ResponseMessageData{Content: "Ticket created."})
	}); err != nil {
		return err
	}

	if err := itx.SendReply(ctx, discord.ResponseMessageData{
		Content:    "Press the button to run a one-shot handler.",
		Components: []discord.MessageComponent{row},
	}, false, nil); err != nil {
		c.client.UnregisterComponent(id)
		return err
	}

	timer = time.AfterFunc(2*time.Minute, func() {
		if c.client.UnregisterComponent(id) {
			_ = itx.EditReply(ctx, discord.ResponseMessageData{Content: "The ticket button expired. Run /ticket again."}, false, nil)
		}
	})

	return nil
}
