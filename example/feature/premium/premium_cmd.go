// Package premium demonstrates the monetization data available through Tempest's
// REST surface: SKUs, entitlements, and subscriptions.
package premium

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

const (
	skuDurable           uint8 = 2
	skuConsumable        uint8 = 3
	skuSubscription      uint8 = 5
	skuSubscriptionGroup uint8 = 6

	skuFlagAvailable         uint64 = 1 << 2
	skuFlagGuildSubscription uint64 = 1 << 7
)

type sku struct {
	Name          string        `json:"name"`
	Slug          string        `json:"slug"`
	ID            discord.SkuID `json:"id"`
	ApplicationID discord.AppID `json:"application_id"`
	Flags         uint64        `json:"flags"`
	Type          uint8         `json:"type"`
}

func (s sku) isSubscription() bool { return s.Type == skuSubscription }
func (s sku) available() bool      { return s.Flags&skuFlagAvailable != 0 }
func (s sku) guildSubscription() bool {
	return s.Flags&skuFlagGuildSubscription != 0
}

const (
	subscriptionActive   uint8 = 0
	subscriptionInactive uint8 = 1
	subscriptionEnding   uint8 = 2
)

type subscription struct {
	CurrentPeriodEnd time.Time              `json:"current_period_end"`
	CanceledAt       *time.Time             `json:"canceled_at"`
	SkuIDs           []discord.SkuID        `json:"sku_ids"`
	ID               discord.SubscriptionID `json:"id"`
	UserID           discord.UserID         `json:"user_id"`
	Status           uint8                  `json:"status"`
}

// Shared across subcommands and autocomplete via the SKU cache.
type premiumCommand struct {
	client   *tempest.Client
	skuCache []sku
	mu       sync.Mutex
}

func newPremiumCommand(client *tempest.Client) *premiumCommand {
	return &premiumCommand{client: client}
}

func (c *premiumCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "premium",
		Description:         "Look at this app's SKUs, your entitlements, and your subscriptions.",
		SlashCommandHandler: c.handleRoot,
	}
}

func (c *premiumCommand) handleRoot(ctx context.Context, itx *tempest.CommandInteraction) error {
	return itx.SendLinearReply(ctx, "Pick a /premium subcommand.", true)
}

func (c *premiumCommand) fetchSKUs(ctx context.Context) ([]sku, error) {
	c.mu.Lock()
	cached := c.skuCache
	c.mu.Unlock()

	if cached != nil {
		return cached, nil
	}

	endpoint := "/applications/" + c.client.ApplicationID.String() + "/skus"
	var skus []sku
	if err := c.client.Rest.Request(ctx, http.MethodGet, endpoint, nil, &skus); err != nil {
		return nil, fmt.Errorf("fetch SKUs: %w", err)
	}

	c.mu.Lock()
	c.skuCache = skus
	c.mu.Unlock()

	return skus, nil
}

func (c *premiumCommand) status(ctx context.Context, itx *tempest.CommandInteraction) error {
	user := itx.BaseUser()
	if user == nil {
		return errors.New("the interaction carries no user")
	}

	userEntitlements, err := c.client.FetchEntitlementsPage(ctx, "user_id="+user.ID.String()+"&exclude_ended=true")
	if err != nil {
		return err
	}

	var guildEntitlements []discord.Entitlement
	if itx.GuildID.Snowflake != 0 {
		guildEntitlements, err = c.client.FetchEntitlementsPage(ctx, "guild_id="+itx.GuildID.String()+"&exclude_ended=true")
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

	return itx.SendLinearReply(ctx, b.String(), true)
}

func (c *premiumCommand) skus(ctx context.Context, itx *tempest.CommandInteraction) error {
	skus, err := c.fetchSKUs(ctx)
	if err != nil {
		return err
	}

	if len(skus) == 0 {
		return itx.SendLinearReply(ctx, "This app has no SKUs.", true)
	}

	var b strings.Builder
	b.WriteString("### SKUs\n")

	for _, sku := range skus {
		if sku.Type == skuSubscriptionGroup {
			continue // System-generated companion of a subscription; not sold.
		}

		fmt.Fprintf(&b, "- **%s** (`%s`) %s", sku.Name, sku.Slug, skuKind(sku))
		if sku.available() {
			b.WriteString(", available")
		}
		b.WriteByte('\n')
	}

	return itx.SendLinearReply(ctx, b.String(), true)
}

func (c *premiumCommand) subscriptions(ctx context.Context, itx *tempest.CommandInteraction) error {
	user := itx.BaseUser()
	if user == nil {
		return errors.New("the interaction carries no user")
	}

	skuID, ok := util.SnowflakeOption(itx, "sku")
	if !ok {
		return errors.New("pick a SKU")
	}

	endpoint := "/skus/" + skuID.String() + "/subscriptions?user_id=" + user.ID.String()
	var subscriptions []subscription
	if err := c.client.Rest.Request(ctx, http.MethodGet, endpoint, nil, &subscriptions); err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		return itx.SendLinearReply(ctx, "You have no subscription for that SKU.", true)
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

	return itx.SendLinearReply(ctx, b.String(), true)
}

func (c *premiumCommand) autocomplete(ctx context.Context, itx *tempest.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	typed := strings.ToLower(util.FocusedString(itx))

	skus, err := c.fetchSKUs(ctx)
	if err != nil {
		return nil, err
	}

	choices := make([]discord.CommandOptionChoice, 0, len(skus))
	for _, sku := range skus {
		if !sku.isSubscription() {
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

func writeEntitlements(b *strings.Builder, entitlements []discord.Entitlement, now time.Time) {
	active := 0

	for _, entitlement := range entitlements {
		if !entitlementActive(entitlement, now) {
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

func entitlementActive(entitlement discord.Entitlement, now time.Time) bool {
	if entitlement.Deleted || entitlement.Consumed {
		return false
	}

	if entitlement.StartsAt != nil && now.Before(*entitlement.StartsAt) {
		return false
	}

	return entitlement.EndsAt == nil || now.Before(*entitlement.EndsAt)
}

func skuKind(sku sku) string {
	switch sku.Type {
	case skuDurable:
		return "durable one-time purchase"
	case skuConsumable:
		return "consumable one-time purchase"
	case skuSubscription:
		if sku.guildSubscription() {
			return "guild subscription"
		}
		return "user subscription"
	case skuSubscriptionGroup:
		return "subscription group"
	default:
		return "unknown"
	}
}

func entitlementKind(kind discord.EntitlementType) string {
	switch kind {
	case discord.PurchaseEntitlementType:
		return "purchase"
	case discord.ApplicationSubscriptionEntitlementType:
		return "app subscription"
	case discord.DeveloperGiftEntitlementType:
		return "developer gift"
	case discord.UserGiftEntitlementType:
		return "user gift"
	case discord.TestModePurchaseEntitlementType:
		return "test purchase"
	case discord.FreePurchaseEntitlementType:
		return "free purchase"
	case discord.PremiumSubscriptionEntitlementType:
		return "Nitro"
	case discord.PremiumPurchaseEntitlementType:
		return "Nitro claim"
	default:
		return "unknown"
	}
}

func subscriptionStatus(status uint8) string {
	switch status {
	case subscriptionActive:
		return "active"
	case subscriptionInactive:
		return "inactive"
	case subscriptionEnding:
		return "ending"
	default:
		return "unknown"
	}
}
