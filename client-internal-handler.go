package tempest

import (
	"crypto/ed25519"
	"encoding/json"
	"io"
	"net/http"
)

func (client *Client) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	verified := verifyRequest(r, ed25519.PublicKey(client.PublicKey))
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		panic(err) // Should never happen
	}

	var extractor InteractionTypeExtractor
	err = json.Unmarshal(buf, &extractor)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		panic(err) // Should never happen
	}
	defer r.Body.Close()

	switch extractor.Type {
	case PING_INTERACTION_TYPE:
		w.Header().Add("Content-Type", "application/json")
		w.Write(private_PING_RESPONSE_RAW_BODY)
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		command, itx, available := client.seekCommand(interaction)
		if !available {
			w.Header().Add("Content-Type", "application/json")
			w.Write(private_UNKNOWN_COMMAND_RESPONSE_RAW_BODY)
			return
		}

		w.WriteHeader(http.StatusNoContent)

		if !command.AvailableInDM && interaction.GuildID == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if client.commandMiddlewareHandler != nil && !client.commandMiddlewareHandler(itx) {
			return
		}

		command.SlashCommandHandler(itx)
		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var itx ComponentInteraction
		err := json.Unmarshal(buf, &itx)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
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
			w.Header().Add("Content-Type", "application/json")
			w.Write(private_ACKNOWLEDGE_RESPONSE_RAW_BODY)
			signalChannel <- &itx
			return
		}

		if client.componentHandler != nil {
			itx.w = w
			client.componentHandler(itx)
		}

		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		command, itx, available := client.seekCommand(interaction)
		if !available || command.AutoCompleteHandler == nil || len(command.Options) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		choices := command.AutoCompleteHandler(AutoCompleteInteraction(itx))
		body, err := json.Marshal(ResponseAutoComplete{
			Type: AUTOCOMPLETE_RESPONSE_TYPE,
			Data: &ResponseAutoCompleteData{
				Choices: choices,
			},
		})

		if err != nil {
			panic("failed to parse payload received from client's \"auto complete\" handler (make sure it's in JSON format)")
		}

		w.Header().Add("Content-Type", "application/json")
		w.Write(body)
		return
	case MODAL_SUBMIT_INTERACTION_TYPE:
		var itx ModalInteraction
		err := json.Unmarshal(buf, &itx)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
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
			w.Header().Add("Content-Type", "application/json")
			w.Write(private_ACKNOWLEDGE_RESPONSE_RAW_BODY)
			signalChannel <- &itx
		}

		if client.modalHandler != nil {
			itx.w = w
			client.modalHandler(itx)
		}

		return
	}
}
