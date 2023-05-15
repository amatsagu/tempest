package tempest

import "errors"

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
// This method doesn't rely on any in-memory state so it's safe to use it for bot single instance applications as well as network of instances.
func (client Client) RegisterComponent(customIDs []string, fn func(ComponentInteraction)) error {
	for _, ID := range customIDs {
		_, exists := client.components[ID]
		if exists {
			return errors.New("client already has registered \"" + ID + "\" component (custom id already in use)")
		}
	}

	for _, ID := range customIDs {
		client.components[ID] = fn
	}

	return nil
}

// Bind function to modal with matching custom id. App will automatically run bound function whenever receiving modal interaction with matching custom id.
// This method doesn't rely on any in-memory state so it's safe to use it for bot single instance applications as well as network of instances.
func (client Client) RegisterModal(customID string, fn func(ModalInteraction)) error {
	_, exists := client.modals[customID]
	if exists {
		return errors.New("client already has registered \"" + customID + "\" modal (custom id already in use)")
	}

	client.modals[customID] = fn
	return nil
}
