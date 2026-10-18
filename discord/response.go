package discord

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-response-object-interaction-callback-type
type ResponseType uint8

const (
	ResponsePong ResponseType = iota + 1
	_
	_
	ResponseChannelMessageWithSource
	ResponseDeferredChannelMessageWithSource
	ResponseDeferredUpdateMessage
	ResponseUpdateMessage
	ResponseApplicationCommandAutocomplete
	ResponseModal
	_
	_
	ResponseLaunchActivity
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-response-object-interaction-response-structure
type Response struct {
	Data any          `json:"data,omitzero"`
	Type ResponseType `json:"type"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-callback-interaction-callback-activity-instance-resource
type InteractionCallbackActivityInstanceResource struct {
	ID string `json:"id"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-callback-interaction-callback-resource-object
type InteractionCallbackResource struct {
	Activity *InteractionCallbackActivityInstanceResource `json:"activity_instance,omitzero"`
	Message  *Message                                     `json:"message,omitzero"`
	Type     ResponseType                                 `json:"type"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-callback-interaction-callback-response-object
type InteractionCallbackResponse struct {
	Resource    *InteractionCallbackResource           `json:"resource,omitzero"`
	Interaction InteractionCallbackResponseInteraction `json:"interaction"`
}

// InteractionCallbackResponseInteraction is the "interaction" member of a callback
// response; it mirrors InteractionCallbackResource plus delivery fields.
//
// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-callback-interaction-callback-response-object
type InteractionCallbackResponseInteraction struct {
	Activity                 *InteractionCallbackActivityInstanceResource `json:"activity_instance,omitzero"`
	ID                       Snowflake                                    `json:"id"`
	ResponseMessageID        Snowflake                                    `json:"response_message_id,omitzero"`
	Type                     ResponseType                                 `json:"type"`
	ResponseMessageLoading   bool                                         `json:"response_message_loading,omitzero"`
	ResponseMessageEphemeral bool                                         `json:"response_message_ephemeral,omitzero"`
}
