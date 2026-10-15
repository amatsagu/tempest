// Package calc shows an interactive calculator with in-memory session state.
package calc

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the calculator command and the single button pattern every
// session's keys route through.
func Register(client *tempest.Client) ([]discord.Command, error) {
	calc := newCalcCommand()

	if err := client.Command("calc", calc.handle); err != nil {
		return nil, fmt.Errorf("register /calc: %w", err)
	}

	if err := client.Component("calc/{session}/{key}", calc.handleButton); err != nil {
		return nil, fmt.Errorf("register calc buttons: %w", err)
	}

	return []discord.Command{calc.Definition()}, nil
}
