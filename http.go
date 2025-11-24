package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	Client
	PublicKey ed25519.PublicKey

	applicationAuthorizedEventHandler   func(event *ApplicationAuthorizedEvent)
	applicationDeauthorizedEventHandler func(event *ApplicationDeauthorizedEvent)
	entitlementCreationEventHandler     func(event *EntitlementCreationEvent)
}

type HTTPClientOptions struct {
	ClientOptions
	PublicKey                           string
	ApplicationAuthorizedEventHandler   func(event *ApplicationAuthorizedEvent)   // Function that runs when app/bot is added by user or server.
	ApplicationDeauthorizedEventHandler func(event *ApplicationDeauthorizedEvent) // Function that runs when app/bot is removed from user apps.
	EntitlementCreationEventHandler     func(event *EntitlementCreationEvent)     // Function that runs when user purchases or is otherwise granted one of your app's SKUs.
}

func NewHTTPClient(opt HTTPClientOptions) HTTPClient {
	discordPublicKey, err := hex.DecodeString(opt.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	return HTTPClient{
		Client: NewClient(ClientOptions{
			Token:                      opt.Token,
			DefaultInteractionContexts: opt.DefaultInteractionContexts,
			PreCommandHook:             opt.PreCommandHook,
			PostCommandHook:            opt.PostCommandHook,
			ComponentHandler:           opt.ComponentHandler,
			ModalHandler:               opt.ModalHandler,
		}),
		PublicKey:                           discordPublicKey,
		applicationAuthorizedEventHandler:   opt.ApplicationAuthorizedEventHandler,
		applicationDeauthorizedEventHandler: opt.ApplicationDeauthorizedEventHandler,
		entitlementCreationEventHandler:     opt.EntitlementCreationEventHandler,
	}
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

func (client *HTTPClient) DiscordWebhookEventHandler(w http.ResponseWriter, r *http.Request) {
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

	var webhook WebhookEvent
	if err := json.Unmarshal(rawData, &webhook); err != nil {
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	if webhook.Type == PING_WEBHOOK_TYPE || webhook.Event == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch webhook.Event.Type {
	case APPLICATION_AUTHORIZED_EVENT_TYPE:
		var event ApplicationAuthorizedEvent
		if err := json.Unmarshal(webhook.Event.Data, &event); err != nil {
			http.Error(w, "bad request - failed to decode Webhook.Event.Data", http.StatusBadRequest)
			return
		}

		if client.applicationAuthorizedEventHandler != nil {
			event.Client = &client.Client
			client.applicationAuthorizedEventHandler(&event)
		}
		return
	case APPLICATION_DEAUTHORIZED_EVENT_TYPE:
		var event ApplicationDeauthorizedEvent
		if err := json.Unmarshal(webhook.Event.Data, &event); err != nil {
			http.Error(w, "bad request - failed to decode Webhook.Event.Data", http.StatusBadRequest)
			return
		}

		if client.applicationDeauthorizedEventHandler != nil {
			event.Client = &client.Client
			client.applicationDeauthorizedEventHandler(&event)
		}
		return
	case ENTITLEMENT_CREATE_EVENT_TYPE:
		var event EntitlementCreationEvent
		if err := json.Unmarshal(webhook.Event.Data, &event); err != nil {
			http.Error(w, "bad request - failed to decode Webhook.Event.Data", http.StatusBadRequest)
			return
		}

		if client.entitlementCreationEventHandler != nil {
			event.Client = &client.Client
			client.entitlementCreationEventHandler(&event)
		}
		return
	}

}

func (client *HTTPClient) commandInteractionHandler(w http.ResponseWriter, interaction CommandInteraction) {
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyUnknownCommandResponse)
		return
	}

	itx.Client = &client.Client

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

func (client *HTTPClient) componentInteractionHandler(w http.ResponseWriter, interaction ComponentInteraction) {
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

func (client *HTTPClient) modalInteractionHandler(w http.ResponseWriter, interaction ModalInteraction) {
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
