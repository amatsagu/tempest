// Package image demonstrates a file round trip: /rotate takes a PNG and returns it
// rotated 90 degrees clockwise.
package image

import (
	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
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

func (c *rotateCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "rotate",
		Description:         "Upload a PNG and get it back rotated 90 degrees clockwise.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

// handle answers with a modal; the upload accepts only .png.
func (c *rotateCommand) handle(itx *tempest.CommandInteraction) error {
	return itx.SendModal(rotateModal())
}

// rotateModal builds the modal /rotate opens; extracted so a test can assert its
// wire format.
func rotateModal() tempest.ResponseModalData {
	return util.ModalResponse(rotateModalID, "Rotate a PNG",
		util.Label("PNG", util.FileUpload(rotateUploadID, ".png")),
	)
}
