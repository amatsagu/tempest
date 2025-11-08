package tempest

// https://discord.com/developers/docs/resources/guild#unavailable-guild-object
type UnavailableGuild struct {
	ID          Snowflake `json:"id"`
	Unavailable bool      `json:"unavailable"`
}

// https://discord.com/developers/docs/resources/guild#guild-object-default-message-notification-level
type MessageNotificationLevel uint8

const (
	ALL_MESSAGES_NOTIFICATION_LEVEL MessageNotificationLevel = iota
	ONLY_MENTIONS_NOTIFICATION_LEVEL
)

// https://discord.com/developers/docs/resources/guild#guild-object-explicit-content-filter-level
type ExplicitContentFilter uint8

const (
	DISABLED_CONTENT_FILTER ExplicitContentFilter = iota
	MEMBERS_WITHOUT_ROLES_CONTENT_FILTER
	ALL_MEMBERS_CONTENT_FILTER
)

// https://discord.com/developers/docs/resources/guild#guild-object-mfa-level
type MFALevel uint8

const (
	NONE_MFA_LEVEL MFALevel = iota
	ELEVATED_MFA_LEVEL
)

// https://discord.com/developers/docs/resources/guild#guild-object-system-channel-flags
type SystemChannelFlags BitSet

const (
	SUPPRESS_JOIN_NOTIFICATIONS_SYSTEM_FLAG = 1 << iota
	SUPPRESS_PREMIUM_SUBSCRIPTIONS_SYSTEM_FLAG
	SUPPRESS_GUILD_REMINDER_NOTIFICATIONS_SYSTEM_FLAG
	SUPPRESS_JOIN_NOTIFICATION_REPLIES_SYSTEM_FLAG
	SUPPRESS_ROLE_SUBSCRIPTION_PURCHASE_NOTIFICATIONS_SYSTEM_FLAG
	SUPPRESS_ROLE_SUBSCRIPTION_PURCHASE_NOTIFICATION_REPLIES_SYSTEM_FLAG
)

// https://discord.com/developers/docs/resources/guild#guild-object-premium-tier
type PremiumTier uint8

const (
	NONE_PREMIUM_TIER PremiumTier = iota
	BOOST_1_PREMIUM_TIER
	BOOST_2_PREMIUM_TIER
	BOOST_3_PREMIUM_TIER
)

// https://discord.com/developers/docs/resources/guild#guild-object-guild-structure
type Guild struct {
	ID                          Snowflake                `json:"id"`
	Name                        string                   `json:"name"`
	AvatarHash                  string                   `json:"icon,omitempty"`             // Hash code used to access guild's icon. Call Guild.IconURL to get direct url.
	SplashHash                  string                   `json:"splash,omitempty"`           // Hash code used to access guild's splash background. Call Guild.SplashURL to get direct url.
	DiscoverySplashHash         string                   `json:"discovery_splash,omitempty"` // Hash code used to access guild's special discovery splash background (only available for "DISCOVERABLE" guilds). Call Guild.DiscoverySplashURL to get direct url.
	OwnerID                     Snowflake                `json:"owner_id"`
	AFKChannelID                Snowflake                `json:"afk_channel_id,omitempty"`
	AFKChannelTimeout           uint32                   `json:"afk_timeout"`                 // AFK timeout value in seconds.
	WidgetEnabled               bool                     `json:"widget_enabled"`              // Whether server uses widget.
	WidgetChannelID             Snowflake                `json:"widget_channel_id,omitempty"` // The channel ID that the widget will generate an invite to, or null if set to no invite.
	DefaultMessageNotifications MessageNotificationLevel `json:"default_message_notifications"`
	ExplicitContentFilter       ExplicitContentFilter    `json:"explicit_content_filter"`
	Roles                       []Role                   `json:"roles,omitzero"`
	Emojis                      []Emoji                  `json:"emojis,omitzero"`
	Features                    []string                 `json:"features,omitzero"` // // https://discord.com/developers/docs/resources/guild#guild-object-guild-features
	MFALevel                    MFALevel                 `json:"mfa_level"`
	ApplicationID               Snowflake                `json:"application_id,omitempty"` // Application id of the guild creator if it is bot-created (never seen it in use).
	SystemChannelID             Snowflake                `json:"system_channel_id,omitempty"`
	SystemChannelFlags          SystemChannelFlags       `json:"system_channel_flags"`
	RulesChannelID              Snowflake                `json:"rules_channel_id,omitempty"`
	MaxPresences                uint32                   `json:"max_presences,omitempty"` // The maximum number of presences for the guild (null is always returned, apart from the largest of guilds).
	MaxMembers                  uint32                   `json:"max_members,omitempty"`
	VanityURL                   string                   `json:"vanity_url_code,omitempty"`
	Description                 string                   `json:"description,omitempty"`
	BannerHash                  string                   `json:"banner,omitempty"` // Hash code used to access guild's icon. Call Guild.BannerURL to get direct url.
	PremiumTier                 PremiumTier              `json:"premium_tier"`
	PremiumSubscriptionCount    uint32                   `json:"premium_subscription_count,omitempty"` // The number of boosts this guild currently has.
	PrefferedLocale             string                   `json:"preferred_locale,omitempty"`

	// Some fields were ignored on purpose as they are random junk...

	ApproximateMemberCount    uint32 `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount  uint32 `json:"approximate_presence_count,omitempty"`
	PremiumProgressBarEnabled bool   `json:"premium_progress_bar_enabled"`
}
