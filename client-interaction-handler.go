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

	var interaction Interaction
	err := sonnet.NewDecoder(r.Body).Decode(&interaction)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		panic(err) // Should never happen
	}
	defer r.Body.Close()

	switch interaction.Type {
	case PING_INTERACTION_TYPE:
		w.Write([]byte(`{"type":1}`))
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		command, interaction, available := client.seekCommand(interaction)
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
			content := client.preCommandExecutionHandler(interaction)
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
		command.SlashCommandHandler(interaction)
		return
	}
}
