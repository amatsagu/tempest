package builder

import "github.com/amatsagu/tempest/discord"

func PongResponse() discord.Response {
	return discord.Response{Type: discord.ResponsePong}
}

func DeferredResponse(ephemeral bool) discord.Response {
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

func DeferredUpdateResponse() discord.Response {
	return discord.Response{Type: discord.ResponseDeferredUpdateMessage}
}

func MessageResponse(content string, flags discord.MessageFlags, components ...discord.Component) discord.Response {
	return discord.Response{
		Type: discord.ResponseChannelMessageWithSource,
		Data: discord.ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: discord.Components(components),
		},
	}
}

func EphemeralMessage(content string) discord.Response {
	return MessageResponse(content, discord.MessageFlagEphemeral)
}

func UpdateMessageResponse(content string, flags discord.MessageFlags, components ...discord.Component) discord.Response {
	return discord.Response{
		Type: discord.ResponseUpdateMessage,
		Data: discord.ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: discord.Components(components),
		},
	}
}

func ModalResponse(customID, title string, components ...discord.Component) discord.Response {
	return discord.Response{
		Type: discord.ResponseModal,
		Data: discord.ResponseModalData{
			CustomID:   customID,
			Title:      title,
			Components: discord.Components(components),
		},
	}
}

func AutocompleteResponse(choices ...discord.CommandOptionChoice) discord.Response {
	return discord.Response{
		Type: discord.ResponseApplicationCommandAutocomplete,
		Data: discord.ResponseAutocompleteData{
			Choices: choices,
		},
	}
}
