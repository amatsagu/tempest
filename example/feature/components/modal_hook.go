package components

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

func (c *panelCommand) handleSubmitForm(ctx context.Context, itx *tempest.ModalInteraction) error {
	title := itx.GetInputValue(titleInputID)
	details := itx.GetInputValue(detailsInputID)

	if title == "" {
		title = "Untitled"
	}

	card := util.Embed(title, details)

	return itx.AcknowledgeWithMessage(ctx, discord.ResponseMessageData{Embeds: []discord.Embed{card}}, true)
}
