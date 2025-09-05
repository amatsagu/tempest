package tempest

import (
	"crypto/ed25519"
	"encoding/json"
	"io"
	"net/http"
)

func (client *Client) DiscordRequestHandler(w http.ResponseWriter, r *http.Request) {
	verified := verifyRequest(r, ed25519.PublicKey(client.PublicKey))
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limitedReader := http.MaxBytesReader(w, r.Body, MAX_REQUEST_BODY_SIZE)
	rawData, err := io.ReadAll(limitedReader)
	limitedReader.Close() // closes underlying r.Body
	if err != nil {
		http.Error(w, "bad request - failed to read body payload", http.StatusBadRequest)
		return
	}

	var interaction Interaction
	if err := json.Unmarshal(rawData, &interaction); err != nil {
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}
	interaction.Client = client

	switch interaction.Type {
	case PING_INTERACTION_TYPE:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyPingResponse)
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var data CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		client.commandInteractionHandler(w, CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		})
		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var data ComponentInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		client.componentInteractionHandler(w, ComponentInteraction{
			Interaction: &interaction,
			Data:        data,
			w:           w,
		})
		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var data CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		client.autoCompleteInteractionHandler(w, CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		})
		return
	case MODAL_SUBMIT_INTERACTION_TYPE:
		var data ModalInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		client.modalInteractionHandler(w, ModalInteraction{
			Interaction: &interaction,
			Data:        data,
			w:           w,
		})
		return
	}
}

func (client *Client) commandInteractionHandler(w http.ResponseWriter, interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyUnknownCommandResponse)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	itx.Client = client

	if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
		return
	}

	command.SlashCommandHandler(&itx)

	if client.postCommandHandler != nil {
		client.postCommandHandler(command, &itx)
	}
}

func (client *Client) autoCompleteInteractionHandler(w http.ResponseWriter, interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	choices := command.AutoCompleteHandler(itx)
	body, err := json.Marshal(ResponseAutoComplete{
		Type: AUTOCOMPLETE_RESPONSE_TYPE,
		Data: &ResponseAutoCompleteData{
			Choices: choices,
		},
	})

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	w.Write(body)
}

func (client *Client) componentInteractionHandler(w http.ResponseWriter, interaction ComponentInteraction) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		fn(interaction)
		return
	}

	if signalChan, ok := client.queuedComponents.Get(interaction.Data.CustomID); ok && signalChan != nil {
		w.Header().Set("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyAcknowledgeResponse)

		select {
		case signalChan <- &interaction:
			// Successfully sent
		default:
			// Receiver gone, drop silently
		}
		return
	}

	if client.componentHandler != nil {
		client.componentHandler(&interaction)
	}
}

func (client *Client) modalInteractionHandler(w http.ResponseWriter, interaction ModalInteraction) {
	fn, available := client.staticModals.Get(interaction.Data.CustomID)
	if available {
		fn(interaction)
		return
	}

	signalChannel, available := client.queuedModals.Get(interaction.Data.CustomID)
	if available && signalChannel != nil {
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyAcknowledgeResponse)
		signalChannel <- &interaction
		return
	}

	if client.modalHandler != nil {
		client.modalHandler(&interaction)
	}
}
