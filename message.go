package tempest

import (
	"strconv"
	"time"
)

// https://docs.discord.com/developers/resources/message#message-object-message-flags
type MessageFlags = BitSet

const (
	CROSSPOSTED_MESSAGE_FLAG MessageFlags = 1 << iota
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

// https://docs.discord.com/developers/resources/message#message-object-message-types
type MessageType uint8

const (
	DEFAULT_MESSAGE_TYPE MessageType = iota
	RECIPIENT_ADD_MESSAGE_TYPE
	RECIPIENT_REMOVE_MESSAGE_TYPE
	CALL_MESSAGE_TYPE
	CHANNEL_NAME_CHANGE_MESSAGE_TYPE
	CHANNEL_ICON_CHANGE_MESSAGE_TYPE
	CHANNEL_PINNED_MESSAGE_TYPE
	USER_JOIN_MESSAGE_TYPE
	GUILD_BOOST_MESSAGE_TYPE
	GUILD_BOOST_TIER_1_MESSAGE_TYPE
	GUILD_BOOST_TIER_2_MESSAGE_TYPE
	GUILD_BOOST_TIER_3_MESSAGE_TYPE
	CHANNEL_FOLLOW_ADD_MESSAGE_TYPE
	_
	GUILD_DISCOVERY_DISQUALIFIED_MESSAGE_TYPE
	GUILD_DISCOVERY_REQUALIFIED_MESSAGE_TYPE
	GUILD_DISCOVERY_GRACE_PERIOD_INITIAL_WARNING_MESSAGE_TYPE
	GUILD_DISCOVERY_GRACE_PERIOD_FINAL_WARNING_MESSAGE_TYPE
	THREAD_CREATED_MESSAGE_TYPE
	REPLY_MESSAGE_TYPE
	CHAT_INPUT_COMMAND_MESSAGE_TYPE
	THREAD_STARTER_MESSAGE_TYPE
	GUILD_INVITE_REMINDER_MESSAGE_TYPE
	CONTEXT_MENU_COMMAND_MESSAGE_TYPE
	AUTO_MODERATION_ACTION_MESSAGE_TYPE
	ROLE_SUBSCRIPTION_PURCHASE_MESSAGE_TYPE
	INTERACTION_PREMIUM_UPSELL_MESSAGE_TYPE
	STAGE_START_MESSAGE_TYPE
	STAGE_END_MESSAGE_TYPE
	STAGE_SPEAKER_MESSAGE_TYPE
	_
	STAGE_TOPIC_MESSAGE_TYPE
	GUILD_APPLICATION_PREMIUM_SUBSCRIPTION_MESSAGE_TYPE
	_
	_
	_
	GUILD_INCIDENT_ALERT_MODE_ENABLED_MESSAGE_TYPE
	GUILD_INCIDENT_ALERT_MODE_DISABLED_MESSAGE_TYPE
	GUILD_INCIDENT_REPORT_RAID_MESSAGE_TYPE
	GUILD_INCIDENT_REPORT_FALSE_ALARM_MESSAGE_TYPE
	_
	_
	_
	_
	PURCHASE_NOTIFICATION_MESSAGE_TYPE
	_
	POLL_RESULT_MESSAGE_TYPE
)

// https://docs.discord.com/developers/resources/channel#channel-object-channel-types
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

// https://docs.discord.com/developers/resources/sticker#sticker-object-sticker-format-types
type StickerFormatType uint8

const (
	PNG_STICKER_FORMAT_TYPE StickerFormatType = iota + 1
	APNG_STICKER_FORMAT_TYPE
	LOTTIE_STICKER_FORMAT_TYPE
	GIF_STICKER_FORMAT_TYPE
)

// https://docs.discord.com/developers/resources/message#allowed-mentions-object-allowed-mention-types
type AllowedMentionsType string

const (
	ALLOWED_ROLE_MENTION_TYPE     AllowedMentionsType = "roles"
	ALLOWED_USERS_MENTION_TYPE    AllowedMentionsType = "users"
	ALLOWED_EVERYONE_MENTION_TYPE AllowedMentionsType = "everyone"
)

// https://docs.discord.com/developers/resources/poll#layout-type
type PoolLayoutType uint8

const (
	DEFAULT_POOL_LAYOUT_TYPE PoolLayoutType = iota + 1
)

// https://docs.discord.com/developers/resources/message#allowed-mentions-object
type AllowedMentions struct {
	Parse       []AllowedMentionsType `json:"parse,omitzero"`
	Roles       []Snowflake           `json:"roles,omitzero"`
	Users       []Snowflake           `json:"users,omitzero"`
	RepliedUser bool                  `json:"replied_user"`
}

// https://docs.discord.com/developers/resources/channel#channel-object
type PartialChannel struct {
	Name            string          `json:"name"`
	ID              Snowflake       `json:"id"`
	PermissionFlags PermissionFlags `json:"permissions,string"`
	Type            ChannelType     `json:"type"`
}

// https://docs.discord.com/developers/resources/message#channel-mention-object-channel-mention-structure
type ChannelMention struct {
	Name    string      `json:"name"`
	ID      Snowflake   `json:"id"`
	GuildID Snowflake   `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// https://docs.discord.com/developers/resources/message#reaction-count-details-object-reaction-count-details-structure
type ReactionCountDetails struct {
	Burst  uint32 `json:"burst"`
	Normal uint32 `json:"normal"`
}

// https://docs.discord.com/developers/resources/message#reaction-object-reaction-structure
type Reaction struct {
	BurstColors  []string             `json:"burst_colors"` // HEX colors used for super reaction
	Emoji        Emoji                `json:"emoji"`
	CountDetails ReactionCountDetails `json:"count_details"`
	Count        uint32               `json:"count"`
	Me           bool                 `json:"me"`
	MeBurst      bool                 `json:"me_burst"`
}

// https://docs.discord.com/developers/resources/sticker#sticker-item-object-sticker-item-structure
type StickerItem struct {
	Name       string            `json:"name"`
	ID         Snowflake         `json:"id"`
	FormatType StickerFormatType `json:"format_type"`
}

// https://docs.discord.com/developers/resources/emoji#emoji-object
type Emoji struct {
	User          *User       `json:"user,omitempty"`
	Name          string      `json:"name,omitempty"` // Note: may be empty for deleted emojis.
	Roles         []Snowflake `json:"roles,omitzero"`
	ID            Snowflake   `json:"id,omitempty"`
	RequireColons bool        `json:"require_colons"`
	Managed       bool        `json:"managed"`
	Animated      bool        `json:"animated"`
	Available     bool        `json:"available"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-flags
type EmbedFlags = BitSet

const (
	CONTAINS_EXPLICIT_CONTENT_EMBED_FLAG  EmbedFlags = 1 << 4
	IS_CONTENT_INVENTORY_ENTRY_EMBED_FLAG EmbedFlags = 1 << 5
	CONTAINS_GORE_CONTENT_EMBED_FLAG      EmbedFlags = 1 << 6
	CONTAINS_SELF_HARM_CONTENT_EMBED_FLAG EmbedFlags = 1 << 7
)

// https://docs.discord.com/developers/resources/message#embed-object-embed-media-flags
type EmbedMediaFlags = BitSet

const (
	IS_ANIMATED_EMBED_MEDIA_FLAG EmbedMediaFlags = 1 << 5
)

// https://docs.discord.com/developers/resources/message#embed-object
type Embed struct {
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Timestamp   *time.Time      `json:"timestamp,omitempty"`
	Title       string          `json:"title,omitempty"`
	Type        string          `json:"type,omitempty"`
	URL         string          `json:"url,omitempty"`
	Description string          `json:"description,omitempty"`
	Fields      []EmbedField    `json:"fields,omitzero"`
	Color       uint32          `json:"color,omitempty"`
	Flags       EmbedFlags      `json:"flags,omitempty"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-author-structure
type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-image-structure
type EmbedThumbnail struct {
	URL                string          `json:"url"`
	ProxyURL           string          `json:"proxy_url,omitempty"`
	ContentType        string          `json:"content_type,omitempty"`
	Placeholder        string          `json:"placeholder,omitempty"`
	Description        string          `json:"description,omitempty"`
	Flags              EmbedMediaFlags `json:"flags,omitempty"`
	Width              uint32          `json:"width,omitempty"`
	Height             uint32          `json:"height,omitempty"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitempty"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-field-structure
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-footer-structure
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-image-structure
type EmbedImage struct {
	URL                string          `json:"url"`
	ProxyURL           string          `json:"proxy_url,omitempty"`
	ContentType        string          `json:"content_type,omitempty"`
	Placeholder        string          `json:"placeholder,omitempty"`
	Description        string          `json:"description,omitempty"`
	Flags              EmbedMediaFlags `json:"flags,omitempty"`
	Width              uint32          `json:"width,omitempty"`
	Height             uint32          `json:"height,omitempty"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitempty"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-video-structure
type EmbedVideo struct {
	URL                string          `json:"url,omitempty"`
	ProxyURL           string          `json:"proxy_url,omitempty"`
	ContentType        string          `json:"content_type,omitempty"`
	Placeholder        string          `json:"placeholder,omitempty"`
	Description        string          `json:"description,omitempty"`
	Flags              EmbedMediaFlags `json:"flags,omitempty"`
	Width              uint32          `json:"width,omitempty"`
	Height             uint32          `json:"height,omitempty"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitempty"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-provider-structure
type EmbedProvider struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

// https://docs.discord.com/developers/resources/message#message-object
type Message struct {
	MessageReference    *MessageReference           `json:"message_reference,omitempty"`
	InteractionMetadata *MessageInteractionMetadata `json:"interaction_metadata,omitempty"`
	Poll                *Poll                       `json:"poll,omitempty"`
	Member              *Member                     `json:"member,omitempty"`
	Resolved            *InteractionDataResolved    `json:"resolved,omitempty"`
	Author              *User                       `json:"author,omitempty"` // The author of this message.
	Timestamp           *time.Time                  `json:"timestamp"`
	EditedTimestamp     *time.Time                  `json:"edited_timestamp,omitempty"`
	ReferencedMessage   *Message                    `json:"referenced_message,omitempty"`
	Content             string                      `json:"content,omitempty"` // The message's content.
	Nonce               Nonce                       `json:"nonce,omitempty"`   // Used for validating a message was sent.
	Attachments         []Attachment                `json:"attachments"`
	Mentions            []User                      `json:"mentions"` // All users specifically mentioned in the message.
	MentionRoles        []Snowflake                 `json:"mention_roles"`
	MentionChannels     []ChannelMention            `json:"mention_channels,omitzero"`
	Embeds              []Embed                     `json:"embeds"`
	StickerItems        []StickerItem               `json:"sticker_items,omitzero"`
	Reactions           []Reaction                  `json:"reactions,omitzero"`
	Components          []MessageComponent          `json:"components,omitzero"` // All components included in the message.
	WebhookID           Snowflake                   `json:"webhook_id,omitempty"`
	ApplicationID       Snowflake                   `json:"application_id,omitempty"`
	GuildID             Snowflake                   `json:"guild_id,omitempty"`
	Flags               MessageFlags                `json:"flags,omitempty"`
	ChannelID           Snowflake                   `json:"channel_id"`
	ID                  Snowflake                   `json:"id"`
	Position            uint32                      `json:"position,omitempty"`
	Type                MessageType                 `json:"type,omitempty"`
	MentionEveryone     bool                        `json:"mention_everyone"` // Whether this message mentions everyone.
	TTS                 bool                        `json:"tts"`              // Whether this was a TTS message.
	Pinned              bool                        `json:"pinned"`
}

// https://docs.discord.com/developers/resources/message#message-reference-structure
type MessageReference struct {
	MessageID       Snowflake            `json:"message_id,omitempty"`
	ChannelID       Snowflake            `json:"channel_id,omitempty"`
	GuildID         Snowflake            `json:"guild_id,omitempty"`
	Type            MessageReferenceType `json:"type,omitempty"`
	FailIfNotExists bool                 `json:"fail_if_not_exists"`
}

// https://docs.discord.com/developers/resources/message#message-reference-types
type MessageReferenceType uint8

const (
	DEFAULT_MESSAGE_REFERENCE_TYPE MessageReferenceType = iota // A standard reference used by replies. Coupled with: referenced_message
	FORWARD_MESSAGE_REFERENCE_TYPE                             // Reference used to point to a message at a point in time. Coupled with: message_snapshot
)

// https://docs.discord.com/developers/resources/message#message-interaction-metadata-object-application-command-interaction-metadata-structure
type MessageInteractionMetadata struct {
	TargetUser      *User           `json:"target_user,omitempty"`
	User            User            `json:"user"`
	ID              Snowflake       `json:"id"`
	TargetMessageID Snowflake       `json:"target_message_id,omitempty"`
	Type            InteractionType `json:"type"`
}

// https://docs.discord.com/developers/resources/message#attachment-object-attachment-flags
type AttachmentFlags = BitSet

const (
	IS_CLIP_ATTACHMENT_FLAG                    AttachmentFlags = 1 << 0
	IS_THUMBNAIL_ATTACHMENT_FLAG               AttachmentFlags = 1 << 1
	IS_REMIX_ATTACHMENT_FLAG                   AttachmentFlags = 1 << 2
	IS_SPOILER_ATTACHMENT_FLAG                 AttachmentFlags = 1 << 3
	CONTAINS_EXPLICIT_CONTENT_ATTACHMENT_FLAG  AttachmentFlags = 1 << 4
	IS_ANIMATED_ATTACHMENT_FLAG                AttachmentFlags = 1 << 5
	CONTAINS_GORE_CONTENT_ATTACHMENT_FLAG      AttachmentFlags = 1 << 6
	CONTAINS_SELF_HARM_CONTENT_ATTACHMENT_FLAG AttachmentFlags = 1 << 7
)

// https://docs.discord.com/developers/resources/message#attachment-object
type Attachment struct {
	ClipCreatedAt      *time.Time      `json:"clip_created_at,omitempty"`
	URL                string          `json:"url"`
	Waveform           string          `json:"waveform,omitempty"`
	Title              string          `json:"title,omitempty"`
	Description        string          `json:"description,omitempty"`
	ContentType        string          `json:"content_type,omitempty"`
	ProxyURL           string          `json:"proxy_url"`
	Placeholder        string          `json:"placeholder,omitempty"`
	FileName           string          `json:"filename"`
	ClipParticipants   []User          `json:"clip_participants,omitzero"`
	ID                 Snowflake       `json:"id"`
	Size               uint64          `json:"size"`
	Flags              AttachmentFlags `json:"flags,omitempty"`
	DurationSecs       float64         `json:"duration_secs,omitempty"`
	Height             uint32          `json:"height,omitempty"`
	Width              uint32          `json:"width,omitempty"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitempty"`
	Ephemeral          bool            `json:"ephemeral"`
}

// https://docs.discord.com/developers/resources/poll#poll-object
type Poll struct {
	Expiry      *time.Time     `json:"expiry,omitempty"`
	Results     *PoolResult    `json:"results,omitempty"`
	Question    PollMedia      `json:"question"`
	Answers     []PollAnswer   `json:"answers,omitzero"`
	Duration    uint16         `json:"duration,omitempty"` // Number of hours the poll should be open for, up to 32 days (defaults to 24)
	Multiselect bool           `json:"allow_multiselect"`
	LayoutType  PoolLayoutType `json:"layout_type"`
}

// https://docs.discord.com/developers/resources/poll#poll-media-object
type PollMedia struct {
	Emoji *Emoji `json:"emoji,omitempty"`
	Text  string `json:"text,omitempty"`
}

// https://docs.discord.com/developers/resources/poll#poll-answer-object
type PollAnswer struct {
	PollMedia PollMedia `json:"poll_media"`
	AnswerID  uint32    `json:"answer_id,omitempty"`
}

// https://docs.discord.com/developers/resources/poll#poll-results-object
type PoolResult struct {
	AnswerCounts []PoolAnswerCounts `json:"answer_counts"`
	Finalized    bool               `json:"is_finalized"`
}

// https://docs.discord.com/developers/resources/poll#poll-results-object
type PoolAnswerCounts struct {
	ID      uint32 `json:"id"` // The answer_id
	Count   uint32 `json:"count"`
	MeVoted bool   `json:"me_voted"`
}
