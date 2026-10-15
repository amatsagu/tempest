// Package commands shows slash commands: options, autocomplete, embeds,
// ephemeral replies, and deferring work to a follow-up edit.
package commands

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires the slash command handlers and returns their definitions for
// the startup command sync. Each command is a struct so its dependencies are
// captured once and reached from its methods.
//
// Registration failures are reported rather than swallowed, so main.go can stop
// with a clear error instead of running without a handler.
func Register(client *tempest.Client) ([]discord.Command, error) {
	ping := newPingCommand()
	echo := newEchoCommand()
	embed := newEmbedCommand()
	slow := newSlowCommand()
	color := newColorCommand()

	if err := client.Command("ping", ping.handle); err != nil {
		return nil, fmt.Errorf("register /ping: %w", err)
	}
	if err := client.Command("echo", echo.handle); err != nil {
		return nil, fmt.Errorf("register /echo: %w", err)
	}
	if err := client.Command("embed", embed.handle); err != nil {
		return nil, fmt.Errorf("register /embed: %w", err)
	}
	if err := client.Command("slow", slow.handle); err != nil {
		return nil, fmt.Errorf("register /slow: %w", err)
	}
	if err := client.Command("color", color.handle); err != nil {
		return nil, fmt.Errorf("register /color: %w", err)
	}
	if err := client.CommandAutocomplete("color", color.autocomplete); err != nil {
		return nil, fmt.Errorf("register /color autocomplete: %w", err)
	}

	return []discord.Command{
		ping.Definition(),
		echo.Definition(),
		embed.Definition(),
		slow.Definition(),
		color.Definition(),
	}, nil
}
