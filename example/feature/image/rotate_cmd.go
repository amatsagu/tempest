// Package image demonstrates a file round trip: /rotate takes a PNG and returns it
// rotated 90 degrees clockwise.
package image

import (
	"context"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

// The modal and its file-upload field, registered once at startup.
const (
	rotateModalID  = "image:rotate"
	rotateUploadID = "image:rotate:file"
)

type rotateCommand struct{}

func newRotateCommand() *rotateCommand {
	return &rotateCommand{}
}

func (c *rotateCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "rotate",
		Description: "Upload a PNG and get it back rotated 90 degrees clockwise.",
	}
}

// handle answers with a modal through Respond; the upload accepts only .png.
func (c *rotateCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	return itx.Respond(rotateModal())
}

// rotateModal builds the modal /rotate opens; extracted so a test can assert its wire format.
func rotateModal() discord.Response {
	return builder.ModalResponse(rotateModalID, "Rotate a PNG",
		builder.LabelWithFileUpload("PNG",
			builder.FileUpload(rotateUploadID, ".png"),
		),
	)
}
