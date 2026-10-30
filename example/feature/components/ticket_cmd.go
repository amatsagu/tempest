package components

import (
	"time"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

type ticketCommand struct {
	client *tempest.BaseClient
}

func newTicketCommand(client *tempest.BaseClient) *ticketCommand {
	return &ticketCommand{client: client}
}

func (c *ticketCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "ticket",
		Description:         "Post a button whose handler exists only for that one button.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle registers a one-shot component listener keyed by this interaction's ID.
// AwaitComponent removes it after the callback returns true.
func (c *ticketCommand) handle(itx *tempest.CommandInteraction) error {
	id := "ticket:" + itx.ID.String()
	row := util.ActionRow(util.Button(id, "Create ticket", tempest.SUCCESS_BUTTON_STYLE))

	if err := itx.SendReply(tempest.ResponseMessageData{
		Content:    "Press the button to run a one-shot handler.",
		Components: []tempest.MessageComponent{row},
	}, false, nil); err != nil {
		return err
	}

	user := itx.BaseUser()

	return c.client.AwaitComponent([]string{id}, 2*time.Minute, func(click *tempest.ComponentInteraction) bool {
		if user != nil {
			clickUser := click.BaseUser()
			if clickUser == nil || clickUser.ID != user.ID {
				return false
			}
		}

		if err := itx.EditReply(tempest.ResponseMessageData{Content: "Ticket created."}, false); err != nil {
			return true
		}
		return true
	}, func() {
		_ = itx.EditReply(tempest.ResponseMessageData{Content: "The ticket button expired. Run /ticket again."}, false)
	})
}
