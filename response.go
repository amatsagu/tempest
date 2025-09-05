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

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-messages
type ResponseMessageData struct {
	TTS             bool              `json:"tts"`
	Content         string            `json:"content,omitempty"`
	Embeds          []Embed           `json:"embeds,omitzero"`
	AllowedMentions *AllowedMentions  `json:"allowed_mentions,omitempty"`
	Flags           MessageFlags      `json:"flags,omitempty"`
	Components      []LayoutComponent `json:"components,omitzero"`
	Attachments     []Attachment      `json:"attachments,omitzero"`
	Poll            *Poll             `json:"poll,omitempty"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-autocomplete
type ResponseAutoCompleteData struct {
	Choices []CommandOptionChoice `json:"choices,omitzero"`
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-modal
type ResponseModalData struct {
	CustomID   string            `json:"custom_id"`
	Title      string            `json:"title"`
	Components []LayoutComponent `json:"components,omitzero"`
}
