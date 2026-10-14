package discord

// https://docs.discord.com/developers/resources/channel#channel-object-channel-types
type ChannelType uint8

const (
	ChannelGuildText ChannelType = iota
	ChannelDM
	ChannelGuildVoice
	ChannelGroupDM
	ChannelGuildCategory
	ChannelGuildAnnouncement
	_
	_
	_
	_
	ChannelGuildAnnouncementThread
	ChannelGuildPublicThread
	ChannelGuildPrivateThread
	ChannelGuildStageVoice
	ChannelGuildDirectory
	ChannelGuildForum
	ChannelGuildMedia
)

// https://docs.discord.com/developers/resources/channel#channel-object-channel-flags
type ChannelFlags = BitSet

const (
	ChannelFlagPinned                   ChannelFlags = 1 << 1
	ChannelFlagRequireTag               ChannelFlags = 1 << 4
	ChannelFlagHideMediaDownloadOptions ChannelFlags = 1 << 15
)

// https://docs.discord.com/developers/resources/channel#channel-object
type Channel struct {
	Name             string       `json:"name,omitzero"`
	Topic            string       `json:"topic,omitzero"`
	ID               ChannelID    `json:"id"`
	GuildID          GuildID      `json:"guild_id,omitzero"`
	ParentID         ChannelID    `json:"parent_id,omitzero"`
	LastMessageID    MessageID    `json:"last_message_id,omitzero"`
	Permissions      Permissions  `json:"permissions,string,omitzero"`
	Type             ChannelType  `json:"type"`
	Position         int          `json:"position,omitzero"`
	Bitrate          uint32       `json:"bitrate,omitzero"`
	UserLimit        uint32       `json:"user_limit,omitzero"`
	RateLimitPerUser uint32       `json:"rate_limit_per_user,omitzero"`
	Flags            ChannelFlags `json:"flags,omitzero"`
	NSFW             bool         `json:"nsfw,omitzero"`
}

func (c Channel) Mention() string {
	return c.ID.Mention()
}
