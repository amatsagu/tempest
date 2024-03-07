package tempest

import (
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

// https://discord.com/developers/docs/resources/user#user-object-user-structure
type User struct {
	ID          Snowflake `json:"id"`
	Username    string    `json:"username"`
	GlobalName  string    `json:"global_name,omitempty"` // User's display name, if it is set. For bots, this is the application name.
	AvatarHash  string    `json:"avatar,omitempty"`      // Hash code used to access user's profile. Call User.AvatarURL to get direct url.
	Bot         bool      `json:"bot,omitempty"`
	MFA         bool      `json:"mfa_enabled,omitempty"`
	BannerHash  string    `json:"banner,omitempty"`       // Hash code used to access user's baner. Call User.BannerURL to get direct url.
	AccentColor uint32    `json:"accent_color,omitempty"` // User's banner color, encoded as an integer representation of hexadecimal color code.
	Locale      string    `json:"locale,omitempty"`
	PremiumType NitroType `json:"premium_type,omitempty"`
	PublicFlags uint64    `json:"public_flags,omitempty"` // (Same as regular flags)
}

func (user User) Mention() string {
	return "<@" + user.ID.String() + ">"
}

// Returns a direct url to user's avatar. It'll return url to default Discord's avatar if targeted user don't use avatar.
func (user User) AvatarURL() string {
	if user.AvatarHash == "" {
		return DiscordCDNURL + "/embed/avatars/" + strconv.FormatUint(uint64(user.ID>>22)%6, 10) + ".png"
	}

	if strings.HasPrefix(user.AvatarHash, "a_") {
		return DiscordCDNURL + "/avatars/" + user.ID.String() + "/" + user.AvatarHash + ".gif"
	}

	return DiscordCDNURL + "/avatars/" + user.ID.String() + "/" + user.AvatarHash
}

// Returns a direct url to user's banner. It'll return empty string if targeted user don't use avatar.
func (user User) BannerURL() string {
	if user.BannerHash == "" {
		return ""
	}

	if strings.HasPrefix(user.BannerHash, "a_") {
		return DiscordCDNURL + "/banners/" + user.ID.String() + "/" + user.BannerHash + ".gif"
	}

	return DiscordCDNURL + "/banners/" + user.ID.String() + "/" + user.BannerHash
}

// https://discord.com/developers/docs/resources/guild#guild-member-object-guild-member-structure
type Member struct {
	User                       *User       `json:"user,omitempty"`
	Nickname                   string      `json:"nick,omitempty"`
	GuildAvatarHash            string      `json:"avatar,omitempty"` // Hash code used to access member's custom, guild profile. Call Member.GuildAvatarURL to get direct url.
	RoleIDs                    []Snowflake `json:"roles"`
	JoinedAt                   *time.Time  `json:"joined_at,omitempty"`
	PremiumSince               *time.Time  `json:"premium_since,omitempty"`
	Dead                       bool        `json:"deaf"`
	Mute                       bool        `json:"mute"`
	Flags                      uint64      `json:"flags"`
	Pending                    bool        `json:"pending,omitempty"`
	PermissionFlags            uint64      `json:"permissions,string"`
	CommunicationDisabledUntil *time.Time  `json:"communication_disabled_until,omitempty"`
	GuildID                    Snowflake   `json:"-"`
}

// Returns a direct url to members's guild specific avatar. It'll return empty string if targeted member don't use custom avatar for that server.
func (member Member) GuildAvatarURL() string {
	if member.GuildAvatarHash == "" {
		return ""
	}

	if strings.HasPrefix(member.GuildAvatarHash, "a_") {
		return DiscordCDNURL + "/guilds/" + member.GuildID.String() + "/users/" + member.User.ID.String() + "/avatars/" + member.GuildAvatarHash + ".gif"
	}

	return DiscordCDNURL + "/guilds/" + member.GuildID.String() + "/users/" + member.User.ID.String() + "/avatars/" + member.GuildAvatarHash
}

// https://discord.com/developers/docs/topics/permissions#role-object-role-structure
type Role struct {
	ID              Snowflake  `json:"id"`
	Name            string     `json:"name"`
	Color           uint32     `json:"color"` // Integer representation of hexadecimal color code. Roles without colors (color == 0) do not count towards the final computed color in the user list.
	Hoist           bool       `json:"hoist"` // Whether this role is pinned in the user listing.
	IconHash        string     `json:"icon,omitempty"`
	UnicodeEmoji    string     `json:"unicode_emoji,omitempty"`
	Position        uint8      `json:"position"`
	PermissionFlags uint64     `json:"permissions,string"`
	Managed         bool       `json:"managed"`     // Whether this role is managed by an integration.
	Mentionable     bool       `json:"mentionable"` // Whether this role is mentionable.
	Tags            []*RoleTag `json:"tags,omitempty"`
}

func (role Role) Mention() string {
	return "<@&" + role.ID.String() + ">"
}

// https://discord.com/developers/docs/topics/permissions#role-object-role-tags-structure
type RoleTag struct {
	BotID         Snowflake `json:"bot_id,omitempty"`
	IntegrationID Snowflake `json:"integration_id,omitempty"`
	// PremiumSubscriber bool <== UNKNOWN DOCUMENTATION
}

func (role Role) IconURL() string {
	if role.IconHash == "" {
		return ""
	}

	return DiscordCDNURL + "/role-icons/" + role.ID.String() + "/" + role.IconHash + ".png"
}
