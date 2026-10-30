package components

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// handleSubmitForm reads the modal's text inputs and replies with an ephemeral
// embed.
func (c *panelCommand) handleSubmitForm(itx *tempest.ModalInteraction) error {
	title := itx.GetInputValue(titleInputID)
	details := itx.GetInputValue(detailsInputID)

	if title == "" {
		title = "Untitled"
	}

	card := util.Embed(title, details)

	return itx.AcknowledgeWithMessage(tempest.ResponseMessageData{Embeds: []tempest.Embed{card}}, true)
}
