// Package calc shows an interactive calculator with in-memory session state.
package calc

import (
	"fmt"

	"github.com/amatsagu/tempest"
)

// Register wires the calculator command. Each invocation registers its own
// one-shot-family component listener through AwaitComponent.
func Register(client *tempest.BaseClient) ([]tempest.Command, error) {
	calc := newCalcCommand()

	definition := calc.Definition()
	if err := client.RegisterCommand(definition); err != nil {
		return nil, fmt.Errorf("register /calc: %w", err)
	}

	return []tempest.Command{definition}, nil
}
