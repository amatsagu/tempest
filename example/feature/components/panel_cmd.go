package components

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

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

func (c *panelCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "panel",
		Description:         "Show a panel with buttons and a select menu.",
		SlashCommandHandler: c.handle,
	}
}

func (c *panelCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	buttons := builder.ActionRow(
		builder.Button(openID, "Open form", discord.PrimaryButtonStyle),
		builder.Button(dismissID, "Dismiss", discord.SecondaryButtonStyle),
		builder.LinkButton("https://discord.com/developers/docs/interactions/overview", "Docs"),
	)

	picker := builder.ActionRow(
		builder.StringSelect(
			pickID,
			"Pick an option",
			builder.SelectOption("First", "first"),
			builder.SelectOption("Second", "second"),
		),
	)

	card := util.Embed("Control panel", "Each control is wired to a handler registered at startup.")

	return itx.SendReply(ctx, discord.ResponseMessageData{
		Embeds:     []discord.Embed{card},
		Components: []discord.MessageComponent{buttons, picker},
	}, false, nil)
}
