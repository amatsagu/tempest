package components

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// handleOpenForm answers a button with a modal (through Respond, since a modal is
// not a message); its submission is handled by handleSubmitForm.
func (c *panelCommand) handleOpenForm(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	return itx.Respond(builder.ModalResponse(formID, "Tell the bot something",
		builder.Label("Title", builder.ShortTextInput(titleInputID, "A short title")),
		builder.Label("Details", builder.ParagraphTextInput(detailsInputID, "Anything longer")),
	))
}

// handleDismiss edits the message the button was attached to.
func (c *panelCommand) handleDismiss(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	return itx.Update(discord.ResponseMessageData{Content: "Panel dismissed."})
}

// handlePick answers the string select from Data.Values.
func (c *panelCommand) handlePick(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	picked := "nothing"
	if len(itx.Data.Values) > 0 {
		picked = itx.Data.Values[0]
	}
	return itx.Update(discord.ResponseMessageData{Content: "You picked: " + picked})
}
