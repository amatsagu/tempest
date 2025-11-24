package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

type HTTPClient struct {
	Client
	PublicKey ed25519.PublicKey
}

type HTTPClientOptions struct {
	ClientOptions
	PublicKey string
	Trace     bool // Whether to enable basic logging for the client actions.
}

func NewHTTPClient(opt HTTPClientOptions) HTTPClient {
	discordPublicKey, err := hex.DecodeString(opt.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	client := HTTPClient{
		Client: NewClient(ClientOptions{
			Token:                      opt.Token,
			DefaultInteractionContexts: opt.DefaultInteractionContexts,
			PreCommandHook:             opt.PreCommandHook,
			PostCommandHook:            opt.PostCommandHook,
			ComponentHandler:           opt.ComponentHandler,
			ModalHandler:               opt.ModalHandler,
		}),
		PublicKey: discordPublicKey,
	}

	if opt.Trace {
		client.traceLogger.SetOutput(os.Stdout)
		client.tracef("HTTP Client tracing enabled.")
	}

	return client
}

func (m *HTTPClient) tracef(format string, v ...any) {
	m.traceLogger.Printf("[(HTTP) CLIENT] "+format, v...)
}

func (client *HTTPClient) DiscordRequestHandler(w http.ResponseWriter, r *http.Request) {
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
	interaction.Client = &client.Client

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

		// Channel to receive the first response from the command handler
		responseChan := make(chan []byte, 1)

		client.commandInteractionHandler(w, CommandInteraction{
			Interaction:  &interaction,
			Data:         data,
			w:            w,
			responseChan: responseChan,
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

func (client *HTTPClient) commandInteractionHandler(w http.ResponseWriter, interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Received command interaction but there's no matching command! (requested \"%s\")", interaction.Data.Name)
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyUnknownCommandResponse)
		return
	}

	itx.Client = &client.Client
	client.tracef("Received command interaction - moved to target command's handler.")

	// Run command handler in goroutine
	go func() {
		allowed := true
		if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
			allowed = false
		}

		if allowed {
			command.SlashCommandHandler(&itx)
			if client.postCommandHandler != nil {
				client.postCommandHandler(command, &itx)
			}
		}
	}()

	// Wait for first response with timeout
	select {
	case response := <-itx.responseChan:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(response)
		return
	case <-time.After(2900 * time.Millisecond):
		// Timeout - send 204 to acknowledge
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

func (client *HTTPClient) autoCompleteInteractionHandler(w http.ResponseWriter, interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Dropped auto complete interaction. You see this trace message because client received slash command's auto complete interaction but there's no defined handler for it.")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	client.tracef("Received slash command's auto complete interaction - moved to target (sub) command auto complete handler.")
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

func (client *HTTPClient) componentInteractionHandler(w http.ResponseWriter, interaction ComponentInteraction) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		client.tracef("Received component interaction with matching custom ID for static handler - moved to registered handler.")
		fn(interaction)
		return
	}

	if signalChan, ok := client.queuedComponents.Get(interaction.Data.CustomID); ok && signalChan != nil {
		client.tracef("Received component interaction with matching custom ID for dynamic handler - moved to listener.")
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
		client.tracef("Received component interaction - moved to defined component handler.")
		client.componentHandler(&interaction)
		return
	}

	client.tracef("Dropped component interaction. You see this trace message because client received component interaction but there's no defined handler for it.")
}

func (client *HTTPClient) modalInteractionHandler(w http.ResponseWriter, interaction ModalInteraction) {
	fn, available := client.staticModals.Get(interaction.Data.CustomID)
	if available {
		client.tracef("Received modal interaction with matching custom ID for static handler - moved to registered handler.")
		fn(interaction)
		return
	}

	if signalChan, ok := client.queuedModals.Get(interaction.Data.CustomID); ok && signalChan != nil {
		client.tracef("Received modal interaction with matching custom ID for dynamic handler - moved to listener.")
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyAcknowledgeResponse)

		select {
		case signalChan <- &interaction:
			// Successfully sent
		default:
			// Receiver gone, drop silently
		}
		return
	}

	if client.modalHandler != nil {
		client.tracef("Received modal interaction - moved to defined modal handler.")
		client.modalHandler(&interaction)
		return
	}

	client.tracef("Dropped modal interaction. You see this trace message because client received modal interaction but there's no defined handler for it.")
}
