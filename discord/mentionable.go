package discord

import (
	"strconv"
	"strings"
	"time"
)

// https://docs.discord.com/developers/resources/user#user-object-premium-types
type NitroType uint8

const (
	NoNitroType NitroType = iota
	ClassicNitroType
	FullNitroType
	BasicNitroType
)

// https://docs.discord.com/developers/resources/user#avatar-decoration-data-object
type AvatarDecoration struct {
	AssetHash string `json:"asset"` // Hash code used to access user's avatar decoration.
	SkuID     SkuID  `json:"sku_id"`
}

// Returns a direct url to targets's avatar decoration. It'll return empty string if target doesn't use avatar decoration.
func (adc *AvatarDecoration) DecorationURL() string {
	if adc.AssetHash == "" {
		return ""
	}

	base := DiscordCDNBaseURL()
	var suffix string
	if strings.HasPrefix(adc.AssetHash, "a_") {
		suffix = ".webp?animated=true"
	}

	var sb strings.Builder
	sb.Grow(len(base) + len("/avatar-decoration-presets/") + len(adc.AssetHash) + len(suffix))
	sb.WriteString(base)
	sb.WriteString("/avatar-decoration-presets/")
	sb.WriteString(adc.AssetHash)
	sb.WriteString(suffix)
	return sb.String()
}

func cdnAssetURL(kind string, id Snowflake, hash, staticExt string, allowAnimated bool) string {
	if hash == "" {
		return ""
	}

	var numBuf [20]byte
	idBytes := strconv.AppendUint(numBuf[:0], uint64(id), 10)
	base := DiscordCDNBaseURL()

	suffix := staticExt
	if allowAnimated && strings.HasPrefix(hash, "a_") {
		suffix = ".webp?animated=true"
	}

	var sb strings.Builder
	sb.Grow(len(base) + 1 + len(kind) + 1 + len(idBytes) + 1 + len(hash) + len(suffix))
	sb.WriteString(base)
	sb.WriteByte('/')
	sb.WriteString(kind)
	sb.WriteByte('/')
	sb.Write(idBytes)
	sb.WriteByte('/')
	sb.WriteString(hash)
	sb.WriteString(suffix)
	return sb.String()
}

func memberCDNURL(guildID GuildID, user *User, kind, hash, missingGuildPanic string) string {
	if hash == "" {
		return ""
	}
	if guildID.Snowflake == 0 {
		panic(missingGuildPanic)
	}
	if user == nil {
		return ""
	}

	var gidBuf, uidBuf [20]byte
	gidBytes := strconv.AppendUint(gidBuf[:0], uint64(guildID.Snowflake), 10)
	uidBytes := strconv.AppendUint(uidBuf[:0], uint64(user.ID.Snowflake), 10)
	base := DiscordCDNBaseURL()

	var suffix string
	if strings.HasPrefix(hash, "a_") {
		suffix = ".webp?animated=true"
	}

	var sb strings.Builder
	sb.Grow(len(base) + len("/guilds/") + len(gidBytes) + len("/users/") + len(uidBytes) + 1 + len(kind) + 1 + len(hash) + len(suffix))
	sb.WriteString(base)
	sb.WriteString("/guilds/")
	sb.Write(gidBytes)
	sb.WriteString("/users/")
	sb.Write(uidBytes)
	sb.WriteByte('/')
	sb.WriteString(kind)
	sb.WriteByte('/')
	sb.WriteString(hash)
	sb.WriteString(suffix)
	return sb.String()
}

// https://docs.discord.com/developers/resources/user#user-object-user-flags
type UserFlags = BitSet

