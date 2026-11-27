package builder

import "github.com/amatsagu/tempest/discord"

// Builds a Ping ACK interaction response.
func PongResponse() discord.ResponseMessage {
	return discord.ResponseMessage{Type: discord.PongResponseType}
}

// Builds a deferred channel message ("Bot is thinking...") response.
func DeferredResponse(ephemeral bool) discord.ResponseMessage {
	var data *discord.ResponseMessageData
	if ephemeral {
		data = &discord.ResponseMessageData{Flags: discord.EphemeralMessageFlag}
	}

	return discord.ResponseMessage{
		Type: discord.DeferredChannelMessageWithSourceResponseType,
		Data: data,
	}
}

// Builds a deferred component update ACK response.
func DeferredUpdateResponse() discord.ResponseMessage {
	return discord.ResponseMessage{Type: discord.DeferredUpdateMessageResponseType}
}

// Builds message payload data for interaction responses or REST endpoints.
func MessageData(content string, flags discord.MessageFlags, components ...discord.MessageComponent) discord.ResponseMessageData {
	return discord.ResponseMessageData{
		Content:    content,
		Flags:      flags,
		Components: components,
	}
}

func messageResponse(kind discord.ResponseType, content string, flags discord.MessageFlags, components []discord.MessageComponent) discord.ResponseMessage {
	return discord.ResponseMessage{
		Type: kind,
		Data: &discord.ResponseMessageData{
			Content:    content,
			Flags:      flags,
			Components: components,
		},
	}
}

// Builds an immediate interaction message reply.
func MessageResponse(content string, flags discord.MessageFlags, components ...discord.MessageComponent) discord.ResponseMessage {
	return messageResponse(discord.ChannelMessageWithSourceResponseType, content, flags, components)
}

// Builds an ephemeral text-only interaction reply.
func EphemeralMessage(content string) discord.ResponseMessage {
	return MessageResponse(content, discord.EphemeralMessageFlag)
}

// Builds an interaction response that edits the source component's message.
func UpdateMessageResponse(content string, flags discord.MessageFlags, components ...discord.MessageComponent) discord.ResponseMessage {
	return messageResponse(discord.UpdateMessageResponseType, content, flags, components)
}

// Builds the inner data payload for a modal dialog.
func ModalData(customID, title string, components ...discord.ModalComponent) discord.ResponseModalData {
	return discord.ResponseModalData{
		CustomID:   customID,
		Title:      title,
		Components: components,
	}
}

// Builds a modal dialog interaction response.
func ModalResponse(customID, title string, components ...discord.ModalComponent) discord.ResponseModal {
	return discord.ResponseModal{
		Type: discord.ModalResponseType,
		Data: &discord.ResponseModalData{
			CustomID:   customID,
			Title:      title,
			Components: components,
		},
	}
}

// Builds autocomplete choice payload data.
func AutocompleteData(choices ...discord.CommandOptionChoice) discord.ResponseAutoCompleteData {
	return discord.ResponseAutoCompleteData{Choices: choices}
}

// Builds an application command autocomplete response.
func AutocompleteResponse(choices ...discord.CommandOptionChoice) discord.ResponseAutoComplete {
	return discord.ResponseAutoComplete{
		Type: discord.AutocompleteResponseType,
		Data: &discord.ResponseAutoCompleteData{Choices: choices},
	}
}
