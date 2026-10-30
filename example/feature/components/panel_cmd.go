package components

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
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

func (c *panelCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "panel",
		Description:         "Show a panel with buttons and a select menu.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

func (c *panelCommand) handle(itx *tempest.CommandInteraction) error {
	buttons := util.ActionRow(
		util.Button(openID, "Open form", tempest.PRIMARY_BUTTON_STYLE),
		util.Button(dismissID, "Dismiss", tempest.SECONDARY_BUTTON_STYLE),
		util.LinkButton("https://discord.com/developers/docs/interactions/overview", "Docs"),
	)

	picker := util.ActionRow(
		util.StringSelect(
			pickID,
			"Pick an option",
			util.SelectOption("First", "first"),
			util.SelectOption("Second", "second"),
		),
	)

	card := util.Embed("Control panel", "Each control is wired to a handler registered at startup.")

	return itx.SendReply(tempest.ResponseMessageData{
		Embeds:     []tempest.Embed{card},
		Components: []tempest.MessageComponent{buttons, picker},
	}, false, nil)
}
