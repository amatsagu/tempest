package premium

import (
	"fmt"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// Register wires /premium and its autocomplete. Entitlement events are handled
// by the gateway event handler in example/events.
func Register(client *tempest.BaseClient) ([]tempest.Command, error) {
	premium := newPremiumCommand(client)
	root := premium.Definition()

	if err := client.RegisterCommand(root); err != nil {
		return nil, fmt.Errorf("register /premium: %w", err)
	}

	status := tempest.Command{
		Name:                "status",
		Description:         "Show what you and this server are entitled to right now.",
		SlashCommandHandler: util.CommandHandler(premium.status),
	}

	skus := tempest.Command{
		Name:                "skus",
		Description:         "List the app's premium offerings.",
		SlashCommandHandler: util.CommandHandler(premium.skus),
	}

	subscriptions := tempest.Command{
		Name:        "subscriptions",
		Description: "List your subscriptions for one SKU.",
		Options: []tempest.CommandOption{
			{Type: tempest.STRING_OPTION_TYPE, Name: "sku", Description: "The subscription SKU.", Required: true, AutoComplete: true},
		},
		AutoCompleteHandler: premium.autocomplete,
		SlashCommandHandler: util.CommandHandler(premium.subscriptions),
	}

	definitions := []tempest.Command{root, status, skus, subscriptions}
	for _, definition := range definitions[1:] {
		if err := client.RegisterSubCommand(definition, "premium"); err != nil {
			return nil, fmt.Errorf("register /premium %s: %w", definition.Name, err)
		}
	}

	return definitions, nil
}
