package premium

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// Register wires /premium and its autocomplete.
func Register(client *tempest.Client) ([]tempest.Command, error) {
	premium := newPremiumCommand(client)
	root := premium.Definition()

	if err := client.RegisterCommand(root); err != nil {
		return nil, fmt.Errorf("register /premium: %w", err)
	}

	status := tempest.Command{
		Command: discord.Command{
			Name:        "status",
			Description: "Show what you and this server are entitled to right now.",
		},
		SlashCommandHandler: premium.status,
	}

	skus := tempest.Command{
		Command: discord.Command{
			Name:        "skus",
			Description: "List the app's premium offerings.",
		},
		SlashCommandHandler: premium.skus,
	}

	subscriptions := tempest.Command{
		Command: discord.Command{
			Name:        "subscriptions",
			Description: "List your subscriptions for one SKU.",
			Options: []discord.CommandOption{
				{Type: discord.StringOptionType, Name: "sku", Description: "The subscription SKU.", Required: true, AutoComplete: true},
			},
		},
		AutoCompleteHandler: premium.autocomplete,
		SlashCommandHandler: premium.subscriptions,
	}

	definitions := []tempest.Command{root, status, skus, subscriptions}
	for _, definition := range definitions[1:] {
		if err := client.RegisterSubCommand(definition, "premium"); err != nil {
			return nil, fmt.Errorf("register /premium %s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
