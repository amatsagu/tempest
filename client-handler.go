package qord

import (
	"encoding/json"
	"net/http"
	"qord/api"
	"qord/gateway"
)

func (client *Client) eventHandler(shardID uint16, packet gateway.EventPacket) {
	if client.customEventHandler != nil {
		client.customEventHandler(shardID, packet)
	}

	if packet.Event != gateway.INTERACTION_CREATE_EVENT {
		return
	}

	var interaction api.Interaction
	if err := json.Unmarshal(packet.Data, &interaction); err != nil {
		client.tracef("Received interaction event but failed to parse it: %v", err)
		return
	}

	interaction.ShardID = shardID
	interaction.Rest = client.Rest

	switch interaction.Type {
	case api.APPLICATION_COMMAND_INTERACTION_TYPE:
		var data api.CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			client.tracef("Received command interaction event but failed to parse its data: %v", err)
			return
		}

		client.commandInteractionHandler(api.CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		})
		return
	case api.MESSAGE_COMPONENT_INTERACTION_TYPE:
		var data api.ComponentInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			client.tracef("Received component interaction event but failed to parse its data: %v", err)
			return
		}

		client.componentInteractionHandler(api.ComponentInteraction{
			Interaction: &interaction,
			Data:        data,
		})
		return
	case api.APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var data api.CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			client.tracef("Received auto complete interaction event but failed to parse its data: %v", err)
			return
		}

		client.autoCompleteInteractionHandler(api.CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		})
		return
	case api.MODAL_SUBMIT_INTERACTION_TYPE:
		var data api.ModalInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			client.tracef("Received modal interaction event but failed to parse its data: %v", err)
			return
		}

		client.modalInteractionHandler(api.ModalInteraction{
			Interaction: &interaction,
			Data:        data,
		})
		return
	}
}

func (client *Client) commandInteractionHandler(interaction api.CommandInteraction) {
	itx, command, available := client.ItxManager.HandleCommandInteraction(interaction)
	if !available {
		itx.SendLinearReply("Oh uh.. It looks like you tried to use outdated/unknown command. Please report this bug to bot owner.", true)
		return
	}

	allowed := true
	if client.ItxManager.PreCommandHandler != nil &&
		!client.ItxManager.PreCommandHandler(command, &itx) {
		allowed = false
	}

	if allowed {
		command.SlashCommandHandler(&itx)
		if client.ItxManager.PostCommandHandler != nil {
			client.ItxManager.PostCommandHandler(command, &itx)
		}
	}
}

func (client *Client) autoCompleteInteractionHandler(interaction api.CommandInteraction) {
	itx, command, available := client.ItxManager.HandleCommandInteraction(interaction)
	if !available {
		return
	}

	choices := command.AutoCompleteHandler(itx)
	itx.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", api.ResponseAutoComplete{
		Type: api.AUTOCOMPLETE_RESPONSE_TYPE,
		Data: &api.ResponseAutoCompleteData{
			Choices: choices,
		},
	})
}

func (client *Client) componentInteractionHandler(interaction api.ComponentInteraction) {
	if fn, ok := client.ItxManager.Components.Get(interaction.Data.CustomID); ok {
		fn(interaction)
		return
	}

	if signalChan, ok := client.ItxManager.QueuedComponents.Get(interaction.Data.CustomID); ok && signalChan != nil {
		interaction.Acknowledge()
	}

	if client.ItxManager.ComponentHandler != nil {
		client.ItxManager.ComponentHandler(&interaction)
	}
}

func (client *Client) modalInteractionHandler(interaction api.ModalInteraction) {
	if fn, ok := client.ItxManager.Modals.Get(interaction.Data.CustomID); ok {
		fn(interaction)
		return
	}

	if signalChan, ok := client.ItxManager.QueuedModals.Get(interaction.Data.CustomID); ok && signalChan != nil {
		interaction.Acknowledge()
	}

	if client.ItxManager.ModalHandler != nil {
		client.ItxManager.ModalHandler(&interaction)
	}
}
