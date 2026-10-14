package builder

import "github.com/amatsagu/tempest/discord"

func NewPongResponse() discord.Response {
	return discord.Response{Type: discord.ResponsePong}
}

func NewDeferredResponse(ephemeral bool) discord.Response {
	var flags discord.MessageFlags
	if ephemeral {
		flags = discord.MessageFlagEphemeral
	}

	return discord.Response{
		Type: discord.ResponseDeferredChannelMessageWithSource,
		Data: discord.ResponseMessageData{
			Flags: flags,
		},
	}
}

func NewDeferredUpdateResponse() discord.Response {
	return discord.Response{Type: discord.ResponseDeferredUpdateMessage}
}

func NewMessageResponse(content string, flags discord.MessageFlags, components ...discord.Component) discord.Response {
	return discord.Response{
		Type: discord.ResponseChannelMessageWithSource,
		Data: discord.ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: discord.Components(components),
		},
	}
}

func NewEphemeralMessage(content string) discord.Response {
	return NewMessageResponse(content, discord.MessageFlagEphemeral)
}

func NewUpdateMessageResponse(content string, flags discord.MessageFlags, components ...discord.Component) discord.Response {
	return discord.Response{
		Type: discord.ResponseUpdateMessage,
		Data: discord.ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: discord.Components(components),
		},
	}
}

func NewModalResponse(customID, title string, components ...discord.Component) discord.Response {
	return discord.Response{
		Type: discord.ResponseModal,
		Data: discord.ResponseModalData{
			CustomID:   customID,
			Title:      title,
			Components: discord.Components(components),
		},
	}
}

func NewAutocompleteResponse(choices ...discord.CommandOptionChoice) discord.Response {
	return discord.Response{
		Type: discord.ResponseApplicationCommandAutocomplete,
		Data: discord.ResponseAutocompleteData{
			Choices: choices,
		},
	}
}
