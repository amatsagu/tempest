package discord

import (
	"bytes"
	"encoding/json/v2"
	"time"
)

// https://docs.discord.com/developers/resources/message#message-object-message-types
type MessageType uint8

const (
	MessageTypeDefault                              MessageType = 0
	MessageTypeRecipientAdd                         MessageType = 1
	MessageTypeRecipientRemove                      MessageType = 2
	MessageTypeCall                                 MessageType = 3
	MessageTypeChannelNameChange                    MessageType = 4
	MessageTypeChannelIconChange                    MessageType = 5
	MessageTypeChannelPinnedMessage                 MessageType = 6
	MessageTypeUserJoin                             MessageType = 7
	MessageTypeGuildBoost                           MessageType = 8
	MessageTypeGuildBoostTier1                      MessageType = 9
	MessageTypeGuildBoostTier2                      MessageType = 10
	MessageTypeGuildBoostTier3                      MessageType = 11
	MessageTypeChannelFollowAdd                     MessageType = 12
	MessageTypeGuildDiscoveryDisqualified           MessageType = 14
	MessageTypeGuildDiscoveryRequalified            MessageType = 15
	MessageTypeGuildDiscoveryGracePeriodInitialWarn MessageType = 16
	MessageTypeGuildDiscoveryGracePeriodFinalWarn   MessageType = 17
	MessageTypeThreadCreated                        MessageType = 18
	MessageTypeReply                                MessageType = 19
	MessageTypeChatInputCommand                     MessageType = 20
	MessageTypeThreadStarterMessage                 MessageType = 21
	MessageTypeGuildInviteReminder                  MessageType = 22
	MessageTypeContextMenuCommand                   MessageType = 23
	MessageTypeAutoModerationAction                 MessageType = 24
	MessageTypeRoleSubscriptionPurchase             MessageType = 25
	MessageTypeInteractionPremiumUpsell             MessageType = 26
	MessageTypeStageStart                           MessageType = 27
	MessageTypeStageEnd                             MessageType = 28
	MessageTypeStageSpeaker                         MessageType = 29
	MessageTypeStageTopic                           MessageType = 31
	MessageTypeGuildApplicationPremiumSubscription  MessageType = 32
	MessageTypeGuildIncidentAlertModeEnabled        MessageType = 36
	MessageTypeGuildIncidentAlertModeDisabled       MessageType = 37
	MessageTypeGuildIncidentReportRaid              MessageType = 38
	MessageTypeGuildIncidentReportFalseAlarm        MessageType = 39
	MessageTypePurchaseNotification                 MessageType = 44
	MessageTypePollResult                           MessageType = 46
)

// https://docs.discord.com/developers/resources/message#message-object-message-flags
type MessageFlags = BitSet

const (
	MessageFlagCrossposted                      MessageFlags = 1 << 0
	MessageFlagIsCrosspost                      MessageFlags = 1 << 1
	MessageFlagSuppressEmbeds                   MessageFlags = 1 << 2
	MessageFlagSourceMessageDeleted             MessageFlags = 1 << 3
	MessageFlagUrgent                           MessageFlags = 1 << 4
	MessageFlagHasThread                        MessageFlags = 1 << 5
	MessageFlagEphemeral                        MessageFlags = 1 << 6
	MessageFlagLoading                          MessageFlags = 1 << 7
	MessageFlagFailedToMentionSomeRolesInThread MessageFlags = 1 << 8
	MessageFlagSuppressNotifications            MessageFlags = 1 << 12
	MessageFlagIsVoiceMessage                   MessageFlags = 1 << 13
	MessageFlagHasSnapshot                      MessageFlags = 1 << 14
	MessageFlagIsComponentsV2                   MessageFlags = 1 << 15
)

// Nonce is a message nonce, normalized to a string; decoding accepts integer or string.
//
// https://docs.discord.com/developers/resources/message#message-object
type Nonce string

func (n *Nonce) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, nullLiteral) {
		*n = ""
		return nil
	}

	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}

		*n = Nonce(s)
		return nil
	}

	*n = Nonce(b)
	return nil
}

// https://docs.discord.com/developers/resources/message#message-reference-types
type MessageReferenceType uint8

const (
	MessageReferenceDefault MessageReferenceType = 0
	MessageReferenceForward MessageReferenceType = 1
)

// https://docs.discord.com/developers/resources/message#message-reference-structure
type MessageReference struct {
	FailIfNotExists *bool                `json:"fail_if_not_exists,omitzero"`
	MessageID       MessageID            `json:"message_id,omitzero"`
	ChannelID       ChannelID            `json:"channel_id,omitzero"`
	GuildID         GuildID              `json:"guild_id,omitzero"`
	Type            MessageReferenceType `json:"type,omitzero"`
}

