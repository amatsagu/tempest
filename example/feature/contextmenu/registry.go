// Package contextmenu shows the USER and MESSAGE context menu command types.
package contextmenu

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the context menu handlers; the target arrives as Data.TargetID and
// Data.Resolved.
func Register(client *tempest.Client) ([]discord.Command, error) {
	userInfo := newUserInfoCommand()
	quote := newQuoteCommand()

	if err := client.Command("User info", userInfo.handle); err != nil {
		return nil, fmt.Errorf("register User info: %w", err)
	}

	if err := client.Command("Quote", quote.handle); err != nil {
		return nil, fmt.Errorf("register Quote: %w", err)
	}

	return []discord.Command{
		userInfo.Definition(),
		quote.Definition(),
	}, nil
}
