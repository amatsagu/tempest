package discord

import "time"

// https://docs.discord.com/developers/resources/entitlement#entitlement-object-entitlement-types
type EntitlementType uint8

const (
	// EntitlementPurchase was bought by the user.
	EntitlementPurchase EntitlementType = 1
	// EntitlementPremiumSubscription covers Discord Nitro.
	EntitlementPremiumSubscription EntitlementType = 2
	// EntitlementDeveloperGift was gifted by the developer.
	EntitlementDeveloperGift EntitlementType = 3
	// EntitlementTestModePurchase was bought by a developer in test mode.
	EntitlementTestModePurchase EntitlementType = 4
	// EntitlementFreePurchase was granted because the SKU is free.
	EntitlementFreePurchase EntitlementType = 5
	// EntitlementUserGift was gifted by another user.
	EntitlementUserGift EntitlementType = 6
	// EntitlementPremiumPurchase was claimed for free by a Nitro subscriber.
	EntitlementPremiumPurchase EntitlementType = 7
	// EntitlementApplicationSubscription is an app subscription.
	EntitlementApplicationSubscription EntitlementType = 8
)

// https://docs.discord.com/developers/resources/entitlement#create-test-entitlement-create-test-entitlement-json-params
type EntitlementOwnerType uint8

const (
	// EntitlementOwnerGuild grants a guild subscription.
	EntitlementOwnerGuild EntitlementOwnerType = iota + 1
	// EntitlementOwnerUser grants a user subscription.
	EntitlementOwnerUser
)

// https://docs.discord.com/developers/resources/entitlement#entitlement-object
type Entitlement struct {
	StartsAt       *time.Time      `json:"starts_at,omitzero"`
	EndsAt         *time.Time      `json:"ends_at,omitzero"`
	PromotionID    Snowflake       `json:"promotion_id,omitzero"`
	ID             EntitlementID   `json:"id"`
	SkuID          SkuID           `json:"sku_id"`
	ApplicationID  AppID           `json:"application_id"`
	SubscriptionID SubscriptionID  `json:"subscription_id,omitzero"`
	UserID         UserID          `json:"user_id,omitzero"`
	GuildID        GuildID         `json:"guild_id,omitzero"`
	GiftCodeFlags  BitSet          `json:"gift_code_flags,omitzero"`
	Type           EntitlementType `json:"type"`
	Deleted        bool            `json:"deleted,omitzero"`
	Consumed       bool            `json:"consumed,omitzero"`
}

// Active reports whether the entitlement grants access at the given time: it must
// not be deleted or consumed, and the time must fall inside its window. This is the
// check to gate premium features on, rather than a subscription's status.
func (e Entitlement) Active(at time.Time) bool {
	if e.Deleted || e.Consumed {
		return false
	}

	if e.StartsAt != nil && at.Before(*e.StartsAt) {
		return false
	}

	return e.EndsAt == nil || at.Before(*e.EndsAt)
}

// https://docs.discord.com/developers/resources/entitlement#create-test-entitlement-create-test-entitlement-json-params
type TestEntitlementPayload struct {
	// SKUID is the subscription or item to grant.
	SKUID SkuID `json:"sku_id"`
	// OwnerID is the user or guild that receives it.
	OwnerID Snowflake `json:"owner_id"`
	// OwnerType chooses whether OwnerID is read as a guild or a user.
	OwnerType EntitlementOwnerType `json:"owner_type"`
}

// EntitlementFilter narrows ListEntitlements. The zero value asks for the
// application's entitlements with Discord's defaults.
type EntitlementFilter struct {
	// SkuIDs limits the result to these SKUs.
	SkuIDs  []SkuID
	UserID  UserID
	GuildID GuildID
	Before  EntitlementID
	After   EntitlementID
	// Limit caps the result; Discord allows 1 to 100 and defaults to 100.
	Limit int
	// ExcludeEnded omits entitlements whose window already ended.
	ExcludeEnded bool
	// IncludeDeleted also returns deleted entitlements. Discord excludes them, so
	// the zero value of this field matches the API default.
	IncludeDeleted bool
}
