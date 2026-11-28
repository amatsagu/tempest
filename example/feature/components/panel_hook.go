package components

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

func (c *panelCommand) handleOpenForm(ctx context.Context, itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithModal(ctx, builder.ModalData(formID, "Tell the bot something",
		builder.Label("Title", builder.ShortTextInput(titleInputID, "A short title")),
		builder.Label("Details", builder.ParagraphTextInput(detailsInputID, "Anything longer")),
	))
}

func (c *panelCommand) handleDismiss(ctx context.Context, itx *tempest.ComponentInteraction) error {
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{Content: "Panel dismissed."})
}

func (c *panelCommand) handlePick(ctx context.Context, itx *tempest.ComponentInteraction) error {
	picked := "nothing"
	if len(itx.Data.Values) > 0 {
		picked = itx.Data.Values[0]
	}
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{Content: "You picked: " + picked})
}
