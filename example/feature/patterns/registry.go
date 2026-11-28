// Package patterns showcases Tempest v2 pattern matching (exact O(1), single/multi
// wildcards, specificity precedence, {rest...} catch-all, patterned modals), the builder
// package, and native Discord polls.
package patterns

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires /action and /poll along with their exact and pattern-matched
// component and modal routes.
func Register(client *tempest.Client) ([]tempest.Command, error) {
	action := newActionCommand()
	poll := newPollCommand()

	// Intentionally register broader {verb} before specific mute/{duration} to
	// demonstrate deterministic specificity precedence.
	componentRoutes := []struct {
		handler tempest.ComponentHandler
		pattern string
	}{
		{handler: action.handleHelp, pattern: actionHelpID},
		{handler: action.handleVerb, pattern: "act/{user_id}/{verb}"},
		{handler: action.handleMute, pattern: "act/{user_id}/mute/{duration}"},
		{handler: action.handleNoteTrail, pattern: "act:note:{trail...}"},
		{handler: poll.handleClose, pattern: "poll/{owner_id}/close"},
	}

	for _, route := range componentRoutes {
		if err := client.RegisterComponent(route.pattern, route.handler); err != nil {
			return nil, fmt.Errorf("register component pattern %q: %w", route.pattern, err)
		}
	}

	if err := client.RegisterModal("act:modal:{user_id}:{kind}", action.handleModalSubmit); err != nil {
		return nil, fmt.Errorf("register modal pattern: %w", err)
	}

	definitions := []tempest.Command{action.Definition(), poll.Definition()}
	for _, definition := range definitions {
		if err := client.RegisterCommand(definition); err != nil {
			return nil, fmt.Errorf("register /%s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
