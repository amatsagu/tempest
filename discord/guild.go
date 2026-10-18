package discord

import (
	"strings"
	"time"
)

// https://docs.discord.com/developers/resources/guild#guild-object-default-message-notification-level
type MessageNotificationLevel uint8

const (
	NotificationAllMessages  MessageNotificationLevel = 0
	NotificationOnlyMentions MessageNotificationLevel = 1
)

// https://docs.discord.com/developers/resources/guild#guild-object-explicit-content-filter-level
type ExplicitContentFilter uint8

const (
	ContentFilterDisabled            ExplicitContentFilter = 0
	ContentFilterMembersWithoutRoles ExplicitContentFilter = 1
	ContentFilterAllMembers          ExplicitContentFilter = 2
)

// https://docs.discord.com/developers/resources/guild#guild-object-mfa-level
type MFALevel uint8

const (
	MFANone     MFALevel = 0
	MFAElevated MFALevel = 1
)

// https://docs.discord.com/developers/resources/guild#guild-object-verification-level
type VerificationLevel uint8

const (
	VerificationNone     VerificationLevel = 0
	VerificationLow      VerificationLevel = 1
	VerificationMedium   VerificationLevel = 2
	VerificationHigh     VerificationLevel = 3
	VerificationVeryHigh VerificationLevel = 4
)

// https://docs.discord.com/developers/resources/guild#guild-object-guild-nsfw-level
type NSFWLevel uint8

const (
	NSFWLevelDefault       NSFWLevel = 0
	NSFWLevelExplicit      NSFWLevel = 1
	NSFWLevelSafe          NSFWLevel = 2
	NSFWLevelAgeRestricted NSFWLevel = 3
)

// https://docs.discord.com/developers/resources/guild#guild-object-premium-tier
type PremiumTier uint8

const (
	PremiumTierNone PremiumTier = 0
	PremiumTier1    PremiumTier = 1
	PremiumTier2    PremiumTier = 2
	PremiumTier3    PremiumTier = 3
)

// https://docs.discord.com/developers/resources/guild#guild-object-system-channel-flags
type SystemChannelFlags = BitSet

const (
	SystemChannelFlagSuppressJoinNotifications               SystemChannelFlags = 1 << 0
	SystemChannelFlagSuppressPremiumSubscriptions            SystemChannelFlags = 1 << 1
	SystemChannelFlagSuppressGuildReminderNotifications      SystemChannelFlags = 1 << 2
	SystemChannelFlagSuppressJoinNotificationReplies         SystemChannelFlags = 1 << 3
	SystemChannelFlagSuppressRoleSubscriptionPurchase        SystemChannelFlags = 1 << 4
	SystemChannelFlagSuppressRoleSubscriptionPurchaseReplies SystemChannelFlags = 1 << 5
)

// https://docs.discord.com/developers/resources/guild#guild-object
//
// The fields Discord documents but no bot reads (owner, permissions, region,
// welcome_screen, incidents_data, guild_hashes) are left out on purpose.
type Guild struct {
	Description               string             `json:"description,omitzero"`
	Name                      string             `json:"name"`
	IconHash                  string             `json:"icon,omitzero"`
	SplashHash                string             `json:"splash,omitzero"`
	DiscoverySplashHash       string             `json:"discovery_splash,omitzero"`
	BannerHash                string             `json:"banner,omitzero"`
	VanityURLCode             string             `json:"vanity_url_code,omitzero"`
	PreferredLocale           Locale             `json:"preferred_locale,omitzero"`
	Features                  []string           `json:"features,omitzero"`
	Emojis                    []Emoji            `json:"emojis,omitzero"`
	Stickers                  []Sticker          `json:"stickers,omitzero"`
	Roles                     []Role             `json:"roles,omitzero"`
	WidgetChannelID           ChannelID          `json:"widget_channel_id,omitzero"`
	ApplicationID             AppID              `json:"application_id,omitzero"`
	OwnerID                   UserID             `json:"owner_id,omitzero"`
	AFKChannelID              ChannelID          `json:"afk_channel_id,omitzero"`
	SystemChannelID           ChannelID          `json:"system_channel_id,omitzero"`
	RulesChannelID            ChannelID          `json:"rules_channel_id,omitzero"`
	ID                        GuildID            `json:"id"`
	PublicUpdatesChannelID    ChannelID          `json:"public_updates_channel_id,omitzero"`
	SafetyAlertsChannelID     ChannelID          `json:"safety_alerts_channel_id,omitzero"`
	SystemChannelFlags        SystemChannelFlags `json:"system_channel_flags,omitzero"`
	MaxVideoChannelUsers      uint32             `json:"max_video_channel_users,omitzero"`
	MaxStageVideoChannelUsers uint32             `json:"max_stage_video_channel_users,omitzero"`
	ApproximatePresenceCount  uint32             `json:"approximate_presence_count,omitzero"`
	ApproximateMemberCount    uint32             `json:"approximate_member_count,omitzero"`
	PremiumSubscriptionCount  uint32             `json:"premium_subscription_count,omitzero"`
	MaxMembers                uint32             `json:"max_members,omitzero"`
	AFKTimeout                uint32             `json:"afk_timeout"`
	MaxPresences              uint32             `json:"max_presences,omitzero"`
	PremiumTier               PremiumTier        `json:"premium_tier"`
	// These are the enums Discord sends as small integers.
	DefaultMessageNotifications MessageNotificationLevel `json:"default_message_notifications"`
	ExplicitContentFilter       ExplicitContentFilter    `json:"explicit_content_filter"`
	NSFWLevel                   NSFWLevel                `json:"nsfw_level"`
	VerificationLevel           VerificationLevel        `json:"verification_level"`
	MFALevel                    MFALevel                 `json:"mfa_level"`
	// And these are the counters and toggles.
	WidgetEnabled             bool `json:"widget_enabled,omitzero"`
	PremiumProgressBarEnabled bool `json:"premium_progress_bar_enabled,omitzero"`
}

