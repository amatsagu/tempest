package components

import (
	"context"
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

type ticketCommand struct {
	client *tempest.Client
}

func newTicketCommand(client *tempest.Client) *ticketCommand {
	return &ticketCommand{client: client}
}

func (c *ticketCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "ticket",
		Description: "Post a button whose handler exists only for that one button.",
	}
}

// handle registers a one-shot component handler keyed by this interaction's ID and
// removes it once it fires: dynamic routing for IDs not known at startup.
func (c *ticketCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	id := "ticket:" + itx.ID.String()

	if err := c.client.Component(id, func(_ context.Context, oneShot *discord.ComponentInteraction, _ *route.Params) error {
		c.client.RemoveComponent(id)
		return oneShot.Update(discord.ResponseMessageData{Content: "Ticket created."})
	}); err != nil {
		return fmt.Errorf("register one-shot ticket handler: %w", err)
	}

	row := builder.ActionRow(builder.Button(id, "Create ticket", discord.ButtonSuccess))

	return itx.Reply(discord.ResponseMessageData{
		Content:    "Press the button to run a one-shot handler.",
		Components: discord.Components{row},
	}, false)
}
