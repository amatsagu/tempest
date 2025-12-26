package tempest

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// https://discord.com/developers/docs/resources/user#user-object-premium-types
type NitroType uint8

const (
	NO_NITRO_TYPE NitroType = iota
	CLASSIC_NITRO_TYPE
	FULL_NITRO_TYPE
	BASIC_NITRO_TYPE
)

// https://discord.com/developers/docs/resources/user#avatar-decoration-data-object-avatar-decoration-data-structure
type AvatarDecoration struct {
	AssetHash string    `json:"avatar"` // Hash code used to access user's avatar decoration.
	SkuID     Snowflake `json:"sku_id"`
}

// Returns a direct url to targets's avatar decoration. It'll return empty string if target doesn't use avatar decoration.
func (adc *AvatarDecoration) DecorationURL() string {
	if adc.AssetHash == "" {
		return ""
	}

	if strings.HasPrefix(adc.AssetHash, "a_") {
		return DISCORD_CDN_URL + "/avatar-decoration-presets/" + adc.AssetHash + ".gif"
	}

	return DISCORD_CDN_URL + "/avatar-decoration-presets/" + adc.AssetHash
}

// https://discord.com/developers/docs/resources/user#user-object-user-flags
type UserFlags BitSet

const (
	DISCORD_EMPLOYEE_USER_FLAG       UserFlags = 1 << iota // Discord Employee, Staff
	PARTNERED_SERVER_OWNER_USER_FLAG                       // Partner
	HYPESQUAD_USER_FLAG
	BUG_HUNTER_LEVEL_1_USER_FLAG
	_
	_
	HYPESQUAD_ONLINE_HOUSE_1_USER_FLAG
	HYPESQUAD_ONLINE_HOUSE_2_USER_FLAG
	HYPESQUAD_ONLINE_HOUSE_3_USER_FLAG
	PREMIUM_EARLY_SUPPORTER_USER_FLAG
	TEAM_USER_USER_FLAG // Discord docs mentions "pseudo user" and that user is a team...
	_
	_
	_
	BUG_HUNTER_LEVEL_2_USER_FLAG
	_
	VERIFIED_BOT_USER_FLAG
	VERIFIED_DEVELOPER_USER_FLAG    // Early Verified Bot Developer
	CERTIFIED_MODERATOR_USER_FLAG   // Moderator Programs Alumni
	BOT_HTTP_INTERACTIONS_USER_FLAG // Bot/App uses only HTTP interactions and is shown in the online member list.
	_
	_
	ACTIVE_DEVELOPER_USER_FLAG // User has regular discord developer badge
)

// https://discord.com/developers/docs/resources/user#user-object-user-structure
type User struct {
	ID                   Snowflake         `json:"id"`
	Username             string            `json:"username"`
	GlobalName           string            `json:"global_name,omitempty"`  // User's display name. Tempest lib will make it equal to user.Username if it was empty.
	AvatarHash           string            `json:"avatar,omitempty"`       // Hash code used to access user's profile. Call User.AvatarURL to get direct url.
	Bot                  bool              `json:"bot"`                    // Whether it's bot/app account.
	System               bool              `json:"system"`                 // Whether user is Discord System Message account.
	BannerHash           string            `json:"banner,omitempty"`       // Hash code used to access user's baner. Call User.BannerURL to get direct url.
	AccentColor          uint32            `json:"accent_color,omitempty"` // User's banner color, encoded as an integer representation of hexadecimal color code.
	Locale               string            `json:"locale,omitempty"`
	PremiumType          NitroType         `json:"premium_type,omitempty"`
	PublicFlags          UserFlags         `json:"public_flags,omitempty"` // (Same as regular, user flags)
	AvatarDecorationData *AvatarDecoration `json:"avatar_decoration_data,omitempty"`
}

func (u *User) UnmarshalJSON(data []byte) error {
	// Define a local type to avoid recursion
	type Alias User
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(u),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if u.GlobalName == "" {
		u.GlobalName = u.Username
	}

	return nil
}

func (user *User) Mention() string {
	return "<@" + user.ID.String() + ">"
}

// Returns a direct url to user's avatar. It'll return url to default Discord's avatar if targeted user don't use avatar.
func (user *User) AvatarURL() string {
	if user.AvatarHash == "" {
		return DISCORD_CDN_URL + "/embed/avatars/" + strconv.FormatUint(uint64(user.ID>>22)%6, 10) + ".png"
	}

	if strings.HasPrefix(user.AvatarHash, "a_") {
		return DISCORD_CDN_URL + "/avatars/" + user.ID.String() + "/" + user.AvatarHash + ".gif"
	}

	return DISCORD_CDN_URL + "/avatars/" + user.ID.String() + "/" + user.AvatarHash
}

// Returns a direct url to user's banner. It'll return empty string if targeted user don't use avatar.
func (user *User) BannerURL() string {
	if user.BannerHash == "" {
		return ""
	}

	if strings.HasPrefix(user.BannerHash, "a_") {
		return DISCORD_CDN_URL + "/banners/" + user.ID.String() + "/" + user.BannerHash + ".gif"
	}

	return DISCORD_CDN_URL + "/banners/" + user.ID.String() + "/" + user.BannerHash
}

// https://discord.com/developers/docs/resources/guild#guild-member-object-guild-member-flags
type MemberFlags BitSet

