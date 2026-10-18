package discord

import "time"

// https://docs.discord.com/developers/resources/subscription#subscription-statuses
type SubscriptionStatus uint8

const (
	// SubscriptionActive is scheduled to renew.
	SubscriptionActive SubscriptionStatus = iota
	// SubscriptionInactive is not being charged.
	SubscriptionInactive
	// SubscriptionEnding is active but will not renew.
	SubscriptionEnding
)

// https://docs.discord.com/developers/resources/subscription#subscription-object
type Subscription struct {
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	CanceledAt         *time.Time         `json:"canceled_at,omitzero"`
	Country            string             `json:"country,omitzero"`
	SkuIDs             []SkuID            `json:"sku_ids,omitzero"`
	EntitlementIDs     []EntitlementID    `json:"entitlement_ids,omitzero"`
	RenewalSkuIDs      []SkuID            `json:"renewal_sku_ids,omitzero"`
	ID                 SubscriptionID     `json:"id"`
	UserID             UserID             `json:"user_id"`
	Status             SubscriptionStatus `json:"status"`
}

// SubscriptionFilter narrows ListSKUSubscriptions. UserID is required unless the
// caller authenticates with an OAuth2 token instead of a bot token.
type SubscriptionFilter struct {
	UserID UserID
	Before SubscriptionID
	After  SubscriptionID
	// Limit caps the result; Discord allows 1 to 100 and defaults to 50.
	Limit int
}
