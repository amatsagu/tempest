package discord

import "strings"

// https://docs.discord.com/developers/topics/permissions#role-object-role-flags
type RoleFlags = BitSet

const (
	RoleFlagInPrompt RoleFlags = 1 << 0
)

// https://docs.discord.com/developers/topics/permissions#role-object-role-tags-structure
type RoleTag struct {
	BotID                 UserID        `json:"bot_id,omitzero"`
	IntegrationID         IntegrationID `json:"integration_id,omitzero"`
	SubscriptionListingID SkuID         `json:"subscription_listing_id,omitzero"`
	PremiumSubscriber     bool          `json:"premium_subscriber,omitzero"`
	AvailableForPurchase  bool          `json:"available_for_purchase,omitzero"`
	GuildConnections      bool          `json:"guild_connections,omitzero"`
}

// https://docs.discord.com/developers/topics/permissions#role-object-role-colors-object
type RoleColors struct {
	SecondaryColor *uint32 `json:"secondary_color,omitzero"`
	TertiaryColor  *uint32 `json:"tertiary_color,omitzero"`
	PrimaryColor   uint32  `json:"primary_color"`
}

// https://docs.discord.com/developers/topics/permissions#role-object
type Role struct {
	Tags         *RoleTag    `json:"tags,omitzero"`
	Colors       *RoleColors `json:"colors,omitzero"`
	Name         string      `json:"name"`
	Icon         string      `json:"icon,omitzero"`
	UnicodeEmoji string      `json:"unicode_emoji,omitzero"`
	ID           RoleID      `json:"id"`
	Permissions  Permissions `json:"permissions,string"`
	Color        uint32      `json:"color"`
	Position     int         `json:"position"`
	Flags        RoleFlags   `json:"flags,omitzero"`
	Hoist        bool        `json:"hoist,omitzero"`
	Managed      bool        `json:"managed,omitzero"`
	Mentionable  bool        `json:"mentionable,omitzero"`
}

func (r Role) Mention() string {
	return r.ID.Mention()
}

func (r Role) IconURL() string {
	if r.Icon == "" {
		return ""
	}
	if strings.HasPrefix(r.Icon, "a_") {
		return CDNBaseURL + "/role-icons/" + r.ID.String() + "/" + r.Icon + ".webp?animated=true"
	}
	return CDNBaseURL + "/role-icons/" + r.ID.String() + "/" + r.Icon + ".png"
}