const (
	DID_REJOIN_MEMBER_FLAG MemberFlags = 1 << iota
	COMPLETED_ONBOARDING_MEMBER_FLAG
	BYPASSES_VERIFICATION_MEMBER_FLAG
	STARTED_ONBOARDING_MEMBER_FLAG
	IS_GUEST_MEMBER_FLAG
	STARTED_HOME_ACTIONS_MEMBER_FLAG
	COMPLETED_HOME_ACTIONS_MEMBER_FLAG
	AUTOMOD_QUARANTINED_USERNAME_MEMBER_FLAG
	_
	DM_SETTINGS_UPSELL_ACKNOWLEDGED_MEMBER_FLAG
)

// https://discord.com/developers/docs/resources/guild#guild-member-object-guild-member-structure
type Member struct {
	User                       *User             `json:"user,omitempty"`
	Nickname                   string            `json:"nick,omitempty"`
	GuildAvatarHash            string            `json:"avatar,omitempty"` // Hash code used to access member's custom, guild avatar. Call Member.GuildAvatarURL to get direct url.
	GuildBannerHash            string            `json:"banner,omitempty"` // Hash code used to access member's custom, guild banner. Call Member.GuildBannerURL to get direct url.
	RoleIDs                    []Snowflake       `json:"roles"`
	JoinedAt                   *time.Time        `json:"joined_at"`
	PremiumSince               *time.Time        `json:"premium_since,omitempty"`
	Deaf                       bool              `json:"deaf"`
	Mute                       bool              `json:"mute"`
	Flags                      MemberFlags       `json:"flags"`
	Pending                    bool              `json:"pending"`
	PermissionFlags            PermissionFlags   `json:"permissions,string"`
	CommunicationDisabledUntil *time.Time        `json:"communication_disabled_until,omitempty"`
	AvatarDecorationData       *AvatarDecoration `json:"avatar_decoration_data,omitempty"`

	// It's not part of Member API data struct but tempest Client should always attach it for conveniency.
	GuildID Snowflake `json:"-"`
}

// Returns a direct url to members's guild specific avatar.
// It'll return empty string if targeted member don't use custom avatar for that server.
func (member *Member) GuildAvatarURL() string {
	if member.GuildAvatarHash == "" {
		return ""
	}

	if member.GuildID == 0 {
		panic("member struct is missing guild ID which is required in avatar url method - it appears to be problem of your custom tempest client implementation")
	}

	if strings.HasPrefix(member.GuildAvatarHash, "a_") {
		return DISCORD_CDN_URL + "/guilds/" + member.GuildID.String() + "/users/" + member.User.ID.String() + "/avatars/" + member.GuildAvatarHash + ".gif"
	}

	return DISCORD_CDN_URL + "/guilds/" + member.GuildID.String() + "/users/" + member.User.ID.String() + "/avatars/" + member.GuildAvatarHash
}

// Returns a direct url to members's guild specific banner.
// It'll return empty string if targeted member don't use custom banner for that server.
func (member *Member) GuildBannerURL() string {
	if member.GuildBannerHash == "" {
		return ""
	}

	if member.GuildID == 0 {
		panic("member struct is missing guild ID which is required in banner url method - it appears to be problem of your custom tempest client implementation")
	}

	if strings.HasPrefix(member.GuildBannerHash, "a_") {
		return DISCORD_CDN_URL + "/guilds/" + member.GuildID.String() + "/users/" + member.User.ID.String() + "/banners/" + member.GuildBannerHash + ".gif"
	}

	return DISCORD_CDN_URL + "/guilds/" + member.GuildID.String() + "/users/" + member.User.ID.String() + "/banners/" + member.GuildBannerHash
}

// https://discord.com/developers/docs/topics/permissions#role-object-role-structure
type Role struct {
	ID              Snowflake       `json:"id"`
	Name            string          `json:"name"`
	Color           uint32          `json:"color"` // Integer representation of hexadecimal color code. Roles without colors (color == 0) do not count towards the final computed color in the user list.
	Hoist           bool            `json:"hoist"` // Whether this role is pinned in the user listing.
	IconHash        string          `json:"icon,omitempty"`
	UnicodeEmoji    string          `json:"unicode_emoji,omitempty"`
	Position        uint8           `json:"position"`
	PermissionFlags PermissionFlags `json:"permissions,string"`
	Managed         bool            `json:"managed"`     // Whether this role is managed by an integration.
	Mentionable     bool            `json:"mentionable"` // Whether this role is mentionable.
	Tags            RoleTag         `json:"tags"`
	Flags           BitSet          `json:"flags"` // https://discord.com/developers/docs/topics/permissions#role-object-role-flags
}

func (role *Role) Mention() string {
	return "<@&" + role.ID.String() + ">"
}

// Returns a direct url to role icon. It'll return empty string if there's no custom icon.
func (role *Role) IconURL() string {
	if role.IconHash == "" {
		return ""
	}

	if strings.HasPrefix(role.IconHash, "a_") {
		return DISCORD_CDN_URL + "/avatars/" + role.ID.String() + "/" + role.IconHash + ".gif"
	}

	return DISCORD_CDN_URL + "/avatars/" + role.ID.String() + "/" + role.IconHash
}

// https://discord.com/developers/docs/topics/permissions#role-object-role-tags-structure
type RoleTag struct {
	BotID                 Snowflake `json:"bot_id,omitempty"`
	IntegrationID         Snowflake `json:"integration_id,omitempty"`          // The id of the integration this role belongs to.
	PremiumSubscriber     bool      `json:"premium_subscriber"`                // Whether this is the guild's Booster role.
	SubscriptionListingID Snowflake `json:"subscription_listing_id,omitempty"` // The id of this role's subscription sku and listing.
	AvailableForPurchase  bool      `json:"available_for_purchase"`
	GuildConnections      bool      `json:"guild_connections"` // Whether this role is a guild's linked role.
}
