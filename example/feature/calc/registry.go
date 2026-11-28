// Package calc shows an interactive calculator with in-memory session state.
package calc

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

func Register(client *tempest.Client) ([]tempest.Command, error) {
	calc := newCalcCommand()

	if err := client.RegisterComponent("calc:{session}:{key}", calc.handleKey); err != nil {
		return nil, fmt.Errorf("register calc component pattern: %w", err)
	}

	definition := calc.Definition()
	if err := client.RegisterCommand(definition); err != nil {
		return nil, fmt.Errorf("register /calc: %w", err)
	}

	return []tempest.Command{definition}, nil
}
