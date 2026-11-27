package discord

// https://docs.discord.com/developers/resources/guild#guild-object-default-message-notification-level
type MessageNotificationLevel uint8

const (
	AllMessagesNotificationLevel MessageNotificationLevel = iota
	OnlyMentionsNotificationLevel
)

// https://docs.discord.com/developers/resources/guild#guild-object-explicit-content-filter-level
type ExplicitContentFilter uint8

const (
	DisabledContentFilter ExplicitContentFilter = iota
	MembersWithoutRolesContentFilter
	AllMembersContentFilter
)

// https://docs.discord.com/developers/resources/guild#guild-object-mfa-level
type MFALevel uint8

const (
	NoneMFALevel MFALevel = iota
	ElevatedMFALevel
)

// https://docs.discord.com/developers/resources/guild#guild-object-verification-level
type VerificationLevel uint8

const (
	NoneVerificationLevel VerificationLevel = iota
	LowVerificationLevel
	MediumVerificationLevel
	HighVerificationLevel
	VeryHighVerificationLevel
)

// https://docs.discord.com/developers/resources/guild#guild-object-guild-nsfw-level
type NSFWLevel uint8

const (
	DefaultNSFWLevel NSFWLevel = iota
	ExplicitNSFWLevel
	SafeNSFWLevel
	AgeRestrictedNSFWLevel
)

// https://docs.discord.com/developers/resources/guild#guild-object-system-channel-flags
type SystemChannelFlags = BitSet

const (
	SuppressJoinNotificationsSystemFlag SystemChannelFlags = 1 << iota
	SuppressPremiumSubscriptionsSystemFlag
	SuppressGuildReminderNotificationsSystemFlag
	SuppressJoinNotificationRepliesSystemFlag
	SuppressRoleSubscriptionPurchaseNotificationsSystemFlag
	SuppressRoleSubscriptionPurchaseNotificationRepliesSystemFlag
)

// https://docs.discord.com/developers/resources/guild#guild-object-premium-tier
type PremiumTier uint8

const (
	NonePremiumTier PremiumTier = iota
	Boost1PremiumTier
	Boost2PremiumTier
	Boost3PremiumTier
)

// https://docs.discord.com/developers/resources/guild#guild-object
type Guild struct {
	PreferredLocale           string             `json:"preferred_locale,omitzero"`
	Name                      string             `json:"name"`
	IconHash                  string             `json:"icon,omitzero"` // Hash code used to access guild's icon. Call Guild.IconURL to get direct url.
	VanityURL                 string             `json:"vanity_url_code,omitzero"`
	DiscoverySplashHash       string             `json:"discovery_splash,omitzero"` // Hash code used to access guild's special discovery splash background (only available for "DISCOVERABLE" guilds). Call Guild.DiscoverySplashURL to get direct url.
	Description               string             `json:"description,omitzero"`
	SplashHash                string             `json:"splash,omitzero"` // Hash code used to access guild's splash background. Call Guild.SplashURL to get direct url.
	BannerHash                string             `json:"banner,omitzero"` // Hash code used to access guild's icon. Call Guild.BannerURL to get direct url.
	Emojis                    []Emoji            `json:"emojis,omitzero"`
	Roles                     []Role             `json:"roles,omitzero"`
	Features                  []string           `json:"features,omitzero"` // https://docs.discord.com/developers/resources/guild#guild-object-guild-features
	ID                        GuildID            `json:"id"`
	AFKChannelID              ChannelID          `json:"afk_channel_id,omitzero"`
	OwnerID                   UserID             `json:"owner_id"`
	RulesChannelID            ChannelID          `json:"rules_channel_id,omitzero"`
	WidgetChannelID           ChannelID          `json:"widget_channel_id,omitzero"` // The channel ID that the widget will generate an invite to, or null if set to no invite.
	ApplicationID             AppID              `json:"application_id,omitzero"`    // Application id of the guild creator if it is bot-created (never seen it in use).
	SystemChannelID           ChannelID          `json:"system_channel_id,omitzero"`
	PublicUpdatesChannelID    ChannelID          `json:"public_updates_channel_id,omitzero"`
	SafetyAlertsChannelID     ChannelID          `json:"safety_alerts_channel_id,omitzero"`
	SystemChannelFlags        SystemChannelFlags `json:"system_channel_flags,omitzero"`
	AFKChannelTimeout         uint32             `json:"afk_timeout"`            // AFK timeout value in seconds.
	MaxPresences              uint32             `json:"max_presences,omitzero"` // The maximum number of presences for the guild (null is always returned, apart from the largest of guilds).
	MaxMembers                uint32             `json:"max_members,omitzero"`
	MaxVideoChannelUsers      uint32             `json:"max_video_channel_users,omitzero"`
	MaxStageVideoChannelUsers uint32             `json:"max_stage_video_channel_users,omitzero"`
	PremiumSubscriptionCount  uint32             `json:"premium_subscription_count,omitzero"` // The number of boosts this guild currently has.

	// Some fields were ignored on purpose as they are random junk...

	ApproximateMemberCount      uint32                   `json:"approximate_member_count,omitzero"`
	ApproximatePresenceCount    uint32                   `json:"approximate_presence_count,omitzero"`
	MFALevel                    MFALevel                 `json:"mfa_level"`
	ExplicitContentFilter       ExplicitContentFilter    `json:"explicit_content_filter"`
	DefaultMessageNotifications MessageNotificationLevel `json:"default_message_notifications"`
	NSFWLevel                   NSFWLevel                `json:"nsfw_level"`
	VerificationLevel           VerificationLevel        `json:"verification_level"`
	PremiumTier                 PremiumTier              `json:"premium_tier"`
	WidgetEnabled               bool                     `json:"widget_enabled,omitzero"` // Whether server uses widget.
	PremiumProgressBarEnabled   bool                     `json:"premium_progress_bar_enabled,omitzero"`
}

// Returns a direct url to guild icon. It'll return empty string if there's no custom icon.
func (guild *Guild) IconURL() string {
	return cdnAssetURL("icons", guild.ID.Snowflake, guild.IconHash, "", true)
}

// Returns a direct url to guild splash background. It'll return empty string if there's no custom splash bg.
func (guild *Guild) SplashURL() string {
	return cdnAssetURL("splashes", guild.ID.Snowflake, guild.SplashHash, "", false)
}

// Returns a direct url to guild discovery splash background. It'll return empty string if there's no discovery splash bg.
func (guild *Guild) DiscoverySplashURL() string {
	return cdnAssetURL("discovery-splashes", guild.ID.Snowflake, guild.DiscoverySplashHash, "", false)
}

// Returns a direct url to guild banner. It'll return empty string if there's no custom banner.
func (guild *Guild) BannerURL() string {
	return cdnAssetURL("banners", guild.ID.Snowflake, guild.BannerHash, "", true)
}
