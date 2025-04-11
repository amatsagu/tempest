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

	buf := client.jsonBufferPool.Get().(*[]byte)
	defer client.jsonBufferPool.Put(buf)

	n, err := r.Body.Read(*buf)
	if err != nil && err != io.EOF {
		http.Error(w, "bad request - failed to read body payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var extractor InteractionTypeExtractor
	if err := json.Unmarshal((*buf)[:n], &extractor); err != nil {
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	switch extractor.Type {
	case PING_INTERACTION_TYPE:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyPingResponse)
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal((*buf)[:n], &interaction); err != nil {
			http.Error(w, "bad request - failed to decode CommandInteraction", http.StatusBadRequest)
			return
		}
		client.commandInteractionHandler(w, interaction)
		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var interaction ComponentInteraction
		if err := json.Unmarshal((*buf)[:n], &interaction); err != nil {
			http.Error(w, "bad request - failed to decode ComponentInteraction", http.StatusBadRequest)
			return
		}

		interaction.Client, interaction.w = client, w
		client.componentInteractionHandler(w, interaction)
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal((*buf)[:n], &interaction); err != nil {
			http.Error(w, "bad request - failed to decode CommandInteraction", http.StatusBadRequest)
			return
		}

		client.autoCompleteInteractionHandler(w, interaction)
		return
	case MODAL_SUBMIT_INTERACTION_TYPE:
		var interaction ModalInteraction
		if err := json.Unmarshal((*buf)[:n], &interaction); err != nil {
			http.Error(w, "bad request - failed to decode ModalInteraction", http.StatusBadRequest)
			return
		}

		interaction.Client, interaction.w = client, w
		client.modalInteractionHandler(w, interaction)
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

	if client.preCommandHandler != nil && !client.preCommandHandler(command, itx) {
		return
	}

	command.SlashCommandHandler(&itx)

	if client.postCommandHandler != nil {
		client.postCommandHandler(command, itx)
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
	fn, available := client.staticComponents.Get(interaction.Data.CustomID)
	if available {
		fn(interaction)
		return
	}

	signalChannel, available := client.queuedComponents.Get(interaction.Data.CustomID)
	if available && signalChannel != nil {
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyAcknowledgeResponse)
		signalChannel <- interaction
		return
	}

	if client.componentHandler != nil {
		client.componentHandler(interaction)
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
		signalChannel <- interaction
		return
	}

	if client.modalHandler != nil {
		client.modalHandler(interaction)
	}
}
