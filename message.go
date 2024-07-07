package tempest

import (
	"strconv"
	"time"
)

// https://discord.com/developers/docs/resources/channel#channel-object-channel-types
type ChannelType uint8

const (
	GUILD_TEXT_CHANNEL_TYPE ChannelType = iota
	DM_CHANNEL_TYPE
	GUILD_VOICE_CHANNEL_TYPE
	GROUP_DM_CHANNEL_TYPE
	GUILD_CATEGORY_CHANNEL_TYPE
	GUILD_ANNOUNCEMENT_CHANNEL_TYPE // Formerly news channel.
	_
	_
	_
	_
	GUILD_ANNOUNCEMENT_THREAD_CHANNEL_TYPE
	GUILD_PUBLIC_THREAD_CHANNEL_TYPE
	GUILD_PRIVATE_THREAD_CHANNEL_TYPE
	GUILD_STAGE_VOICE_CHANNEL_TYPE
	GUILD_DIRECTORY_CHANNEL_TYPE
	GUILD_FORUM_CHANNEL_TYPE
)

func (ct ChannelType) MarshalJSON() (p []byte, err error) {
	buf := strconv.FormatUint(uint64(ct), 10)
	return []byte(buf), nil
}

// https://discord.com/developers/docs/resources/sticker#sticker-object-sticker-format-types
type StickerFormatType uint8

const (
	PNG_STICKER_FORMAT_TYPE StickerFormatType = iota + 1
	APNG_STICKER_FORMAT_TYPE
	LOTTIE_STICKER_FORMAT_TYPE
	GIF_STICKER_FORMAT_TYPE
)

// https://discord.com/developers/docs/resources/channel#allowed-mentions-object-allowed-mentions-structure
type AllowedMentions struct {
	Parse       []string    `json:"parse,omitempty"`
	Roles       []Snowflake `json:"roles,omitempty"`
	Users       []Snowflake `json:"users,omitempty"`
	RepliedUser bool        `json:"replied_user,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#channel-object
type PartialChannel struct {
	ID              Snowflake   `json:"id"`
	Name            string      `json:"name"`
	PermissionFlags uint64      `json:"permissions,string"`
	Type            ChannelType `json:"type"`
}

// https://discord.com/developers/docs/resources/channel#channel-mention-object-channel-mention-structure
type ChannelMention struct {
	ID      Snowflake   `json:"id"`
	Name    string      `json:"name"`
	GuildID Snowflake   `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// https://discord.com/developers/docs/resources/emoji#emoji-object-emoji-structure
type PartialEmoji struct {
	ID       Snowflake `json:"id,omitempty"`
	Name     string    `json:"name"`
	Animated bool      `json:"animated,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#reaction-object-reaction-structure
type Reaction struct {
	Count uint          `json:"count"`
	Me    bool          `json:"me"`
	Emoji *PartialEmoji `json:"emoji"`
}

// https://discord.com/developers/docs/resources/sticker#sticker-item-object-sticker-item-structure
type StickerItem struct {
	ID         Snowflake         `json:"id"`
	Name       string            `json:"name"`
	FormatType StickerFormatType `json:"format_type"`
}

// https://discord.com/developers/docs/resources/emoji#emoji-object-emoji-structure
type Emoji struct {
	ID            Snowflake   `json:"id,omitempty"`
	Name          string      `json:"name"`
	Roles         []Snowflake `json:"roles,omitempty"`
	User          *User       `json:"user,omitempty"`
	RequireColons bool        `json:"require_colons,omitempty"`
	Managed       bool        `json:"managed,omitempty"`
	Animated      bool        `json:"animated,omitempty"`
	Available     bool        `json:"available,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-structure (always rich embed type)
type Embed struct {
	Title       string          `json:"title,omitempty"`
	URL         string          `json:"url,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Color       uint32          `json:"color,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Description string          `json:"description,omitempty"`
	Fields      []*EmbedField   `json:"fields,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Timestamp   *time.Time      `json:"timestamp,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-author-structure
type EmbedAuthor struct {
	IconURL string `json:"icon_url,omitempty"`
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-thumbnail-structure
type EmbedThumbnail struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    uint   `json:"width,omitempty"`
	Height   uint   `json:"height,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-field-structure
type EmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-footer-structure
type EmbedFooter struct {
	IconURL string `json:"icon_url,omitempty"`
	Text    string `json:"text,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-image-structure
type EmbedImage struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    uint   `json:"width,omitempty"`
	Height   uint   `json:"height,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-video-structure
type EmbedVideo struct {
	URL    string `json:"url,omitempty"`
	Width  uint   `json:"width,omitempty"`
	Height uint   `json:"height,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-provider-structure
type EmbedProvider struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#message-object-message-structure
type Message struct {
	ID                Snowflake           `json:"id"`
	ChannelID         Snowflake           `json:"channel_id"`
	Author            *User               `json:"author,omitempty"`
	Content           string              `json:"content,omitempty"`
	Timestamp         *time.Time          `json:"timestamp"`
	EditedTimestamp   *time.Time          `json:"edited_timestamp,omitempty"`
	TTS               bool                `json:"tts"`
	MentionEveryone   bool                `json:"mention_everyone"`
	Mentions          []*User             `json:"mentions"`
	MentionRoles      []*Snowflake        `json:"mention_roles"`
	MentionChannels   []*ChannelMention   `json:"mention_channels,omitempty"`
	Embeds            []*Embed            `json:"embeds"`
	Reactions         []*Reaction         `json:"reactions,omitempty"`
	Pinned            bool                `json:"pinned"`
	WebhookID         Snowflake           `json:"webhook_id,omitempty"`
	Type              uint                `json:"type,omitempty"` // https://discord.com/developers/docs/resources/channel#message-object-message-types
	ApplicationID     Snowflake           `json:"application_id,omitempty"`
	MessageReference  *MessageReference   `json:"message_reference,omitempty"`
	Flags             uint64              `json:"flags,omitempty"`
	ReferencedMessage *Message            `json:"referenced_message,omitempty"`
	Interaction       *MessageInteraction `json:"interaction,omitempty"`
	Components        []*ComponentRow     `json:"components,omitempty"`
	StickerItems      []*StickerItem      `json:"sticker_items,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#message-reference-object-message-reference-structure
type MessageReference struct {
	MessageID       Snowflake `json:"message_id,omitempty"`
	ChannelID       Snowflake `json:"channel_id,omitempty"`
	GuildID         Snowflake `json:"guild_id,omitempty"`
	FailIfNotExists bool      `json:"fail_if_not_exists,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#message-interaction-object-message-interaction-structure
type MessageInteraction struct {
	ID     Snowflake       `json:"id"`
	Type   InteractionType `json:"type"`
	Name   string          `json:"name"`
	User   User            `json:"user"`
	Member *Member         `json:"member,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#attachment-object
type Attachment struct {
	ID           Snowflake `json:"id"`
	FileName     string    `json:"filename"`
	Title        string    `json:"title,omitempty"`
	Description  string    `json:"description,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	Size         uint64    `json:"size"`
	URL          string    `json:"url"`
	ProxyURL     string    `json:"proxy_url"`
	Height       uint      `json:"height,omitempty"`
	Width        uint      `json:"width,omitempty"`
	Ephemeral    bool      `json:"ephemeral,omitempty"`
	DurationSecs float64   `json:"duration_secs,omitempty"`
	Waveform     string    `json:"waveform,omitempty"`
	Flags        uint64    `json:"flags,omitempty"`
}
