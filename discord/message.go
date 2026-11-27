package discord

import (
	"encoding/json/jsontext"
	"strconv"
	"time"
)

// https://docs.discord.com/developers/resources/message#message-object-message-flags
type MessageFlags = BitSet

const (
	CrosspostedMessageFlag MessageFlags = 1 << iota
	IsCrosspostMessageFlag
	SuppressEmbedsMessageFlag
	SourceMessageDeletedMessageFlag
	UrgentMessageFlag
	HasThreadMessageFlag
	EphemeralMessageFlag
	LoadingMessageFlag
	FailedToMentionSomeRolesInThreadMessageFlag
	_
	_
	_
	SuppressNotificationsMessageFlag
	IsVoiceMessageMessageFlag
	HasSnapshotMessageFlag
	IsComponentsV2MessageFlag // When used, regular content, embeds, poll & stickers fields will be ignored.
)

// https://docs.discord.com/developers/resources/message#message-object-message-types
type MessageType uint8

const (
	DefaultMessageType MessageType = iota
	RecipientAddMessageType
	RecipientRemoveMessageType
	CallMessageType
	ChannelNameChangeMessageType
	ChannelIconChangeMessageType
	ChannelPinnedMessageType
	UserJoinMessageType
	GuildBoostMessageType
	GuildBoostTier1MessageType
	GuildBoostTier2MessageType
	GuildBoostTier3MessageType
	ChannelFollowAddMessageType
	_
	GuildDiscoveryDisqualifiedMessageType
	GuildDiscoveryRequalifiedMessageType
	GuildDiscoveryGracePeriodInitialWarningMessageType
	GuildDiscoveryGracePeriodFinalWarningMessageType
	ThreadCreatedMessageType
	ReplyMessageType
	ChatInputCommandMessageType
	ThreadStarterMessageType
	GuildInviteReminderMessageType
	ContextMenuCommandMessageType
	AutoModerationActionMessageType
	RoleSubscriptionPurchaseMessageType
	InteractionPremiumUpsellMessageType
	StageStartMessageType
	StageEndMessageType
	StageSpeakerMessageType
	_
	StageTopicMessageType
	GuildApplicationPremiumSubscriptionMessageType
	_
	_
	_
	GuildIncidentAlertModeEnabledMessageType
	GuildIncidentAlertModeDisabledMessageType
	GuildIncidentReportRaidMessageType
	GuildIncidentReportFalseAlarmMessageType
	_
	_
	_
	_
	PurchaseNotificationMessageType
	_
	PollResultMessageType
)

// https://docs.discord.com/developers/resources/channel#channel-object-channel-types
type ChannelType uint8

const (
	GuildTextChannelType ChannelType = iota
	DMChannelType
	GuildVoiceChannelType
	GroupDMChannelType
	GuildCategoryChannelType
	GuildAnnouncementChannelType // Formerly news channel.
	_
	_
	_
	_
	GuildAnnouncementThreadChannelType
	GuildPublicThreadChannelType
	GuildPrivateThreadChannelType
	GuildStageVoiceChannelType
	GuildDirectoryChannelType
	GuildForumChannelType
	GuildMediaChannelType
)

func (ct ChannelType) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.Uint(uint64(ct)))
}

func (ct ChannelType) MarshalJSON() ([]byte, error) {
	return strconv.AppendUint(make([]byte, 0, 3), uint64(ct), 10), nil
}

// https://docs.discord.com/developers/resources/sticker#sticker-object-sticker-format-types
type StickerFormatType uint8

const (
	PNGStickerFormatType StickerFormatType = iota + 1
	APNGStickerFormatType
	LottieStickerFormatType
	GIFStickerFormatType
)

// https://docs.discord.com/developers/resources/message#allowed-mentions-object-allowed-mention-types
type AllowedMentionsType string

const (
	AllowedRoleMentionType     AllowedMentionsType = "roles"
	AllowedUsersMentionType    AllowedMentionsType = "users"
	AllowedEveryoneMentionType AllowedMentionsType = "everyone"
)

// https://docs.discord.com/developers/resources/poll#layout-type
type PollLayoutType uint8

