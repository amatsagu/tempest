// Package image demonstrates a file round trip: /rotate takes a PNG and returns it
// rotated 90 degrees clockwise.
package image

import (
	"context"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

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
		SlashCommandHandler: c.handle,
	}
}

func (c *rotateCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	return itx.SendModal(ctx, rotateModal())
}

func rotateModal() discord.ResponseModalData {
	return builder.ModalData(rotateModalID, "Rotate a PNG",
		builder.Label("PNG", builder.FileUpload(rotateUploadID, ".png")),
	)
}
