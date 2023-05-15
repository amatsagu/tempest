package tempest

import (
	"crypto/ed25519"
	"net/http"

	"github.com/sugawarayuuta/sonnet"
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

	var extractor InteractionTypeExtractor
	err := sonnet.NewDecoder(r.Body).Decode(&extractor)
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
		err := sonnet.NewDecoder(r.Body).Decode(&extractor)
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
				body, err := sonnet.Marshal(ResponseMessage{
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
		err := sonnet.NewDecoder(r.Body).Decode(&extractor)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			panic(err) // Should never happen
		}

		signalChannel, available := client.queuedComponents[interaction.Data.CustomID]
		if available && signalChannel != nil {
			*signalChannel <- &interaction
			acknowledgeComponentInteraction(w)
			return
		}

		if client.componentHandler != nil {
			client.componentHandler(interaction)
		}
		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		err := sonnet.NewDecoder(r.Body).Decode(&extractor)
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
		body, err := sonnet.Marshal(ResponseAutoComplete{
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
	}
}
