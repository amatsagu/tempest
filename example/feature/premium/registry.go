// Package premium demonstrates monetization: SKUs, entitlements, and subscriptions
// for one-time purchases, the shop, and app subscriptions.
package premium

import (
	"fmt"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/gateway"
)

// Register wires /premium, its autocomplete, and the entitlement events that
// fulfillment listens to.
func Register(client *tempest.Client) ([]discord.Command, error) {
	premium := newPremiumCommand(client)

	routes := []struct {
		name   string
		handle tempest.CommandHandler
	}{
		{"premium status", premium.status},
		{"premium skus", premium.skus},
		{"premium subscriptions", premium.subscriptions},
	}

	for _, route := range routes {
		if err := client.Command(route.name, route.handle); err != nil {
			return nil, fmt.Errorf("register /%s: %w", route.name, err)
		}
	}

	if err := client.CommandAutocomplete("premium subscriptions", premium.autocomplete); err != nil {
		return nil, fmt.Errorf("register /premium subscriptions autocomplete: %w", err)
	}

	// A purchase creates an entitlement; that event is where a real app grants the
	// item, and then consumes the entitlement when the SKU is consumable.
	client.Event(gateway.EventEntitlementCreate, premium.onEntitlementEvent)
	client.Event(gateway.EventEntitlementUpdate, premium.onEntitlementEvent)
	client.Event(gateway.EventEntitlementDelete, premium.onEntitlementEvent)

	return []discord.Command{premium.Definition()}, nil
}
