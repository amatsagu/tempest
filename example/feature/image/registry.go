package image

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires /rotate and the modal it opens.
func Register(client *tempest.Client) ([]tempest.Command, error) {
	rotate := newRotateCommand()

	if err := client.RegisterModal(rotateModalID, rotate.handleSubmit); err != nil {
		return nil, fmt.Errorf("register %s: %w", rotateModalID, err)
	}

	definition := rotate.Definition()
	if err := client.RegisterCommand(definition); err != nil {
		return nil, fmt.Errorf("register /rotate: %w", err)
	}

	return []tempest.Command{definition}, nil
}
