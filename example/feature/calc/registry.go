// Package calc shows an interactive calculator: /calc posts a keypad and each
// press edits the same message in place, session state held in memory.
package calc

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the calculator command and the one button handler every
// session's keys route through. The custom ID pattern captures the session and
// key, so a handler does not have to be registered per message.
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
