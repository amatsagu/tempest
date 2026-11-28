package components

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

// handleOpenForm answers a button with a modal; its submission is handled by
// handleSubmitForm.
func (c *panelCommand) handleOpenForm(ctx context.Context, itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithModal(ctx, util.ModalResponse(formID, "Tell the bot something",
		util.Label("Title", util.ShortTextInput(titleInputID, "A short title")),
		util.Label("Details", util.ParagraphTextInput(detailsInputID, "Anything longer")),
	))
}

// handleDismiss edits the message the button was attached to.
func (c *panelCommand) handleDismiss(ctx context.Context, itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{Content: "Panel dismissed."})
}

// handlePick answers the string select from Data.Values.
func (c *panelCommand) handlePick(ctx context.Context, itx *tempest.ComponentInteraction) error {
	picked := "nothing"
	if len(itx.Data.Values) > 0 {
		picked = itx.Data.Values[0]
	}
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{Content: "You picked: " + picked})
}
