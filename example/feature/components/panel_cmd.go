package components

import (
	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

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
		Command: discord.Command{
			Name:        "panel",
			Description: "Show a panel with buttons and a select menu.",
		},
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

func (c *panelCommand) handle(itx *tempest.CommandInteraction) error {
	buttons := util.ActionRow(
		util.Button(openID, "Open form", discord.PrimaryButtonStyle),
		util.Button(dismissID, "Dismiss", discord.SecondaryButtonStyle),
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

	return itx.SendReply(discord.ResponseMessageData{
		Embeds:     []discord.Embed{card},
		Components: []discord.MessageComponent{buttons, picker},
	}, false, nil)
}