// https://docs.discord.com/developers/resources/message#message-object-message-activity-types
type MessageActivityType uint8

const (
	MessageActivityJoin        MessageActivityType = 1
	MessageActivitySpectate    MessageActivityType = 2
	MessageActivityListen      MessageActivityType = 3
	MessageActivityJoinRequest MessageActivityType = 5
)

// https://docs.discord.com/developers/resources/message#message-object-message-activity-structure
type MessageActivity struct {
	PartyID string              `json:"party_id,omitzero"`
	Type    MessageActivityType `json:"type"`
}

// https://docs.discord.com/developers/resources/message#message-call-object-message-call-object-structure
type MessageCall struct {
	EndedTimestamp *time.Time  `json:"ended_timestamp,omitzero"`
	Participants   []Snowflake `json:"participants"`
}

// https://docs.discord.com/developers/resources/message#reaction-object-reaction-structure
type Reaction struct {
	BurstColors  []string             `json:"burst_colors,omitzero"`
	Emoji        Emoji                `json:"emoji"`
	CountDetails ReactionCountDetails `json:"count_details"`
	Count        int                  `json:"count"`
	Me           bool                 `json:"me"`
	MeBurst      bool                 `json:"me_burst"`
}

// https://docs.discord.com/developers/resources/message#reaction-count-details-object-reaction-count-details-structure
type ReactionCountDetails struct {
	Burst  int `json:"burst"`
	Normal int `json:"normal"`
}

// https://docs.discord.com/developers/resources/message#channel-mention-object-channel-mention-structure
type ChannelMention struct {
	Name    string      `json:"name"`
	ID      ChannelID   `json:"id"`
	GuildID GuildID     `json:"guild_id"`
	Type    ChannelType `json:"type"`
}

// https://docs.discord.com/developers/resources/message#role-subscription-data-object-role-subscription-data-object-structure
type RoleSubscriptionData struct {
	TierName                  string    `json:"tier_name"`
	RoleSubscriptionListingID Snowflake `json:"role_subscription_listing_id"`
	TotalMonthsSubscribed     int       `json:"total_months_subscribed"`
	IsRenewal                 bool      `json:"is_renewal"`
}

// https://docs.discord.com/developers/resources/message#message-object
type Message struct {
	Timestamp            time.Time                `json:"timestamp"`
	EditedTimestamp      *time.Time               `json:"edited_timestamp,omitzero"`
	Poll                 *Poll                    `json:"poll,omitzero"`
	Member               *Member                  `json:"member,omitzero"`
	Thread               *Channel                 `json:"thread,omitzero"`
	MessageReference     *MessageReference        `json:"message_reference,omitzero"`
	ReferencedMessage    *Message                 `json:"referenced_message,omitzero"`
	Activity             *MessageActivity         `json:"activity,omitzero"`
	Application          *Application             `json:"application,omitzero"`
	RoleSubscriptionData *RoleSubscriptionData    `json:"role_subscription_data,omitzero"`
	Call                 *MessageCall             `json:"call,omitzero"`
	Resolved             *InteractionDataResolved `json:"resolved,omitzero"`
	Content              string                   `json:"content,omitzero"`
	Nonce                Nonce                    `json:"nonce,omitzero"`
	MentionRoles         []RoleID                 `json:"mention_roles,omitzero"`
	MentionChannels      []ChannelMention         `json:"mention_channels,omitzero"`
	Attachments          []Attachment             `json:"attachments,omitzero"`
	Embeds               []Embed                  `json:"embeds,omitzero"`
	Components           Components               `json:"components,omitzero"`
	StickerItems         []StickerItem            `json:"sticker_items,omitzero"`
	Stickers             []Sticker                `json:"stickers,omitzero"`
	Reactions            []Reaction               `json:"reactions,omitzero"`
	Mentions             []User                   `json:"mentions,omitzero"`
	Author               User                     `json:"author"`
	ID                   MessageID                `json:"id"`
	ChannelID            ChannelID                `json:"channel_id"`
	GuildID              GuildID                  `json:"guild_id,omitzero"`
	WebhookID            Snowflake                `json:"webhook_id,omitzero"`
	ApplicationID        AppID                    `json:"application_id,omitzero"`
	Position             int                      `json:"position,omitzero"`
	Flags                MessageFlags             `json:"flags,omitzero"`
	Type                 MessageType              `json:"type"`
	TTS                  bool                     `json:"tts,omitzero"`
	MentionEveryone      bool                     `json:"mention_everyone,omitzero"`
	Pinned               bool                     `json:"pinned,omitzero"`
}
