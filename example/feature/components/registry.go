// Package components shows buttons, selects, modals, and runtime component registration.
package components

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

func Register(client *tempest.Client) ([]tempest.Command, error) {
	panel := newPanelCommand()
	ticket := newTicketCommand(client)

	components := []struct {
		handler tempest.ComponentHandler
		id      string
	}{
		{handler: panel.handleOpenForm, id: openID},
		{handler: panel.handleDismiss, id: dismissID},
		{handler: panel.handlePick, id: pickID},
	}

	for _, component := range components {
		if err := client.RegisterComponent(component.id, component.handler); err != nil {
			return nil, fmt.Errorf("register %s: %w", component.id, err)
		}
	}

	if err := client.RegisterModal(formID, panel.handleSubmitForm); err != nil {
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
