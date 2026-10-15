package components

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// handleSubmitForm reads the modal's text inputs by their custom IDs and replies
// with an ephemeral embed. Submitted values are looked up with Data.TextInput.
func (c *panelCommand) handleSubmitForm(_ context.Context, itx *discord.ModalInteraction, _ *route.Params) error {
	title, _ := itx.Data.TextInput(titleInputID)
	details, _ := itx.Data.TextInput(detailsInputID)

	if title == "" {
		title = "Untitled"
	}

	card := builder.Embed(title, details)

	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, true)
}
