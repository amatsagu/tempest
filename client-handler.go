package tempest

import (
	"crypto/ed25519"
	"encoding/json"
	"io"
	"net/http"
)

func (client *Client) HandleDiscordRequest(w http.ResponseWriter, r *http.Request) {
	// Deprecated since Go v1.22 - Please specify http method when registering handler.
	//
	// if r.Method != http.MethodPost {
	// 	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	// 	return
	// }

	verified := verifyRequest(r, ed25519.PublicKey(client.PublicKey))
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var extractor InteractionTypeExtractor
	err = json.Unmarshal(buf, &extractor)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch extractor.Type {
	case PING_INTERACTION_TYPE:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyPingResponse)
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		itx, command, available := client.seekCommand(interaction)
		if !available {
			w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
			w.Write(bodyUnknownCommandResponse)
			return
		}

		itx.Client = client

		w.WriteHeader(http.StatusNoContent)

		if !command.AvailableInDM && interaction.GuildID == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
			return
		}

		command.SlashCommandHandler(&itx)

		if client.postCommandHandler != nil {
			client.postCommandHandler(command, &itx)
		}
		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var itx ComponentInteraction
		err := json.Unmarshal(buf, &itx)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		itx.Client = client
		fn, available := client.components[itx.Data.CustomID]
		if available && fn != nil {
			itx.w = w
			fn(itx)
			return
		}

		client.qMu.RLock()
		signalChannel, available := client.queuedComponents[itx.Data.CustomID]
		client.qMu.RUnlock()
		if available && signalChannel != nil {
			w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
			w.Write(bodyAcknowledgeResponse)
			signalChannel <- &itx
			return
		}

		if client.componentHandler != nil {
			itx.w = w
			client.componentHandler(&itx)
		}

		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		itx, command, available := client.seekCommand(interaction)
		if !available || command.AutoCompleteHandler == nil || len(command.Options) == 0 {
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
		return
	case MODAL_SUBMIT_INTERACTION_TYPE:
		var itx ModalInteraction
		err := json.Unmarshal(buf, &itx)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		fn, available := client.modals[itx.Data.CustomID]
		if available && fn != nil {
			itx.w = w
			fn(itx)
			return
		}

		client.qMu.RLock()
		signalChannel, available := client.queuedModals[itx.Data.CustomID]
		client.qMu.RUnlock()
		if available && signalChannel != nil {
			w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
			w.Write(bodyAcknowledgeResponse)
			signalChannel <- &itx
		}

		if client.modalHandler != nil {
			itx.w = w
			client.modalHandler(&itx)
		}

		return
	}
}
