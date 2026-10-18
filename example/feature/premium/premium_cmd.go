package premium

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

// premiumCommand answers /premium. It caches the SKU list because the status
// command, the subscription command, and autocomplete all need it.
type premiumCommand struct {
	client   *tempest.Client
	skuCache []discord.SKU
	mu       sync.Mutex
}

func newPremiumCommand(client *tempest.Client) *premiumCommand {
	return &premiumCommand{client: client}
}

func (c *premiumCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "premium",
		Description: "Look at this app's SKUs, your entitlements, and your subscriptions.",
		Options: []discord.CommandOption{
			{
				Type:        discord.OptionSubCommand,
				Name:        "status",
				Description: "Show what you and this server are entitled to right now.",
			},
			{
				Type:        discord.OptionSubCommand,
				Name:        "skus",
				Description: "List the app's premium offerings.",
			},
			{
				Type:        discord.OptionSubCommand,
				Name:        "subscriptions",
				Description: "List your subscriptions for one SKU.",
				Options: []discord.CommandOption{
					{Type: discord.OptionString, Name: "sku", Description: "The subscription SKU.", Required: true, Autocomplete: true},
				},
			},
		},
	}
}

// fetchSKUs returns the cached SKU list, loading it on first use.
func (c *premiumCommand) fetchSKUs(ctx context.Context) ([]discord.SKU, error) {
	c.mu.Lock()
	cached := c.skuCache
	c.mu.Unlock()

	if cached != nil {
		return cached, nil
	}

	skus, err := c.client.SKUs(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.skuCache = skus
	c.mu.Unlock()

	return skus, nil
}

// status lists the active entitlements of the invoking user and of the server.
func (c *premiumCommand) status(ctx context.Context, itx *discord.CommandInteraction) error {
	user := itx.BaseUser()
	if user == nil {
		return errors.New("the interaction carries no user")
	}

	userEntitlements, err := c.client.Entitlements(ctx, &discord.EntitlementFilter{
		UserID:       user.ID,
		ExcludeEnded: true,
	})
	if err != nil {
		return err
	}

	var guildEntitlements []discord.Entitlement
	if itx.GuildID.Snowflake != 0 {
		guildEntitlements, err = c.client.Entitlements(ctx, &discord.EntitlementFilter{
			GuildID:      itx.GuildID,
			ExcludeEnded: true,
		})
		if err != nil {
			return err
		}
	}

	now := time.Now()

	var b strings.Builder
	b.WriteString("### Active entitlements\n**Yours**\n")
	writeEntitlements(&b, userEntitlements, now)
	b.WriteString("**This server's**\n")
	writeEntitlements(&b, guildEntitlements, now)

	return itx.Reply(discord.ResponseMessageData{Content: b.String()}, true)
}

// skus lists every SKU the app offers, marking the ones that can be bought.
func (c *premiumCommand) skus(ctx context.Context, itx *discord.CommandInteraction) error {
	skus, err := c.fetchSKUs(ctx)
	if err != nil {
		return err
	}

	if len(skus) == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "This app has no SKUs."}, true)
	}

	var b strings.Builder
	b.WriteString("### SKUs\n")

	for _, sku := range skus {
		if sku.Type == discord.SKUSubscriptionGroup {
			continue // System-generated companion of a subscription; not sold.
		}

		fmt.Fprintf(&b, "- **%s** (`%s`) %s", sku.Name, sku.Slug, skuKind(sku))
		if sku.Available() {
			b.WriteString(", available")
		}
		b.WriteByte('\n')
	}

	return itx.Reply(discord.ResponseMessageData{Content: b.String()}, true)
}

// subscriptions lists the invoking user's subscriptions for the chosen SKU.
func (c *premiumCommand) subscriptions(ctx context.Context, itx *discord.CommandInteraction) error {
	user := itx.BaseUser()
	if user == nil {
		return errors.New("the interaction carries no user")
	}

	// The autocomplete sends the SKU ID as the option value.
	skuID, ok := itx.Data.SnowflakeOption("sku")
	if !ok {
		return errors.New("pick a SKU")
	}

	subscriptions, err := c.client.Subscriptions(ctx, discord.SkuID{Snowflake: skuID}, &discord.SubscriptionFilter{
		UserID: user.ID,
	})
	if err != nil {
		return err
	}

	if len(subscriptions) == 0 {
		return itx.Reply(discord.ResponseMessageData{Content: "You have no subscription for that SKU."}, true)
	}

	var b strings.Builder
	b.WriteString("### Subscriptions\n")

	for _, subscription := range subscriptions {
		fmt.Fprintf(&b, "- `%s` %s, renews <t:%d:R>", subscription.ID, subscriptionStatus(subscription.Status), subscription.CurrentPeriodEnd.Unix())
		if subscription.CanceledAt != nil {
			fmt.Fprintf(&b, ", canceled <t:%d:R>", subscription.CanceledAt.Unix())
		}
		b.WriteByte('\n')
	}

	return itx.Reply(discord.ResponseMessageData{Content: b.String()}, true)
}

