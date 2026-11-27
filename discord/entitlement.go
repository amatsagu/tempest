package discord

import "time"

// https://docs.discord.com/developers/resources/entitlement#entitlement-object-entitlement-types
type EntitlementType uint8

const (
	PurchaseEntitlementType EntitlementType = iota + 1
	PremiumSubscriptionEntitlementType
	DeveloperGiftEntitlementType
	TestModePurchaseEntitlementType
	FreePurchaseEntitlementType
	UserGiftEntitlementType
	PremiumPurchaseEntitlementType
	ApplicationSubscriptionEntitlementType
)

// Entitlements in Discord represent that a user or guild has access to a premium offering in your application.
//
// Refer to the Monetization Overview for more information on how to use Entitlements in your app.
//
// https://docs.discord.com/developers/resources/entitlement#entitlement-object
type Entitlement struct {
	StartsAt      *time.Time      `json:"starts_at,omitzero"`
	EndsAt        *time.Time      `json:"ends_at,omitzero"`
	ID            EntitlementID   `json:"id"`
	SkuID         SkuID           `json:"sku_id"`
	ApplicationID AppID           `json:"application_id"`
	UserID        UserID          `json:"user_id,omitzero"` // ID of the user that is granted access to the entitlement's sku
	GuildID       GuildID         `json:"guild_id,omitzero"`
	Type          EntitlementType `json:"type"`
	Deleted       bool            `json:"deleted,omitzero"`  // Whether entitlement was deleted
	Consumed      bool            `json:"consumed,omitzero"` // Whether entitlement was already used
}

// https://docs.discord.com/developers/resources/entitlement#create-test-entitlement-json-params
type EntitlementOwnerType uint8

const (
	GuildEntitlementOwnerType EntitlementOwnerType = iota + 1
	UserEntitlementOwnerType
)

// https://docs.discord.com/developers/resources/entitlement#create-test-entitlement-json-params
type TestEntitlementPayload struct {
	SkuID     SkuID                `json:"sku_id"`
	OwnerID   Snowflake            `json:"owner_id"`
	OwnerType EntitlementOwnerType `json:"owner_type"`
}
