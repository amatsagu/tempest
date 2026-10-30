// Package components shows buttons, selects, modals, and dynamic component handlers.
package components

import (
	"fmt"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// Register wires the static component and modal handlers plus the commands that
// show them.
func Register(client *tempest.BaseClient) ([]tempest.Command, error) {
	panel := newPanelCommand()
	ticket := newTicketCommand(client)

	components := []struct {
		id      string
		handler func(tempest.ComponentInteraction)
	}{
		{openID, util.ComponentHandler(panel.handleOpenForm)},
		{dismissID, util.ComponentHandler(panel.handleDismiss)},
		{pickID, util.ComponentHandler(panel.handlePick)},
	}

	for _, component := range components {
		if err := client.RegisterComponent([]string{component.id}, component.handler); err != nil {
			return nil, fmt.Errorf("register %s: %w", component.id, err)
		}
	}

	if err := client.RegisterModal(formID, util.ModalHandler(panel.handleSubmitForm)); err != nil {
		return nil, fmt.Errorf("register %s: %w", formID, err)
	}

	definitions := []tempest.Command{panel.Definition(), ticket.Definition()}
	for _, definition := range definitions {
		if err := client.RegisterCommand(definition); err != nil {
			return nil, fmt.Errorf("register /%s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
