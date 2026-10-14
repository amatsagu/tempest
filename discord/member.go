package discord

import (
	"strings"
	"time"
)

// https://docs.discord.com/developers/resources/guild#guild-member-object-guild-member-flags
type MemberFlags = BitSet

const (
	MemberFlagDidRejoin                    MemberFlags = 1 << 0
	MemberFlagCompletedOnboarding          MemberFlags = 1 << 1
	MemberFlagBypassesVerification         MemberFlags = 1 << 2
	MemberFlagStartedOnboarding            MemberFlags = 1 << 3
	MemberFlagIsGuest                      MemberFlags = 1 << 4
	MemberFlagStartedHomeActions           MemberFlags = 1 << 5
	MemberFlagCompletedHomeActions         MemberFlags = 1 << 6
	MemberFlagAutomodQuarantinedUsername   MemberFlags = 1 << 7
	MemberFlagDmSettingsUpsellAcknowledged MemberFlags = 1 << 9
)

// https://docs.discord.com/developers/resources/guild#guild-member-object
type Member struct {
	JoinedAt                   time.Time         `json:"joined_at"`
	PremiumSince               *time.Time        `json:"premium_since,omitzero"`
	CommunicationDisabledUntil *time.Time        `json:"communication_disabled_until,omitzero"`
	User                       *User             `json:"user,omitzero"`
	AvatarDecorationData       *AvatarDecoration `json:"avatar_decoration_data,omitzero"`
	Nick                       string            `json:"nick,omitzero"`
	Avatar                     string            `json:"avatar,omitzero"`
	Banner                     string            `json:"banner,omitzero"`
	Roles                      []RoleID          `json:"roles"`
	Permissions                Permissions       `json:"permissions,string,omitzero"`
	Flags                      MemberFlags       `json:"flags,omitzero"`
	Deaf                       bool              `json:"deaf,omitzero"`
	Mute                       bool              `json:"mute,omitzero"`
	Pending                    bool              `json:"pending,omitzero"`
}

func (m Member) DisplayName() string {
	if m.Nick != "" {
		return m.Nick
	}
	if m.User != nil {
		return m.User.EffectiveName()
	}
	return ""
}

func (m Member) GuildAvatarURL(guildID GuildID) string {
	if m.Avatar == "" || m.User == nil || guildID.Snowflake == 0 {
		return ""
	}
	if strings.HasPrefix(m.Avatar, "a_") {
		return CDNBaseURL + "/guilds/" + guildID.String() + "/users/" + m.User.ID.String() + "/avatars/" + m.Avatar + ".webp?animated=true"
	}
	return CDNBaseURL + "/guilds/" + guildID.String() + "/users/" + m.User.ID.String() + "/avatars/" + m.Avatar + ".png"
}

func (m Member) GuildBannerURL(guildID GuildID) string {
	if m.Banner == "" || m.User == nil || guildID.Snowflake == 0 {
		return ""
	}
	if strings.HasPrefix(m.Banner, "a_") {
		return CDNBaseURL + "/guilds/" + guildID.String() + "/users/" + m.User.ID.String() + "/banners/" + m.Banner + ".webp?animated=true"
	}
	return CDNBaseURL + "/guilds/" + guildID.String() + "/users/" + m.User.ID.String() + "/banners/" + m.Banner + ".png"
}
