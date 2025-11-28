package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	// Optimization: Create a shallow copy of the client to inject a custom interactionResponder
	// that captures the response channel for this specific request.
	clientCopy := client.Client
	interaction.Client = &clientCopy

	// Buffered channel ensures the handler doesn't block if the HTTP request times out.
	responseCh := make(chan []byte, 1)

	clientCopy.interactionResponder = func(itx *Interaction, resp Response) error {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		select {
		case responseCh <- body:
			return nil
		default:
			return errors.New("initial interaction response already sent")
		}
	}

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

		client.commandInteractionHandler(CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		}, responseCh)

		client.awaitResponse(w, responseCh)
		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var data ComponentInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		client.componentInteractionHandler(ComponentInteraction{
			Interaction: &interaction,
			Data:        data,
		}, responseCh)

		client.awaitResponse(w, responseCh)
		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var data CommandInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		choices := client.autoCompleteInteractionHandler(CommandInteraction{
			Interaction: &interaction,
			Data:        data,
		})

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
		var data ModalInteractionData
		if err := json.Unmarshal(interaction.Data, &data); err != nil {
			http.Error(w, "bad request - failed to decode Interaction.Data", http.StatusBadRequest)
			return
		}

		client.modalInteractionHandler(ModalInteraction{
			Interaction: &interaction,
			Data:        data,
		}, responseCh)

		client.awaitResponse(w, responseCh)
		return
	}
}

func (client *HTTPClient) awaitResponse(w http.ResponseWriter, ch chan []byte) {
	select {
	case response := <-ch:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(response)
	case <-time.After(2500 * time.Millisecond):
		w.WriteHeader(http.StatusNoContent)
	}
}

func (client *HTTPClient) commandInteractionHandler(interaction CommandInteraction, responseCh chan []byte) {
	go func() {
		itx, command, available := client.handleInteraction(interaction)
		if !available {
			client.tracef("Received command interaction but there's no matching command! (requested \"%s\")", interaction.Data.Name)
			responseCh <- bodyUnknownCommandResponse
			return
		}

		itx.Client = interaction.Client
		client.tracef("Received command interaction - moved to target command's handler.")

		if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
			return
		}

		command.SlashCommandHandler(&itx)

		if client.postCommandHandler != nil {
			client.postCommandHandler(command, &itx)
		}
	}()
}

func (client *HTTPClient) autoCompleteInteractionHandler(interaction CommandInteraction) []CommandOptionChoice {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Dropped auto complete interaction. You see this trace message because client received slash command's auto complete interaction but there's no defined handler for it.")
		return nil
	}

	client.tracef("Received slash command's auto complete interaction - moved to target (sub) command auto complete handler.")
	return command.AutoCompleteHandler(itx)
}

func (client *HTTPClient) componentInteractionHandler(interaction ComponentInteraction, responseCh chan []byte) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		client.tracef("Received component interaction with matching custom ID for static handler - moved to registered handler.")
		go fn(interaction)
		return
	}

	isQueued := client.queuedComponents.Has(interaction.Data.CustomID)
	hasGlobal := client.componentHandler != nil

	if isQueued || hasGlobal {
		if isQueued {
			client.tracef("Received component interaction with matching custom ID for dynamic handler - moved to listener.")
		} else {
			client.tracef("Received component interaction - moved to defined component handler.")
		}

		select {
		case responseCh <- bodyAcknowledgeResponse:
		default:
		}

		interaction.deferred = true

		go func() {
			if isQueued {
				client.queuedComponents.mu.RLock()
				if signalChan, ok := client.queuedComponents.cache[interaction.Data.CustomID]; ok {
					select {
					case signalChan <- &interaction:
					default:
					}
					client.queuedComponents.mu.RUnlock()
					return
				}
				client.queuedComponents.mu.RUnlock()
			}

			if client.componentHandler != nil {
				client.componentHandler(&interaction)
			}
		}()
		return
	}

	client.tracef("Dropped component interaction. You see this trace message because client received component interaction but there's no defined handler for it.")
}

func (client *HTTPClient) modalInteractionHandler(interaction ModalInteraction, responseCh chan []byte) {
	if fn, ok := client.staticModals.Get(interaction.Data.CustomID); ok {
		client.tracef("Received modal interaction with matching custom ID for static handler - moved to registered handler.")
		go fn(interaction)
		return
	}

	isQueued := client.queuedModals.Has(interaction.Data.CustomID)
	hasGlobal := client.modalHandler != nil

	if isQueued || hasGlobal {
		if isQueued {
			client.tracef("Received modal interaction with matching custom ID for dynamic handler - moved to listener.")
		} else {
			client.tracef("Received modal interaction - moved to defined modal handler.")
		}

		select {
		case responseCh <- bodyAcknowledgeResponse:
		default:
		}

		interaction.deferred = true

		go func() {
			if isQueued {
				client.queuedModals.mu.RLock()
				if signalChan, ok := client.queuedModals.cache[interaction.Data.CustomID]; ok {
					select {
					case signalChan <- &interaction:
					default:
					}
					client.queuedModals.mu.RUnlock()
					return
				}
				client.queuedModals.mu.RUnlock()
			}

			if client.modalHandler != nil {
				client.modalHandler(&interaction)
			}
		}()
		return
	}

	client.tracef("Dropped modal interaction. You see this trace message because client received modal interaction but there's no defined handler for it.")
}