package components

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// handleOpenForm answers a button with a modal; its submission is handled by
// handleSubmitForm.
func (c *panelCommand) handleOpenForm(itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithModal(util.ModalResponse(formID, "Tell the bot something",
		util.Label("Title", util.ShortTextInput(titleInputID, "A short title")),
		util.Label("Details", util.ParagraphTextInput(detailsInputID, "Anything longer")),
	))
}

// handleDismiss edits the message the button was attached to.
func (c *panelCommand) handleDismiss(itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithUpdate(tempest.ResponseMessageData{Content: "Panel dismissed."})
}

// handlePick answers the string select from Data.Values.
func (c *panelCommand) handlePick(itx *tempest.ComponentInteraction) error {
	picked := "nothing"
	if len(itx.Data.Values) > 0 {
		picked = itx.Data.Values[0]
	}
	return itx.AcknowledgeWithUpdate(tempest.ResponseMessageData{Content: "You picked: " + picked})
}