const (
	DefaultPollLayoutType PollLayoutType = iota + 1
)

// https://docs.discord.com/developers/resources/message#allowed-mentions-object
type AllowedMentions struct {
	Parse       []AllowedMentionsType `json:"parse,omitzero"`
	Roles       []RoleID              `json:"roles,omitzero"`
	Users       []UserID              `json:"users,omitzero"`
	RepliedUser bool                  `json:"replied_user,omitzero"`
}

// https://docs.discord.com/developers/resources/channel#channel-object
type PartialChannel struct {
	Name            string          `json:"name"`
	ID              ChannelID       `json:"id"`
	PermissionFlags PermissionFlags `json:"permissions,string"`
	Type            ChannelType     `json:"type"`
}

// https://docs.discord.com/developers/resources/message#channel-mention-object-channel-mention-structure
type ChannelMention struct {
	Name    string      `json:"name"`
	ID      ChannelID   `json:"id"`
	GuildID GuildID     `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// https://docs.discord.com/developers/resources/message#reaction-count-details-object-reaction-count-details-structure
type ReactionCountDetails struct {
	Burst  uint32 `json:"burst"`
	Normal uint32 `json:"normal"`
}

// https://docs.discord.com/developers/resources/message#reaction-object-reaction-structure
type Reaction struct {
	BurstColors  []string             `json:"burst_colors,omitzero"` // HEX colors used for super reaction
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
	User          *User    `json:"user,omitzero"`
	Name          string   `json:"name,omitzero"` // Note: may be empty for deleted emojis.
	Roles         []RoleID `json:"roles,omitzero"`
	ID            EmojiID  `json:"id,omitzero"`
	RequireColons bool     `json:"require_colons,omitzero"`
	Managed       bool     `json:"managed,omitzero"`
	Animated      bool     `json:"animated,omitzero"`
	Available     bool     `json:"available,omitzero"`
}

func (e Emoji) Mention() string {
	if e.ID.Snowflake == 0 {
		return e.Name
	}
	return e.ID.Mention(e.Name, e.Animated)
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-flags
type EmbedFlags = BitSet

const (
	ContainsExplicitContentEmbedFlag EmbedFlags = 1 << 4
	IsContentInventoryEntryEmbedFlag EmbedFlags = 1 << 5
	ContainsGoreContentEmbedFlag     EmbedFlags = 1 << 6
	ContainsSelfHarmContentEmbedFlag EmbedFlags = 1 << 7
)

// https://docs.discord.com/developers/resources/message#embed-object-embed-media-flags
type EmbedMediaFlags = BitSet

const (
	IsAnimatedEmbedMediaFlag EmbedMediaFlags = 1 << 5
)

// https://docs.discord.com/developers/resources/message#embed-object
type Embed struct {
	Author      *EmbedAuthor    `json:"author,omitzero"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitzero"`
	Footer      *EmbedFooter    `json:"footer,omitzero"`
	Image       *EmbedImage     `json:"image,omitzero"`
	Video       *EmbedVideo     `json:"video,omitzero"`
	Provider    *EmbedProvider  `json:"provider,omitzero"`
	Timestamp   *time.Time      `json:"timestamp,omitzero"`
	Title       string          `json:"title,omitzero"`
	Type        string          `json:"type,omitzero"`
	URL         string          `json:"url,omitzero"`
	Description string          `json:"description,omitzero"`
	Fields      []EmbedField    `json:"fields,omitzero"`
	Color       uint32          `json:"color,omitzero"`
	Flags       EmbedFlags      `json:"flags,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-author-structure
type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitzero"`
	IconURL      string `json:"icon_url,omitzero"`
	ProxyIconURL string `json:"proxy_icon_url,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-image-structure
type EmbedThumbnail struct {
	URL                string          `json:"url"`
	ProxyURL           string          `json:"proxy_url,omitzero"`
	ContentType        string          `json:"content_type,omitzero"`
	Placeholder        string          `json:"placeholder,omitzero"`
	Description        string          `json:"description,omitzero"`
	Flags              EmbedMediaFlags `json:"flags,omitzero"`
	Width              uint32          `json:"width,omitzero"`
	Height             uint32          `json:"height,omitzero"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-field-structure
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-footer-structure
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitzero"`
	ProxyIconURL string `json:"proxy_icon_url,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-image-structure
type EmbedImage struct {
	URL                string          `json:"url"`
	ProxyURL           string          `json:"proxy_url,omitzero"`
	ContentType        string          `json:"content_type,omitzero"`
	Placeholder        string          `json:"placeholder,omitzero"`
	Description        string          `json:"description,omitzero"`
	Flags              EmbedMediaFlags `json:"flags,omitzero"`
	Width              uint32          `json:"width,omitzero"`
	Height             uint32          `json:"height,omitzero"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-video-structure
type EmbedVideo struct {
	URL                string          `json:"url,omitzero"`
	ProxyURL           string          `json:"proxy_url,omitzero"`
	ContentType        string          `json:"content_type,omitzero"`
	Placeholder        string          `json:"placeholder,omitzero"`
	Description        string          `json:"description,omitzero"`
	Flags              EmbedMediaFlags `json:"flags,omitzero"`
	Width              uint32          `json:"width,omitzero"`
	Height             uint32          `json:"height,omitzero"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-provider-structure
type EmbedProvider struct {
	URL  string `json:"url,omitzero"`
	Name string `json:"name,omitzero"`
}

// https://docs.discord.com/developers/resources/message#message-object
type Message struct {
	MessageReference    *MessageReference           `json:"message_reference,omitzero"`
	InteractionMetadata *MessageInteractionMetadata `json:"interaction_metadata,omitzero"`
	Poll                *Poll                       `json:"poll,omitzero"`
	Member              *Member                     `json:"member,omitzero"`
	Resolved            *InteractionDataResolved    `json:"resolved,omitzero"`
	Author              *User                       `json:"author,omitzero"` // The author of this message.
	Timestamp           *time.Time                  `json:"timestamp,omitzero"`
	EditedTimestamp     *time.Time                  `json:"edited_timestamp,omitzero"`
	ReferencedMessage   *Message                    `json:"referenced_message,omitzero"`
	Content             string                      `json:"content,omitzero"` // The message's content.
	Nonce               Nonce                       `json:"nonce,omitzero"`   // Used for validating a message was sent.
	Attachments         []Attachment                `json:"attachments,omitzero"`
	Mentions            []User                      `json:"mentions,omitzero"` // All users specifically mentioned in the message.
	MentionRoles        []RoleID                    `json:"mention_roles,omitzero"`
	MentionChannels     []ChannelMention            `json:"mention_channels,omitzero"`
	Embeds              []Embed                     `json:"embeds,omitzero"`
	StickerItems        []StickerItem               `json:"sticker_items,omitzero"`
	Reactions           []Reaction                  `json:"reactions,omitzero"`
	Components          []MessageComponent          `json:"components,omitzero"` // All components included in the message.
	WebhookID           Snowflake                   `json:"webhook_id,omitzero"`
	ApplicationID       AppID                       `json:"application_id,omitzero"`
	GuildID             GuildID                     `json:"guild_id,omitzero"`
	Flags               MessageFlags                `json:"flags,omitzero"`
	ChannelID           ChannelID                   `json:"channel_id,omitzero"`
	ID                  MessageID                   `json:"id,omitzero"`
	Position            uint32                      `json:"position,omitzero"`
	Type                MessageType                 `json:"type,omitzero"`
	MentionEveryone     bool                        `json:"mention_everyone,omitzero"` // Whether this message mentions everyone.
	TTS                 bool                        `json:"tts,omitzero"`              // Whether this was a TTS message.
	Pinned              bool                        `json:"pinned,omitzero"`
}

// https://docs.discord.com/developers/resources/message#message-reference-structure
type MessageReference struct {
	FailIfNotExists *bool                `json:"fail_if_not_exists,omitzero"` // Whether to error if the referenced message does not exist (defaults to true when nil/omitted).
	MessageID       MessageID            `json:"message_id,omitzero"`
	ChannelID       ChannelID            `json:"channel_id,omitzero"`
	GuildID         GuildID              `json:"guild_id,omitzero"`
	Type            MessageReferenceType `json:"type,omitzero"`
}

// https://docs.discord.com/developers/resources/message#message-reference-types
type MessageReferenceType uint8

const (
	DefaultMessageReferenceType MessageReferenceType = iota // A standard reference used by replies. Coupled with: referenced_message
	ForwardMessageReferenceType                             // Reference used to point to a message at a point in time. Coupled with: message_snapshot
)

// https://docs.discord.com/developers/resources/message#message-interaction-metadata-object-application-command-interaction-metadata-structure
type MessageInteractionMetadata struct {
	TargetUser      *User           `json:"target_user,omitzero"`
	User            User            `json:"user"`
	ID              Snowflake       `json:"id"`
	TargetMessageID MessageID       `json:"target_message_id,omitzero"`
	Type            InteractionType `json:"type"`
}

// https://docs.discord.com/developers/resources/message#attachment-object-attachment-flags
type AttachmentFlags = BitSet

const (
	IsClipAttachmentFlag                  AttachmentFlags = 1 << 0
	IsThumbnailAttachmentFlag             AttachmentFlags = 1 << 1
	IsRemixAttachmentFlag                 AttachmentFlags = 1 << 2
	IsSpoilerAttachmentFlag               AttachmentFlags = 1 << 3
	ContainsExplicitContentAttachmentFlag AttachmentFlags = 1 << 4
	IsAnimatedAttachmentFlag              AttachmentFlags = 1 << 5
	ContainsGoreContentAttachmentFlag     AttachmentFlags = 1 << 6
	ContainsSelfHarmContentAttachmentFlag AttachmentFlags = 1 << 7
)

// https://docs.discord.com/developers/resources/message#attachment-object
type Attachment struct {
	ClipCreatedAt      *time.Time      `json:"clip_created_at,omitzero"`
	URL                string          `json:"url"`
	Waveform           string          `json:"waveform,omitzero"`
	Title              string          `json:"title,omitzero"`
	Description        string          `json:"description,omitzero"`
	ContentType        string          `json:"content_type,omitzero"`
	ProxyURL           string          `json:"proxy_url"`
	Placeholder        string          `json:"placeholder,omitzero"`
	FileName           string          `json:"filename"`
	ClipParticipants   []User          `json:"clip_participants,omitzero"`
	ID                 AttachmentID    `json:"id"`
	Size               uint64          `json:"size,omitzero"`
	Flags              AttachmentFlags `json:"flags,omitzero"`
	DurationSecs       float64         `json:"duration_secs,omitzero"`
	Height             uint32          `json:"height,omitzero"`
	Width              uint32          `json:"width,omitzero"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitzero"`
	Ephemeral          bool            `json:"ephemeral,omitzero"`
}

// https://docs.discord.com/developers/resources/poll#poll-object
type Poll struct {
	Expiry      *time.Time     `json:"expiry,omitzero"`
	Results     *PollResult    `json:"results,omitzero"`
	Question    PollMedia      `json:"question"`
	Answers     []PollAnswer   `json:"answers,omitzero"`
	Duration    uint16         `json:"duration,omitzero"` // Number of hours the poll should be open for, up to 32 days (defaults to 24)
	Multiselect bool           `json:"allow_multiselect,omitzero"`
	LayoutType  PollLayoutType `json:"layout_type"`
}

// https://docs.discord.com/developers/resources/poll#poll-media-object
type PollMedia struct {
	Emoji *Emoji `json:"emoji,omitzero"`
	Text  string `json:"text,omitzero"`
}

// https://docs.discord.com/developers/resources/poll#poll-answer-object
type PollAnswer struct {
	PollMedia PollMedia `json:"poll_media"`
	AnswerID  uint32    `json:"answer_id,omitzero"`
}

// https://docs.discord.com/developers/resources/poll#poll-results-object
type PollResult struct {
	AnswerCounts []PollAnswerCounts `json:"answer_counts"`
	Finalized    bool               `json:"is_finalized"`
}

// https://docs.discord.com/developers/resources/poll#poll-results-object
type PollAnswerCounts struct {
	ID      uint32 `json:"id"` // The answer_id
	Count   uint32 `json:"count"`
	MeVoted bool   `json:"me_voted"`
}
