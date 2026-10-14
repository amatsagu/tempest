package discord

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-response-object-messages
type ResponseMessageData struct {
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitzero"`
	Poll            *Poll            `json:"poll,omitzero"`
	Content         string           `json:"content,omitzero"`
	Embeds          []Embed          `json:"embeds,omitzero"`
	Components      Components       `json:"components,omitzero"`
	Attachments     []Attachment     `json:"attachments,omitzero"`
	StickerIDs      []Snowflake      `json:"sticker_ids,omitzero"`
	Flags           MessageFlags     `json:"flags,omitzero"`
	TTS             bool             `json:"tts,omitzero"`
}

func NewMessageResponse(content string, flags MessageFlags, components ...Component) Response {
	return Response{
		Type: ResponseChannelMessageWithSource,
		Data: ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: Components(components),
		},
	}
}

func NewEphemeralMessage(content string) Response {
	return NewMessageResponse(content, MessageFlagEphemeral)
}

func NewUpdateMessageResponse(content string, flags MessageFlags, components ...Component) Response {
	return Response{
		Type: ResponseUpdateMessage,
		Data: ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: Components(components),
		},
	}
}
