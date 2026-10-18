package discord

import (
	"encoding/json/jsontext"
	"time"
)

// https://docs.discord.com/developers/resources/sku#sku-object-sku-types
type SKUType uint8

const (
	// SKUDurable is a one-time purchase the buyer keeps.
	SKUDurable SKUType = 2
	// SKUConsumable is a one-time purchase that is used up.
	SKUConsumable SKUType = 3
	// SKUSubscription is a recurring subscription; use this one.
	SKUSubscription SKUType = 5
	// SKUSubscriptionGroup is the system-generated group for each subscription.
	SKUSubscriptionGroup SKUType = 6
)

// https://docs.discord.com/developers/resources/sku#sku-object-sku-flags
type SKUFlags = BitSet

const (
	// SKUFlagAvailable means the SKU can be purchased.
	SKUFlagAvailable SKUFlags = 1 << 2
	// SKUFlagGuildSubscription means the subscription applies to one guild and
	// grants access to every member of it.
	SKUFlagGuildSubscription SKUFlags = 1 << 7
	// SKUFlagUserSubscription means the subscription belongs to the buying user and
	// grants access to them everywhere.
	SKUFlagUserSubscription SKUFlags = 1 << 8
)

// https://docs.discord.com/developers/resources/sku#sku-object
type SKU struct {
	ReleaseDate *time.Time `json:"release_date,omitzero"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	// ManifestLabels and Features are returned by the API but are not documented,
	// so they are kept raw.
	ManifestLabels jsontext.Value `json:"manifest_labels,omitzero"`
	Features       jsontext.Value `json:"features,omitzero"`
	ApplicationID  AppID          `json:"application_id"`
	ID             SkuID          `json:"id"`
	DependentSkuID SkuID          `json:"dependent_sku_id,omitzero"`
	Flags          SKUFlags       `json:"flags,omitzero"`
	AccessType     uint8          `json:"access_type,omitzero"`
	Type           SKUType        `json:"type"`
	Premium        bool           `json:"premium,omitzero"`
	ShowAgeGate    bool           `json:"show_age_gate,omitzero"`
}

// IsSubscription reports whether the SKU is a recurring subscription. Discord
// creates a matching SUBSCRIPTION_GROUP SKU for each one; that group is not used.
func (s SKU) IsSubscription() bool {
	return s.Type == SKUSubscription
}

// Available reports whether the SKU can be purchased right now.
func (s SKU) Available() bool {
	return s.Flags.Has(SKUFlagAvailable)
}

// GuildSubscription reports whether the subscription is bought for a guild, which
// entitles every member of it.
func (s SKU) GuildSubscription() bool {
	return s.Flags.Has(SKUFlagGuildSubscription)
}

// UserSubscription reports whether the subscription is bought by a user for
// themselves, which entitles them everywhere.
func (s SKU) UserSubscription() bool {
	return s.Flags.Has(SKUFlagUserSubscription)
}
