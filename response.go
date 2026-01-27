package tempest

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-interaction-callback-type
type ResponseType uint8

const (
	PONG_RESPONSE_TYPE ResponseType = iota + 1
	ACKNOWLEDGE_RESPONSE_TYPE
	CHANNEL_MESSAGE_RESPONSE_TYPE
	CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE
	DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE
	DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE // Only valid for component-based interactions.
	UPDATE_MESSAGE_RESPONSE_TYPE          // Only valid for component-based interactions.
	AUTOCOMPLETE_RESPONSE_TYPE
	MODAL_RESPONSE_TYPE // Not available for MODAL_SUBMIT and PING interactions.
	_
	_
	LAUNCH_ACTIVITY_RESPONSE_TYPE // Launch the Activity associated with the app. Only available for apps with Activities enabled.
)

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object
type ResponseMessage struct {
	Type ResponseType         `json:"type"`
	Data *ResponseMessageData `json:"data,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object
type ResponseAutoComplete struct {
	Type ResponseType              `json:"type"`
	Data *ResponseAutoCompleteData `json:"data,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object
type ResponseModal struct {
	Type ResponseType       `json:"type"`
	Data *ResponseModalData `json:"data,omitempty"`
}

// ResponseMessageData represents the data sent for message responses.
//
// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-messages
type ResponseMessageData struct {
	TTS             bool               `json:"tts"`                        // Whether the message is a TTS message.
	Content         string             `json:"content,omitempty"`          // The message content.
	Embeds          []Embed            `json:"embeds,omitzero"`            // Up to 10 rich content embeds to include.
	AllowedMentions *AllowedMentions   `json:"allowed_mentions,omitempty"` // Data pertaining to the message's allowed mentions.
	Flags           MessageFlags       `json:"flags,omitempty"`            // A bitfield containing message flags. Note that only [SUPPRESS_EMBEDS_MESSAGE_FLAG], [EPHEMERAL_MESSAGE_FLAG], [IS_COMPONENTS_V2_MESSAGE_FLAG], [IS_VOICE_MESSAGE_MESSAGE_FLAG] and [SUPPRESS_NOTIFICATIONS_MESSAGE_FLAG] can be set.
	Components      []MessageComponent `json:"components,omitzero"`        // Any components to send alongside the message.
	Attachments     []Attachment       `json:"attachments,omitzero"`       // Any attachments to send alongside the message.
	Poll            *Poll              `json:"poll,omitempty"`             // An optional poll to include in the message.
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-autocomplete
type ResponseAutoCompleteData struct {
	Choices []CommandOptionChoice `json:"choices,omitzero"` // The autocomplete choices to show, up to 25 in total.
}

// ResponseModalData represents the data sent for modal responses.
//
// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-modal
type ResponseModalData struct {
	CustomID   string           `json:"custom_id"`           // A custom identifier for the modal. Must be non-empty and between 1-100 characters.
	Title      string           `json:"title"`               // The title of the modal. Must be under 45 characters.
	Components []ModalComponent `json:"components,omitzero"` // 1-5 components that will make up the modal's body. Will be returned populated with user-filled data.
}
