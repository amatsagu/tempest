package discord

import "time"

// https://docs.discord.com/developers/resources/entitlement#entitlement-object-entitlement-types
type EntitlementType uint8

const (
	EntitlementPurchase EntitlementType = iota + 1
	EntitlementPremiumSubscription
	EntitlementDeveloperGift
	EntitlementTestModePurchase
	EntitlementFreePurchase
	EntitlementUserGift
	EntitlementPremiumPurchase
	EntitlementApplicationSubscription
)

// https://docs.discord.com/developers/resources/sku#sku-object-sku-types
type SKUType uint8

const (
	SKUDurable SKUType = iota + 1
	SKUConsumable
	SKUSubscription
	SKUSubscriptionGroup
)

// https://docs.discord.com/developers/resources/sku#sku-object
type SKU struct {
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	ID            SkuID   `json:"id"`
	ApplicationID AppID   `json:"application_id"`
	Flags         BitSet  `json:"flags,omitzero"`
	Type          SKUType `json:"type"`
}

// https://docs.discord.com/developers/resources/entitlement#entitlement-object
type Entitlement struct {
	StartsAt      *time.Time      `json:"starts_at,omitzero"`
	EndsAt        *time.Time      `json:"ends_at,omitzero"`
	ID            Snowflake       `json:"id"`
	SkuID         SkuID           `json:"sku_id"`
	ApplicationID AppID           `json:"application_id"`
	UserID        UserID          `json:"user_id,omitzero"`
	GuildID       GuildID         `json:"guild_id,omitzero"`
	Type          EntitlementType `json:"type"`
	Deleted       bool            `json:"deleted,omitzero"`
	Consumed      bool            `json:"consumed,omitzero"`
}
