package discord

import "time"

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

// https://docs.discord.com/developers/resources/message#message-object
type Message struct {
	EditedTimestamp *time.Time   `json:"edited_timestamp,omitzero"`
	Poll            *Poll        `json:"poll,omitzero"`
	Content         string       `json:"content,omitzero"`
	Mentions        []User       `json:"mentions,omitzero"`
	MentionRoles    []RoleID     `json:"mention_roles,omitzero"`
	Attachments     []Attachment `json:"attachments,omitzero"`
	Embeds          []Embed      `json:"embeds,omitzero"`
	Components      Components   `json:"components,omitzero"`
	Timestamp       time.Time    `json:"timestamp"`
	Author          User         `json:"author"`
	ID              MessageID    `json:"id"`
	ChannelID       ChannelID    `json:"channel_id"`
	Type            MessageType  `json:"type"`
	Flags           MessageFlags `json:"flags,omitzero"`
	TTS             bool         `json:"tts,omitzero"`
	MentionEveryone bool         `json:"mention_everyone,omitzero"`
	Pinned          bool         `json:"pinned,omitzero"`
}