// HasFeature reports whether the guild has the named feature, for example
// "COMMUNITY", "DISCOVERABLE" or "PRUNE_REQUIRES_ADMIN". The full list is at
// https://docs.discord.com/developers/resources/guild#guild-object-guild-features
func (g Guild) HasFeature(feature string) bool {
	for _, have := range g.Features {
		if have == feature {
			return true
		}
	}
	return false
}

// IconURL returns the guild's icon, or "" when it has none.
func (g Guild) IconURL() string {
	if g.IconHash == "" {
		return ""
	}

	if strings.HasPrefix(g.IconHash, "a_") {
		return CDNBaseURL + "/icons/" + g.ID.String() + "/" + g.IconHash + ".webp?animated=true"
	}
	return CDNBaseURL + "/icons/" + g.ID.String() + "/" + g.IconHash + ".png"
}

// SplashURL returns the guild's invite splash, or "" when it has none.
func (g Guild) SplashURL() string {
	if g.SplashHash == "" {
		return ""
	}
	return CDNBaseURL + "/splashes/" + g.ID.String() + "/" + g.SplashHash + ".png"
}

// DiscoverySplashURL returns the guild's discovery splash, or "" when it has none.
func (g Guild) DiscoverySplashURL() string {
	if g.DiscoverySplashHash == "" {
		return ""
	}
	return CDNBaseURL + "/discovery-splashes/" + g.ID.String() + "/" + g.DiscoverySplashHash + ".png"
}

// BannerURL returns the guild's banner, or "" when it has none.
func (g Guild) BannerURL() string {
	if g.BannerHash == "" {
		return ""
	}

	if strings.HasPrefix(g.BannerHash, "a_") {
		return CDNBaseURL + "/banners/" + g.ID.String() + "/" + g.BannerHash + ".webp?animated=true"
	}
	return CDNBaseURL + "/banners/" + g.ID.String() + "/" + g.BannerHash + ".png"
}

// https://docs.discord.com/developers/resources/guild#unavailable-guild-object
type UnavailableGuild struct {
	ID          GuildID `json:"id"`
	Unavailable bool    `json:"unavailable,omitzero"`
}

// GuildCreateData is the GUILD_CREATE payload: the guild plus the state Discord
// sends with it. An unavailable guild only carries the ID and Unavailable.
// https://docs.discord.com/developers/events/gateway-events#guild-create
type GuildCreateData struct {
	*Guild

	JoinedAt    *time.Time `json:"joined_at,omitzero"`
	Members     []Member   `json:"members,omitzero"`
	Channels    []Channel  `json:"channels,omitzero"`
	Threads     []Channel  `json:"threads,omitzero"`
	MemberCount uint32     `json:"member_count,omitzero"`
	Large       bool       `json:"large,omitzero"`
	Unavailable bool       `json:"unavailable,omitzero"`
}
