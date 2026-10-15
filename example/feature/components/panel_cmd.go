package components

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

// Static custom IDs, registered once before the bot starts.
const (
	openID    = "panel:open"
	dismissID = "panel:dismiss"
	pickID    = "panel:pick"
	formID    = "panel:form"

	titleInputID   = "panel:form:title"
	detailsInputID = "panel:form:details"
)

type panelCommand struct{}

func newPanelCommand() *panelCommand { return &panelCommand{} }

func (c *panelCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "panel",
		Description: "Show a panel with buttons and a select menu.",
	}
}

func (c *panelCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	buttons := builder.ActionRow(
		builder.Button(openID, "Open form", discord.ButtonPrimary),
		builder.Button(dismissID, "Dismiss", discord.ButtonSecondary),
		builder.LinkButton("https://discord.com/developers/docs/interactions/overview", "Docs"),
	)

	picker := builder.ActionRowWithStringSelect(
		builder.StringSelect(
			pickID,
			"Pick an option",
			builder.SelectOption("First", "first"),
			builder.SelectOption("Second", "second"),
		),
	)

	card := builder.Embed("Control panel", "Each control is wired to a handler registered at startup.")

	return itx.Reply(discord.ResponseMessageData{
		Embeds:     []discord.Embed{card},
		Components: discord.Components{buttons, picker},
	}, false)
}