// autocomplete suggests the subscription SKUs by name.
func (c *premiumCommand) autocomplete(ctx context.Context, itx *discord.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	_, value, ok := itx.Data.Focused()
	if !ok {
		return nil, nil
	}

	typed, _ := value.(string)
	typed = strings.ToLower(typed)

	skus, err := c.fetchSKUs(ctx)
	if err != nil {
		// Autocomplete has to answer with choices, so a failure just suggests
		// nothing instead of surfacing an error to the user.
		return nil, nil
	}

	choices := make([]discord.CommandOptionChoice, 0, len(skus))
	for _, sku := range skus {
		if !sku.IsSubscription() {
			continue
		}

		if typed != "" && !strings.Contains(strings.ToLower(sku.Name), typed) {
			continue
		}

		choices = append(choices, discord.CommandOptionChoice{Name: sku.Name, Value: sku.ID.String()})
		if len(choices) == 25 {
			break
		}
	}

	return choices, nil
}

// onEntitlementEvent is the fulfillment hook: it logs every entitlement change so
// the shape of the payload is visible, and shows where granting happens.
func (c *premiumCommand) onEntitlementEvent(ctx context.Context, shard int, event string, data jsontext.Value) {
	var entitlement discord.Entitlement
	if err := json.Unmarshal(data, &entitlement); err != nil {
		slog.Error("entitlement event payload", "event", event, "err", err)
		return
	}

	log := slog.With(
		"event", event,
		"shard", shard,
		"id", entitlement.ID,
		"sku", entitlement.SkuID,
		"type", entitlementKind(entitlement.Type),
	)

	if entitlement.GuildID.Snowflake != 0 {
		log = log.With("guild", entitlement.GuildID)
	} else {
		log = log.With("user", entitlement.UserID)
	}

	log.Info("entitlement changed")

	if !entitlement.Active(time.Now()) {
		return
	}

	if c.isConsumable(ctx, entitlement.SkuID) {
		// Deliver the item here, then call
		// c.client.ConsumeEntitlement(ctx, entitlement.ID) so the user cannot use
		// the same purchase twice.
		log.Info("consumable entitlement ready to fulfil", "consume_with", "ConsumeEntitlement")
	}
}

// isConsumable reports whether the SKU behind an entitlement is used up once.
func (c *premiumCommand) isConsumable(ctx context.Context, skuID discord.SkuID) bool {
	skus, err := c.fetchSKUs(ctx)
	if err != nil {
		return false
	}

	for _, sku := range skus {
		if sku.ID == skuID {
			return sku.Type == discord.SKUConsumable
		}
	}
	return false
}

// writeEntitlements appends one line per active entitlement; deleted, consumed,
// and expired ones are skipped.
func writeEntitlements(b *strings.Builder, entitlements []discord.Entitlement, now time.Time) {
	active := 0

	for _, entitlement := range entitlements {
		if !entitlement.Active(now) {
			continue
		}

		active++
		fmt.Fprintf(b, "- SKU `%s` (%s)", entitlement.SkuID, entitlementKind(entitlement.Type))
		if entitlement.EndsAt != nil {
			fmt.Fprintf(b, ", until <t:%d:R>", entitlement.EndsAt.Unix())
		}
		b.WriteByte('\n')
	}

	if active == 0 {
		b.WriteString("- nothing active\n")
	}
}

// skuKind names a SKU type for display.
func skuKind(sku discord.SKU) string {
	switch sku.Type {
	case discord.SKUDurable:
		return "durable one-time purchase"
	case discord.SKUConsumable:
		return "consumable one-time purchase"
	case discord.SKUSubscription:
		if sku.GuildSubscription() {
			return "guild subscription"
		}
		return "user subscription"
	case discord.SKUSubscriptionGroup:
		return "subscription group"
	default:
		return "unknown"
	}
}

// entitlementKind names an entitlement type for display.
func entitlementKind(kind discord.EntitlementType) string {
	switch kind {
	case discord.EntitlementPurchase:
		return "purchase"
	case discord.EntitlementApplicationSubscription:
		return "app subscription"
	case discord.EntitlementDeveloperGift:
		return "developer gift"
	case discord.EntitlementUserGift:
		return "user gift"
	case discord.EntitlementTestModePurchase:
		return "test purchase"
	case discord.EntitlementFreePurchase:
		return "free purchase"
	case discord.EntitlementPremiumSubscription:
		return "Nitro"
	case discord.EntitlementPremiumPurchase:
		return "Nitro claim"
	default:
		return "unknown"
	}
}

// subscriptionStatus names a subscription status for display.
func subscriptionStatus(status discord.SubscriptionStatus) string {
	switch status {
	case discord.SubscriptionActive:
		return "active"
	case discord.SubscriptionInactive:
		return "inactive"
	case discord.SubscriptionEnding:
		return "ending"
	default:
		return "unknown"
	}
}
