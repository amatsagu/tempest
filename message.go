package tempest

import (
	"strconv"
	"time"
)

// https://discord.com/developers/docs/resources/message#message-object-message-flags
type MessageFlags BitSet

const (
	CROSSPOSTED_MESSAGE_FLAG = 1 << iota
	IS_CROSSPOST_MESSAGE_FLAG
	SUPPRESS_EMBEDS_MESSAGE_FLAG
	SOURCE_MESSAGE_DELETED_MESSAGE_FLAG
	URGENT_MESSAGE_FLAG
	HAS_THREAD_MESSAGE_FLAG
	EPHEMERAL_MESSAGE_FLAG
	LOADING_MESSAGE_FLAG
	FAILED_TO_MENTION_SOME_ROLES_IN_THREAD_MESSAGE_FLAG
	_
	_
	_
	SUPPRESS_NOTIFICATIONS_MESSAGE_FLAG
	IS_VOICE_MESSAGE_MESSAGE_FLAG
	HAS_SNAPSHOT_MESSAGE_FLAG
	IS_COMPONENTS_V2_MESSAGE_FLAG // When used, regular content, embeds, poll & stickers fields will be ignored.
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
	GUILD_MEDIA_CHANNEL_TYPE
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

// https://discord.com/developers/docs/resources/message#allowed-mentions-object-allowed-mention-types
type AllowedMentionsType string

const (
	ALLOWED_ROLE_MENTION_TYPE     AllowedMentionsType = "roles"
	ALLOWED_USERS_MENTION_TYPE    AllowedMentionsType = "users"
	ALLOWED_EVERYONE_MENTION_TYPE AllowedMentionsType = "everyone"
)

// https://discord.com/developers/docs/resources/poll#layout-type
type PoolLayoutType uint8

const (
	DEFAULT_POOL_LAYOUT_TYPE PoolLayoutType = iota + 1
)

// https://discord.com/developers/docs/resources/message#allowed-mentions-object
type AllowedMentions struct {
	Parse       []AllowedMentionsType `json:"parse,omitzero"`
	Roles       []Snowflake           `json:"roles,omitzero"`
	Users       []Snowflake           `json:"users,omitzero"`
	RepliedUser bool                  `json:"replied_user,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#channel-object
type PartialChannel struct {
	ID              Snowflake       `json:"id"`
	Name            string          `json:"name"`
	PermissionFlags PermissionFlags `json:"permissions,string"`
	Type            ChannelType     `json:"type"`
}

// https://discord.com/developers/docs/resources/message#channel-mention-object
type ChannelMention struct {
	ID      Snowflake   `json:"id"`
	Name    string      `json:"name"`
	GuildID Snowflake   `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// https://discord.com/developers/docs/resources/message#reaction-count-details-object
type ReactionCountDetails struct {
	Burst  uint32 `json:"burst"`
	Normal uint32 `json:"normal"`
}

// https://discord.com/developers/docs/resources/message#reaction-object
type Reaction struct {
	Count        uint32               `json:"count"`
	CountDetails ReactionCountDetails `json:"count_details"`
	Me           bool                 `json:"me"`
	MeBurst      bool                 `json:"me_burst"`
	Emoji        Emoji                `json:"emoji"`
	BurstColors  []string             `json:"burst_colors"` // HEX colors used for super reaction
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
	Name          string      `json:"name,omitempty"` // Note: may be empty for deleted emojis.
	Roles         []Snowflake `json:"roles,omitzero"`
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
	Fields      []EmbedField    `json:"fields,omitzero"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Timestamp   *time.Time      `json:"timestamp,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-author-structure
type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-thumbnail-structure
type EmbedThumbnail struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    uint32 `json:"width,omitempty"`
	Height   uint32 `json:"height,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-field-structure
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-footer-structure
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-image-structure
type EmbedImage struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    uint32 `json:"width,omitempty"`
	Height   uint32 `json:"height,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-video-structure
type EmbedVideo struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    uint32 `json:"width,omitempty"`
	Height   uint32 `json:"height,omitempty"`
}

// https://discord.com/developers/docs/resources/channel#embed-object-embed-provider-structure
type EmbedProvider struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

// https://discord.com/developers/docs/resources/message#message-object-message-structure
type Message struct {
	ID                Snowflake           `json:"id"`
	ChannelID         Snowflake           `json:"channel_id"`
	Author            *User               `json:"author,omitempty"`
	Content           string              `json:"content,omitempty"`
	Timestamp         *time.Time          `json:"timestamp"`
	EditedTimestamp   *time.Time          `json:"edited_timestamp,omitempty"`
	TTS               bool                `json:"tts"`
	MentionEveryone   bool                `json:"mention_everyone"`
	Mentions          []User              `json:"mentions"`
	MentionRoles      []Snowflake         `json:"mention_roles"`
	MentionChannels   []ChannelMention    `json:"mention_channels,omitzero"`
	Attachments       []Attachment        `json:"attachments"`
	Embeds            []Embed             `json:"embeds"`
	Reactions         []Reaction          `json:"reactions,omitzero"`
	Pinned            bool                `json:"pinned"`
	WebhookID         Snowflake           `json:"webhook_id,omitempty"`
	Type              BitSet              `json:"type,omitempty"` // https://discord.com/developers/docs/resources/channel#message-object-message-types
	ApplicationID     Snowflake           `json:"application_id,omitempty"`
	MessageReference  *MessageReference   `json:"message_reference,omitempty"`
	Flags             uint64              `json:"flags,omitempty"`
	ReferencedMessage *Message            `json:"referenced_message,omitempty"`
	Interaction       *MessageInteraction `json:"interaction,omitempty"`
	Components        []LayoutComponent   `json:"components,omitzero"`
	StickerItems      []StickerItem       `json:"sticker_items,omitzero"`
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
	Height       uint32    `json:"height,omitempty"`
	Width        uint32    `json:"width,omitempty"`
	Ephemeral    bool      `json:"ephemeral,omitempty"`
	DurationSecs float64   `json:"duration_secs,omitempty"`
	Waveform     string    `json:"waveform,omitempty"`
	Flags        uint64    `json:"flags,omitempty"`
}

// https://discord.com/developers/docs/resources/poll#poll-create-request-object
type Poll struct {
	Question    PollMedia      `json:"question"`
	Answers     []PollAnswer   `json:"answers,omitzero"`
	Duration    uint16         `json:"duration,omitempty"` // Number of hours the poll should be open for, up to 32 days (defaults to 24)
	Multiselect bool           `json:"allow_multiselect,omitempty"`
	LayoutType  PoolLayoutType `json:"layout_type"`
}

// https://discord.com/developers/docs/resources/poll#poll-media-object-poll-media-object-structure
type PollMedia struct {
	Text  string `json:"text,omitempty"`
	Emoji *Emoji `json:"emoji,omitempty"`
}

// https://discord.com/developers/docs/resources/poll#poll-answer-object-poll-answer-object-structure
type PollAnswer struct {
	AnswerID  uint32    `json:"answer_id,omitempty"`
	PollMedia PollMedia `json:"poll_media"`
}

// https://discord.com/developers/docs/resources/poll#poll-results-object-poll-results-object-structure
type PoolResult struct {
	Finalized    bool               `json:"is_finalized"`
	AnswerCounts []PoolAnswerCounts `json:"answer_counts"`
}

// https://discord.com/developers/docs/resources/poll#poll-results-object-poll-answer-count-object-structure
type PoolAnswerCounts struct {
	ID      uint32 `json:"id"` // The answer_id
	Count   uint32 `json:"count"`
	MeVoted bool   `json:"me_voted"`
}
