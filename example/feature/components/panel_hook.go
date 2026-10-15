package components

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// handleOpenForm answers a button with a modal. A modal is not a message, so it
// goes out through Respond rather than Reply. The modal's custom ID is formID,
// so its submission is handled by handleSubmitForm.
func (c *panelCommand) handleOpenForm(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	return itx.Respond(builder.ModalResponse(formID, "Tell the bot something",
		builder.Label("Title", builder.ShortTextInput(titleInputID, "A short title")),
		builder.Label("Details", builder.ParagraphTextInput(detailsInputID, "Anything longer")),
	))
}

// handleDismiss edits the message the button was attached to. Update's first
// response is the update-message type, which is what replaces that message.
func (c *panelCommand) handleDismiss(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	return itx.Update(discord.ResponseMessageData{Content: "Panel dismissed."})
}

// handlePick reacts to the string select. The chosen values arrive in
// Data.Values.
func (c *panelCommand) handlePick(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
	picked := "nothing"
	if len(itx.Data.Values) > 0 {
		picked = itx.Data.Values[0]
	}
	return itx.Update(discord.ResponseMessageData{Content: "You picked: " + picked})
}