const (
	DiscordEmployeeUserFlag UserFlags = 1 << iota // Discord Employee, Staff
	PartneredServerOwnerUserFlag
	HypesquadUserFlag
	BugHunterLevel1UserFlag
	_
	_
	HypesquadOnlineHouse1UserFlag
	HypesquadOnlineHouse2UserFlag
	HypesquadOnlineHouse3UserFlag
	PremiumEarlySupporterUserFlag
	TeamUserUserFlag // Discord docs mentions "pseudo user" and that user is a team...
	_
	_
	_
	BugHunterLevel2UserFlag
	_
	VerifiedBotUserFlag
	VerifiedDeveloperUserFlag   // Early Verified Bot Developer
	CertifiedModeratorUserFlag  // Moderator Programs Alumni
	BotHTTPInteractionsUserFlag // Bot/App uses only HTTP interactions and is shown in the online member list.
	_
	_
	ActiveDeveloperUserFlag // User has regular discord developer badge
)

// https://docs.discord.com/developers/resources/user#user-object
type User struct {
	AvatarDecorationData *AvatarDecoration `json:"avatar_decoration_data,omitzero"`
	Username             string            `json:"username"`
	GlobalName           string            `json:"global_name,omitzero"` // User's display name. Tempest lib will make it equal to user.Username if it was empty.
	AvatarHash           string            `json:"avatar,omitzero"`      // Hash code used to access user's profile. Call User.AvatarURL to get direct url.
	BannerHash           string            `json:"banner,omitzero"`      // Hash code used to access user's baner. Call User.BannerURL to get direct url.
	Locale               string            `json:"locale,omitzero"`
	ID                   UserID            `json:"id"`
	PublicFlags          UserFlags         `json:"public_flags,omitzero"` // Public flags on user's account.
	AccentColor          uint32            `json:"accent_color,omitzero"` // User's banner color, encoded as an integer representation of hexadecimal color code.
	Bot                  bool              `json:"bot,omitzero"`          // Whether it's bot/app account.
	System               bool              `json:"system,omitzero"`       // Whether user is Discord System Message account.
	PremiumType          NitroType         `json:"premium_type,omitzero"`
}

func (user User) DisplayName() string {
	if user.GlobalName != "" {
		return user.GlobalName
	}
	return user.Username
}

func (user *User) Mention() string {
	return user.ID.Mention()
}

// Returns a direct url to user's avatar. It'll return url to default Discord's avatar if targeted user don't use avatar.
func (user *User) AvatarURL() string {
	if user.AvatarHash == "" {
		base := DiscordCDNBaseURL()
		var sb strings.Builder
		sb.Grow(len(base) + len("/embed/avatars/0.png"))
		sb.WriteString(base)
		sb.WriteString("/embed/avatars/")
		sb.WriteByte('0' + byte((user.ID.Snowflake>>22)%6))
		sb.WriteString(".png")
		return sb.String()
	}

	return cdnAssetURL("avatars", user.ID.Snowflake, user.AvatarHash, "", true)
}

// Returns a direct url to user's banner. It'll return empty string if targeted user don't use avatar.
func (user *User) BannerURL() string {
	return cdnAssetURL("banners", user.ID.Snowflake, user.BannerHash, "", true)
}

// https://docs.discord.com/developers/resources/guild#guild-member-object-guild-member-flags
type MemberFlags = BitSet

const (
	DidRejoinMemberFlag MemberFlags = 1 << iota
	CompletedOnboardingMemberFlag
	BypassesVerificationMemberFlag
	StartedOnboardingMemberFlag
	IsGuestMemberFlag
	StartedHomeActionsMemberFlag
	CompletedHomeActionsMemberFlag
	AutomodQuarantinedUsernameMemberFlag
	_
	DMSettingsUpsellAcknowledgedMemberFlag
	AutomodQuarantinedGuildTagMemberFlag
)

