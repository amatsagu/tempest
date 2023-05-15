package tempest

import (
	"crypto/ed25519"
	"encoding/json"
	"io"
	"net/http"
)

func (client Client) handleDiscordWebhookRequests(w http.ResponseWriter, r *http.Request) {
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
		w.Write([]byte(`{"type":1}`))
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		command, ctx, available := client.seekCommand(interaction)
		if !available {
			terminateCommandInteraction(w)
			return
		}

		if !command.AvailableInDM && interaction.GuildID == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if interaction.Member != nil {
			interaction.Member.GuildID = interaction.GuildID
		}

		interaction.Client = &client
		if client.preCommandExecutionHandler != nil {
			content := client.preCommandExecutionHandler(ctx)
			if content != nil {
				body, err := json.Marshal(ResponseMessage{
					Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
					Data: content,
				})

				if err != nil {
					panic("failed to parse payload received from client's \"pre command execution\" handler (make sure it's in JSON format)")
				}

				w.Header().Add("Content-Type", "application/json")
				w.Write(body)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
		command.SlashCommandHandler(ctx)
		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var interaction ComponentInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		interaction.Client = &client
		w.Write([]byte(`{"type":6}`))

		fn, available := client.components[interaction.Data.CustomID]
		if available && fn != nil {
			fn(interaction)
			return
		}

		signalChannel, available := client.queuedComponents[interaction.Data.CustomID]
		if available && signalChannel != nil {
			*signalChannel <- &interaction
		}

		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		command, ctx, available := client.seekCommand(interaction)
		if !available || command.AutoCompleteHandler == nil || len(command.Options) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		choices := command.AutoCompleteHandler(AutoCompleteInteraction(ctx))
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
		var interaction ModalInteraction
		err := json.Unmarshal(buf, &interaction)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		interaction.Client = &client
		w.Write([]byte(`{"type":6}`))

		fn, available := client.modals[interaction.Data.CustomID]
		if available && fn != nil {
			fn(interaction)
			return
		}

		signalChannel, available := client.queuedModals[interaction.Data.CustomID]
		if available && signalChannel != nil {
			signalChannel <- &interaction
		}

		return
	}
}
