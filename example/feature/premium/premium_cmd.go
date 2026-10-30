// Package premium demonstrates the monetization data available through master's
// REST surface: SKUs, entitlements, and subscriptions.
package premium

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// Discord SKU types.
const (
	skuDurable           uint8 = 2
	skuConsumable        uint8 = 3
	skuSubscription      uint8 = 5
	skuSubscriptionGroup uint8 = 6
)

// Discord SKU flags.
const (
	skuFlagAvailable         uint64 = 1 << 2
	skuFlagGuildSubscription uint64 = 1 << 7
)

// sku is the subset of Discord's SKU object this example displays.
type sku struct {
	ID            tempest.Snowflake `json:"id"`
	ApplicationID tempest.Snowflake `json:"application_id"`
	Type          uint8             `json:"type"`
	Name          string            `json:"name"`
	Slug          string            `json:"slug"`
	Flags         uint64            `json:"flags"`
}

func (s sku) isSubscription() bool { return s.Type == skuSubscription }
func (s sku) available() bool      { return s.Flags&skuFlagAvailable != 0 }
func (s sku) guildSubscription() bool {
	return s.Flags&skuFlagGuildSubscription != 0
}

// Subscription statuses.
const (
	subscriptionActive   uint8 = 0
	subscriptionInactive uint8 = 1
	subscriptionEnding   uint8 = 2
)

// subscription is the subset of Discord's subscription object this example uses.
type subscription struct {
	CurrentPeriodEnd time.Time           `json:"current_period_end"`
	CanceledAt       *time.Time          `json:"canceled_at"`
	ID               tempest.Snowflake   `json:"id"`
	UserID           tempest.Snowflake   `json:"user_id"`
	SkuIDs           []tempest.Snowflake `json:"sku_ids"`
	Status           uint8               `json:"status"`
}

// premiumCommand answers /premium. It caches the SKU list because the status
// command, the subscription command, and autocomplete all need it.
type premiumCommand struct {
	client   *tempest.BaseClient
	skuCache []sku
	mu       sync.Mutex
}

func newPremiumCommand(client *tempest.BaseClient) *premiumCommand {
	return &premiumCommand{client: client}
}

func (c *premiumCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "premium",
		Description:         "Look at this app's SKUs, your entitlements, and your subscriptions.",
		SlashCommandHandler: util.CommandHandler(c.handleRoot),
	}
}

func (c *premiumCommand) handleRoot(itx *tempest.CommandInteraction) error {
	return itx.SendLinearReply("Pick a /premium subcommand.", true)
}

// fetchSKUs returns the cached SKU list, loading it on first use.
func (c *premiumCommand) fetchSKUs() ([]sku, error) {
	c.mu.Lock()
	cached := c.skuCache
	c.mu.Unlock()

	if cached != nil {
		return cached, nil
	}

	endpoint := "/applications/" + c.client.ApplicationID.String() + "/skus"
	var skus []sku
	if err := c.client.Rest.Request(http.MethodGet, endpoint, nil, &skus); err != nil {
		return nil, fmt.Errorf("fetch SKUs: %w", err)
	}

	c.mu.Lock()
	c.skuCache = skus
	c.mu.Unlock()

	return skus, nil
}

// status lists the active entitlements of the invoking user and of the server.
func (c *premiumCommand) status(itx *tempest.CommandInteraction) error {
	user := itx.BaseUser()
	if user == nil {
		return errors.New("the interaction carries no user")
	}

	userEntitlements, err := c.client.FetchEntitlementsPage("user_id=" + user.ID.String() + "&exclude_ended=true")
	if err != nil {
		return err
	}

	var guildEntitlements []tempest.Entitlement
	if itx.GuildID != 0 {
		guildEntitlements, err = c.client.FetchEntitlementsPage("guild_id=" + itx.GuildID.String() + "&exclude_ended=true")
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

	return itx.SendLinearReply(b.String(), true)
}

// skus lists every SKU the app offers, marking the ones that can be bought.
func (c *premiumCommand) skus(itx *tempest.CommandInteraction) error {
	skus, err := c.fetchSKUs()
	if err != nil {
		return err
	}

	if len(skus) == 0 {
		return itx.SendLinearReply("This app has no SKUs.", true)
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

	return itx.SendLinearReply(b.String(), true)
}

// subscriptions lists the invoking user's subscriptions for the chosen SKU.
func (c *premiumCommand) subscriptions(itx *tempest.CommandInteraction) error {
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
	if err := c.client.Rest.Request(http.MethodGet, endpoint, nil, &subscriptions); err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		return itx.SendLinearReply("You have no subscription for that SKU.", true)
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

	return itx.SendLinearReply(b.String(), true)
}

// autocomplete suggests the subscription SKUs by name.
func (c *premiumCommand) autocomplete(itx tempest.CommandInteraction) []tempest.CommandOptionChoice {
	typed := focusedString(itx)

	skus, err := c.fetchSKUs()
	if err != nil {
		// Autocomplete has to answer with choices, so a failure suggests nothing.
		return nil
	}

	choices := make([]tempest.CommandOptionChoice, 0, len(skus))
	for _, sku := range skus {
		if !sku.isSubscription() {
			continue
		}

		if typed != "" && !strings.Contains(strings.ToLower(sku.Name), typed) {
			continue
		}

		choices = append(choices, tempest.CommandOptionChoice{Name: sku.Name, Value: sku.ID.String()})
		if len(choices) == 25 {
			break
		}
	}

	return choices
}

// writeEntitlements appends one line per active entitlement; deleted, consumed,
// and expired ones are skipped.
func writeEntitlements(b *strings.Builder, entitlements []tempest.Entitlement, now time.Time) {
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

func entitlementActive(entitlement tempest.Entitlement, now time.Time) bool {
	if entitlement.Deleted || entitlement.Consumed {
		return false
	}

	if entitlement.StartsAt != nil && now.Before(*entitlement.StartsAt) {
		return false
	}

	return entitlement.EndsAt == nil || now.Before(*entitlement.EndsAt)
}

// skuKind names a SKU type for display.
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

// entitlementKind names an entitlement type for display.
func entitlementKind(kind tempest.EntitlementType) string {
	switch kind {
	case tempest.PURCHASE_ENTITLEMENT_TYPE:
		return "purchase"
	case tempest.APPLICATION_SUBSCRIPTION_ENTITLEMENT_TYPE:
		return "app subscription"
	case tempest.DEVELOPER_GIFT_ENTITLEMENT_TYPE:
		return "developer gift"
	case tempest.USER_GIFT_ENTITLEMENT_TYPE:
		return "user gift"
	case tempest.TEST_MODE_PURCHASE_ENTITLEMENT_TYPE:
		return "test purchase"
	case tempest.FREE_PURCHASE_ENTITLEMENT_TYPE:
		return "free purchase"
	case tempest.PREMIUM_SUBSCRIPTION_ENTITLEMENT_TYPE:
		return "Nitro"
	case tempest.PREMIUM_PURCHASE_ENTITLEMENT_TYPE:
		return "Nitro claim"
	default:
		return "unknown"
	}
}

// subscriptionStatus names a subscription status for display.
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

func focusedString(itx tempest.CommandInteraction) string {
	for _, option := range itx.Data.Options {
		if !option.Focused {
			continue
		}

		value, _ := option.Value.(string)
		return strings.ToLower(value)
	}
	return ""
}