// https://docs.discord.com/developers/resources/guild#guild-member-object
type Member struct {
	JoinedAt                   time.Time         `json:"joined_at,omitzero"`
	AvatarDecorationData       *AvatarDecoration `json:"avatar_decoration_data,omitzero"`
	CommunicationDisabledUntil *time.Time        `json:"communication_disabled_until,omitzero"`
	User                       *User             `json:"user,omitzero"`
	PremiumSince               *time.Time        `json:"premium_since,omitzero"`
	GuildBannerHash            string            `json:"banner,omitzero"` // Hash code used to access member's custom, guild banner. Call Member.GuildBannerURL to get direct url.
	GuildAvatarHash            string            `json:"avatar,omitzero"` // Hash code used to access member's custom, guild avatar. Call Member.GuildAvatarURL to get direct url.
	Nickname                   string            `json:"nick,omitzero"`
	RoleIDs                    []RoleID          `json:"roles,omitzero"`
	Flags                      MemberFlags       `json:"flags,omitzero"`
	PermissionFlags            PermissionFlags   `json:"permissions,string,omitzero"`

	// It's not part of Member API data struct but tempest Client should always attach it for conveniency.
	GuildID GuildID `json:"-"`
	Deaf    bool    `json:"deaf,omitzero"`
	Mute    bool    `json:"mute,omitzero"`
	Pending bool    `json:"pending,omitzero"`
}

// Returns a direct url to members's guild specific avatar.
// It'll return empty string if targeted member don't use custom avatar for that server.
func (member *Member) GuildAvatarURL() string {
	return memberCDNURL(
		member.GuildID,
		member.User,
		"avatars",
		member.GuildAvatarHash,
		"member struct is missing guild ID which is required in avatar url method - it appears to be problem of your custom tempest client implementation",
	)
}

// Returns a direct url to members's guild specific banner.
// It'll return empty string if targeted member don't use custom banner for that server.
func (member *Member) GuildBannerURL() string {
	return memberCDNURL(
		member.GuildID,
		member.User,
		"banners",
		member.GuildBannerHash,
		"member struct is missing guild ID which is required in banner url method - it appears to be problem of your custom tempest client implementation",
	)
}

// https://docs.discord.com/developers/topics/permissions#role-object-role-flags
type RoleFlags = BitSet

const (
	InPromptRoleFlag RoleFlags = 1 << 0
)

// https://docs.discord.com/developers/topics/permissions#role-object
type Role struct {
	Colors          RoleColors      `json:"colors"`
	Tags            *RoleTag        `json:"tags,omitzero"`
	Name            string          `json:"name"`
	IconHash        string          `json:"icon,omitzero"`
	UnicodeEmoji    string          `json:"unicode_emoji,omitzero"`
	ID              RoleID          `json:"id"`
	PermissionFlags PermissionFlags `json:"permissions,string"`
	Flags           RoleFlags       `json:"flags,omitzero"`
	Hoist           bool            `json:"hoist,omitzero"`
	Position        uint8           `json:"position"`
	Managed         bool            `json:"managed,omitzero"`
	Mentionable     bool            `json:"mentionable,omitzero"`
}

func (role *Role) Mention() string {
	return role.ID.Mention()
}

// Returns a direct url to role icon. It'll return empty string if there's no custom icon.
func (role *Role) IconURL() string {
	return cdnAssetURL("role-icons", role.ID.Snowflake, role.IconHash, ".png", true)
}

// https://docs.discord.com/developers/topics/permissions#role-object-role-tags-structure
type RoleTag struct {
	BotID                 UserID        `json:"bot_id,omitzero"`
	IntegrationID         IntegrationID `json:"integration_id,omitzero"`          // The id of the integration this role belongs to.
	SubscriptionListingID SkuID         `json:"subscription_listing_id,omitzero"` // The id of this role's subscription sku and listing.
	PremiumSubscriber     bool          `json:"premium_subscriber,omitzero"`      // Whether this is the guild's Booster role.
	AvailableForPurchase  bool          `json:"available_for_purchase,omitzero"`
	GuildConnections      bool          `json:"guild_connections,omitzero"` // Whether this role is a guild's linked role.
}

// https://docs.discord.com/developers/topics/permissions#role-object-role-colors-object
type RoleColors struct {
	SecondaryColor *uint32 `json:"secondary_color,omitzero"`
	TertiaryColor  *uint32 `json:"tertiary_color,omitzero"`
	PrimaryColor   uint32  `json:"primary_color"`
}
