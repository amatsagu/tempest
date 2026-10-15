// Package image demonstrates a file round trip: /rotate opens a modal that only
// accepts a PNG, and the submission comes back rotated 90 degrees clockwise.
// Everything — download, rotation, encode — uses the standard library; the only
// non-standard call is the multipart upload on the shared REST client.
package image

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

// The modal and its file-upload field, registered once at startup.
const (
	rotateModalID  = "image:rotate"
	rotateUploadID = "image:rotate:file"
)

type rotateCommand struct {
	client *tempest.Client
}

func newRotateCommand(client *tempest.Client) *rotateCommand {
	return &rotateCommand{client: client}
}

func (c *rotateCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "rotate",
		Description: "Upload a PNG and get it back rotated 90 degrees clockwise.",
	}
}

// handle answers the command with a modal rather than a message, so it goes
// through Respond. The label's file upload is restricted to the .png extension;
// Discord rejects anything else before the submission reaches the bot.
func (c *rotateCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	return itx.Respond(rotateModal())
}

// rotateModal builds the modal /rotate opens. Extracted so a test can assert the
// wire format of the file upload, in particular its file_types filter.
func rotateModal() discord.Response {
	return builder.ModalResponse(rotateModalID, "Rotate a PNG",
		builder.LabelWithFileUpload("PNG",
			builder.FileUpload(rotateUploadID, ".png"),
		),
	)
}
