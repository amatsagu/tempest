package discord

import (
	"strconv"
	"strings"
	"time"
)

const CDNBaseURL = "https://cdn.discordapp.com"

// https://docs.discord.com/developers/resources/user#user-object-user-flags
type UserFlags = BitSet

const (
	UserFlagStaff                 UserFlags = 1 << 0
	UserFlagPartner               UserFlags = 1 << 1
	UserFlagHypesquad             UserFlags = 1 << 2
	UserFlagBugHunterLevel1       UserFlags = 1 << 3
	UserFlagHypesquadOnlineHouse1 UserFlags = 1 << 6
	UserFlagHypesquadOnlineHouse2 UserFlags = 1 << 7
	UserFlagHypesquadOnlineHouse3 UserFlags = 1 << 8
	UserFlagPremiumEarlySupporter UserFlags = 1 << 9
	UserFlagTeamPseudoUser        UserFlags = 1 << 10
	UserFlagBugHunterLevel2       UserFlags = 1 << 14
	UserFlagVerifiedBot           UserFlags = 1 << 16
	UserFlagVerifiedDeveloper     UserFlags = 1 << 17
	UserFlagCertifiedModerator    UserFlags = 1 << 18
	UserFlagBotHTTPInteractions   UserFlags = 1 << 19
	UserFlagActiveDeveloper       UserFlags = 1 << 22
)

// https://docs.discord.com/developers/resources/user#user-object-premium-types
type NitroType uint8

const (
	NitroNone NitroType = iota
	NitroClassic
	NitroFull
	NitroBasic
)

// https://docs.discord.com/developers/resources/user#avatar-decoration-data-object
type AvatarDecoration struct {
	Asset string `json:"asset"`
	SkuID SkuID  `json:"sku_id"`
}

// https://docs.discord.com/developers/resources/user#nameplate-nameplate-structure
type Nameplate struct {
	ExpiresAt *time.Time `json:"expires_at,omitzero"`
	Asset     string     `json:"asset"`
	Label     string     `json:"label"`
	Palette   string     `json:"palette"`
	SkuID     SkuID      `json:"sku_id"`
}

// https://docs.discord.com/developers/resources/user#collectibles-collectible-structure
type Collectibles struct {
	Nameplate *Nameplate `json:"nameplate,omitzero"`
}

// https://docs.discord.com/developers/resources/user#user-object-user-primary-guild
type UserPrimaryGuild struct {
	Tag             string  `json:"tag,omitzero"`
	Badge           string  `json:"badge,omitzero"`
	IdentityGuildID GuildID `json:"identity_guild_id,omitzero"`
	IdentityEnabled bool    `json:"identity_enabled,omitzero"`
}

// https://docs.discord.com/developers/resources/user#user-object
type User struct {
	AvatarDecorationData *AvatarDecoration `json:"avatar_decoration_data,omitzero"`
	Collectibles         *Collectibles     `json:"collectibles,omitzero"`
	PrimaryGuild         *UserPrimaryGuild `json:"primary_guild,omitzero"`
	Username             string            `json:"username"`
	Discriminator        string            `json:"discriminator,omitzero"`
	GlobalName           string            `json:"global_name,omitzero"`
	Avatar               string            `json:"avatar,omitzero"`
	Banner               string            `json:"banner,omitzero"`
	Locale               Locale            `json:"locale,omitzero"`
	Email                string            `json:"email,omitzero"`
	ID                   UserID            `json:"id"`
	Flags                UserFlags         `json:"flags,omitzero"`
	PublicFlags          UserFlags         `json:"public_flags,omitzero"`
	AccentColor          uint32            `json:"accent_color,omitzero"`
	PremiumType          NitroType         `json:"premium_type,omitzero"`
	Bot                  bool              `json:"bot,omitzero"`
	System               bool              `json:"system,omitzero"`
	MFAEnabled           bool              `json:"mfa_enabled,omitzero"`
	Verified             bool              `json:"verified,omitzero"`
}

func (u User) EffectiveName() string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

func (u User) AvatarURL() string {
	if u.Avatar == "" {
		index := (u.ID.Snowflake >> 22) % 6
		return CDNBaseURL + "/embed/avatars/" + strconv.FormatUint(uint64(index), 10) + ".png"
	}
	if strings.HasPrefix(u.Avatar, "a_") {
		return CDNBaseURL + "/avatars/" + u.ID.String() + "/" + u.Avatar + ".webp?animated=true"
	}
	return CDNBaseURL + "/avatars/" + u.ID.String() + "/" + u.Avatar + ".png"
}

func (u User) BannerURL() string {
	if u.Banner == "" {
		return ""
	}
	if strings.HasPrefix(u.Banner, "a_") {
		return CDNBaseURL + "/banners/" + u.ID.String() + "/" + u.Banner + ".webp?animated=true"
	}
	return CDNBaseURL + "/banners/" + u.ID.String() + "/" + u.Banner + ".png"
}
