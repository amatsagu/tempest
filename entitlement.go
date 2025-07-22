package tempest

import "time"

// https://discord.com/developers/docs/resources/entitlement#entitlement-object-entitlement-types
type EntitlementType uint8

const (
	PURCHASE_ENTITLEMENT_TYPE EntitlementType = iota + 1
	PREMIUM_SUBSCRIPTION_ENTITLEMENT_TYPE
	DEVELOPER_GIFT_ENTITLEMENT_TYPE
	TEST_MODE_PURCHASE_ENTITLEMENT_TYPE
	FREE_PURCHASE_ENTITLEMENT_TYPE
	USER_GIFT_ENTITLEMENT_TYPE
	PREMIUM_PURCHASE_ENTITLEMENT_TYPE
	APPLICATION_SUBSCRIPTION_ENTITLEMENT_TYPE
)

// Entitlements in Discord represent that a user or guild has access to a premium offering in your application.
//
// Refer to the Monetization Overview for more information on how to use Entitlements in your app.
//
// https://discord.com/developers/docs/resources/entitlement#entitlement-object
type Entitlement struct {
	ID            Snowflake       `json:"id"`
	SkuID         Snowflake       `json:"sku_id"`
	ApplicationID Snowflake       `json:"application_id"`
	UserID        Snowflake       `json:"user_id,omitempty"` // ID of the user that is granted access to the entitlement's sku
	Type          EntitlementType `json:"type"`
	Deleted       bool            `json:"deleted,omitempty"` // Whether entitlement was deleted
	StartsAt      *time.Time      `json:"starts_at"`
	EndsAt        *time.Time      `json:"ends_at"`
	GuildID       Snowflake       `json:"guild_id,omitempty"`
	Consumed      bool            `json:"consumed,omitempty"` // Whether entitlement was already used
}

// https://discord.com/developers/docs/resources/entitlement#create-test-entitlement-json-params
type TestEntitlementPayload struct {
	SkuID     Snowflake `json:"sku_id"`
	OwnerID   Snowflake `json:"owner_id"`
	OwnerType uint8     `json:"owner_type"` // 1 for a guild subscription, 2 for a user subscription
}
