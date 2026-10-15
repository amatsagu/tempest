package image

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires /rotate and the modal it opens.
func Register(client *tempest.Client) ([]discord.Command, error) {
	rotate := newRotateCommand(client)

	if err := client.Modal(rotateModalID, rotate.handleSubmit); err != nil {
		return nil, fmt.Errorf("register %s: %w", rotateModalID, err)
	}
	if err := client.Command("rotate", rotate.handle); err != nil {
		return nil, fmt.Errorf("register /rotate: %w", err)
	}

	return []discord.Command{rotate.Definition()}, nil
}
