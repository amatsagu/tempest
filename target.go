package tempest

import (
	"strconv"
	"strings"
)

type User struct {
	Id            Snowflake `json:"id"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	IsBot         bool      `json:"bot,omitempty"`
	AvatarHash    string    `json:"avatar,omitempty"` // Hash code used to access user's profile. Call User.FetchAvatarUrl to get direct url.
	BannerHash    string    `json:"banner,omitempty"` // Hash code used to access user's baner. Call User.FetchBannerUrl to get direct url.
	PublicFlags   uint64    `json:"public_flags,omitempty"`
	AccentColor   uint32    `json:"accent_color,omitempty"` // User's banner color, encoded as an integer representation of hexadecimal color code.
	PremiumType   uint8     `json:"premium_type,omitempty"`
}

func (user User) Tag() string {
	return user.Username + "#" + user.Discriminator
}

func (user User) Mention() string {
	return "<@" + user.Username + ">"
}

// Returns a direct url to user's avatar. It'll return url to default Discord's avatar if targeted user don't use avatar.
func (user User) FetchAvatarUrl() string {
	if user.AvatarHash == "" {
		n, err := strconv.Atoi(user.Discriminator)
		if err != nil {
			return DISCORD_CDN_URL + "/embed/avatars/0.png"
		}

		return DISCORD_CDN_URL + "/embed/avatars/" + strconv.Itoa(n%5) + ".png"
	}

	if strings.HasPrefix(user.AvatarHash, "a_") {
		return DISCORD_CDN_URL + "/avatars/" + user.Id.String() + "/" + user.AvatarHash + ".gif"
	}

	return DISCORD_CDN_URL + "/avatars/" + user.Id.String() + "/" + user.AvatarHash
}

// Returns a direct url to user's banner. It'll return empty string if targeted user don't use avatar.
func (user User) FetchBannerUrl() string {
	if user.BannerHash == "" {
		return ""
	}

	if strings.HasPrefix(user.AvatarHash, "a_") {
		return DISCORD_CDN_URL + "/banners/" + user.Id.String() + "/" + user.BannerHash + ".gif"
	}

	return DISCORD_CDN_URL + "/banners/" + user.Id.String() + "/" + user.BannerHash
}

type Member struct {
	User            *User       `json:"user,omitempty"` // Struct with general user data. In theory it may be empty but I never seen such payload.
	GuildId         Snowflake   `json:"-"`
	GuildAvatarHash string      `json:"avatar,omitempty"` // Hash code used to access member's custom, guild profile. Call Member.FetchGuildAvatarUrl to get direct url.
	Nickname        string      `json:"nick,omitempty"`
	JoinedAt        string      `json:"joined_at"`
	NitroSince      string      `json:"premium_since,omitempty"`
	RoleIds         []Snowflake `json:"roles"`
	PermissionFlags uint64      `json:"permissions,string"`
}

// Returns a direct url to members's guild specific avatar. It'll return empty string if targeted member don't use custom avatar for that server.
func (member Member) FetchGuildAvatarUrl() string {
	if member.GuildAvatarHash == "" {
		return ""
	}

	if strings.HasPrefix(member.GuildAvatarHash, "a_") {
		return DISCORD_CDN_URL + "/guilds/" + member.GuildId.String() + "/users/" + member.User.Id.String() + "/avatars/" + member.GuildAvatarHash + ".gif"
	}

	return DISCORD_CDN_URL + "/guilds/" + member.GuildId.String() + "/users/" + member.User.Id.String() + "/avatars/" + member.GuildAvatarHash
}
