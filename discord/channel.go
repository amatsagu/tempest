package discord

import "time"

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
	// ChannelFlagObfuscated marks a channel whose metadata Discord hid because the
	// bot cannot view it. It only ever arrives over the gateway.
	ChannelFlagObfuscated ChannelFlags = 1 << 17
	// ChannelFlagIsSpoilerChannel marks a channel whose contents users must opt in
	// to see.
	ChannelFlagIsSpoilerChannel ChannelFlags = 1 << 21
)

// https://docs.discord.com/developers/resources/channel#channel-object-video-quality-modes
type VideoQualityMode uint8

const (
	VideoQualityAuto VideoQualityMode = 1
	VideoQualityFull VideoQualityMode = 2
)

// https://docs.discord.com/developers/resources/channel#overwrite-object-overwrite-structure
type PermissionOverwriteType uint8

const (
	OverwriteRole   PermissionOverwriteType = 0
	OverwriteMember PermissionOverwriteType = 1
)

// https://docs.discord.com/developers/resources/channel#overwrite-object-overwrite-structure
type PermissionOverwrite struct {
	ID    Snowflake               `json:"id"`
	Type  PermissionOverwriteType `json:"type"`
	Allow Permissions             `json:"allow,string,omitzero"`
	Deny  Permissions             `json:"deny,string,omitzero"`
}

// https://docs.discord.com/developers/resources/channel#channel-object-sort-order-types
type ChannelSortOrder uint8

const (
	ChannelSortOrderLatestActivity ChannelSortOrder = 0
	ChannelSortOrderCreationDate   ChannelSortOrder = 1
)

// https://docs.discord.com/developers/resources/channel#channel-object-forum-layout-types
type ForumLayout uint8

const (
	ForumLayoutNotSet      ForumLayout = 0
	ForumLayoutListView    ForumLayout = 1
	ForumLayoutGalleryView ForumLayout = 2
)

// https://docs.discord.com/developers/resources/channel#forum-tag-object-forum-tag-structure
type ForumTag struct {
	Name      string    `json:"name"`
	EmojiName string    `json:"emoji_name,omitzero"`
	ID        Snowflake `json:"id"`
	EmojiID   Snowflake `json:"emoji_id,omitzero"`
	Moderated bool      `json:"moderated"`
}

// https://docs.discord.com/developers/resources/channel#thread-metadata-object-thread-metadata-structure
type ThreadMetadata struct {
	ArchiveTimestamp    time.Time  `json:"archive_timestamp"`
	CreateTimestamp     *time.Time `json:"create_timestamp,omitzero"`
	AutoArchiveDuration int        `json:"auto_archive_duration"`
	Archived            bool       `json:"archived"`
	Locked              bool       `json:"locked"`
	Invitable           bool       `json:"invitable,omitzero"`
}

// https://docs.discord.com/developers/resources/channel#thread-member-object-thread-member-structure
type ThreadMember struct {
	JoinTimestamp time.Time `json:"join_timestamp"`
	Member        *Member   `json:"member,omitzero"`
	ID            Snowflake `json:"id,omitzero"`
	UserID        Snowflake `json:"user_id,omitzero"`
	Flags         int       `json:"flags"`
}

// https://docs.discord.com/developers/resources/channel#channel-object
type Channel struct {
	LastPinTimestamp              *time.Time            `json:"last_pin_timestamp,omitzero"`
	ThreadMetadata                *ThreadMetadata       `json:"thread_metadata,omitzero"`
	DefaultReactionEmoji          *Emoji                `json:"default_reaction_emoji,omitzero"`
	Member                        *ThreadMember         `json:"member,omitzero"`
	DefaultSortOrder              *ChannelSortOrder     `json:"default_sort_order,omitzero"`
	RtcRegion                     string                `json:"rtc_region,omitzero"`
	Icon                          string                `json:"icon,omitzero"`
	Name                          string                `json:"name,omitzero"`
	Topic                         string                `json:"topic,omitzero"`
	PermissionOverwrites          []PermissionOverwrite `json:"permission_overwrites,omitzero"`
	Recipients                    []User                `json:"recipients,omitzero"`
	AvailableTags                 []ForumTag            `json:"available_tags,omitzero"`
	AppliedTags                   []Snowflake           `json:"applied_tags,omitzero"`
	OwnerID                       UserID                `json:"owner_id,omitzero"`
	AppPermissions                Permissions           `json:"app_permissions,string,omitzero"`
	GuildID                       GuildID               `json:"guild_id,omitzero"`
	ParentID                      ChannelID             `json:"parent_id,omitzero"`
	LastMessageID                 MessageID             `json:"last_message_id,omitzero"`
	ID                            ChannelID             `json:"id"`
	ApplicationID                 AppID                 `json:"application_id,omitzero"`
	Permissions                   Permissions           `json:"permissions,string,omitzero"`
	Position                      int                   `json:"position,omitzero"`
	Flags                         ChannelFlags          `json:"flags,omitzero"`
	Bitrate                       uint32                `json:"bitrate,omitzero"`
	UserLimit                     uint32                `json:"user_limit,omitzero"`
	RateLimitPerUser              uint32                `json:"rate_limit_per_user,omitzero"`
	MessageCount                  uint32                `json:"message_count,omitzero"`
	MemberCount                   uint32                `json:"member_count,omitzero"`
	DefaultAutoArchiveDuration    uint32                `json:"default_auto_archive_duration,omitzero"`
	DefaultThreadRateLimitPerUser uint32                `json:"default_thread_rate_limit_per_user,omitzero"`
	TotalMessageSent              uint32                `json:"total_message_sent,omitzero"`
	Type                          ChannelType           `json:"type"`
	VideoQualityMode              VideoQualityMode      `json:"video_quality_mode,omitzero"`
	DefaultForumLayout            ForumLayout           `json:"default_forum_layout,omitzero"`
	NSFW                          bool                  `json:"nsfw,omitzero"`
	Managed                       bool                  `json:"managed,omitzero"`
}
