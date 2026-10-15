// Package components shows buttons, selects, modals, and dynamic component handlers.
package components

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the static component and modal handlers plus the commands that show them.
func Register(client *tempest.Client) ([]discord.Command, error) {
	panel := newPanelCommand()
	ticket := newTicketCommand(client)

	if err := client.Component(openID, panel.handleOpenForm); err != nil {
		return nil, fmt.Errorf("register %s: %w", openID, err)
	}

	if err := client.Component(dismissID, panel.handleDismiss); err != nil {
		return nil, fmt.Errorf("register %s: %w", dismissID, err)
	}

	if err := client.Component(pickID, panel.handlePick); err != nil {
		return nil, fmt.Errorf("register %s: %w", pickID, err)
	}

	if err := client.Modal(formID, panel.handleSubmitForm); err != nil {
		return nil, fmt.Errorf("register %s: %w", formID, err)
	}

	if err := client.Command("panel", panel.handle); err != nil {
		return nil, fmt.Errorf("register /panel: %w", err)
	}

	if err := client.Command("ticket", ticket.handle); err != nil {
		return nil, fmt.Errorf("register /ticket: %w", err)
	}

	return []discord.Command{panel.Definition(), ticket.Definition()}, nil
}
