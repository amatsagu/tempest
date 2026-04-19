package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type HTTPClient struct {
	*BaseClient
	PublicKey  ed25519.PublicKey
	bufferPool *sync.Pool
}

type HTTPClientOptions struct {
	BaseClientOptions
	PublicKey string
	Trace     bool // Whether to enable basic logging for the client actions.
}

func NewHTTPClient(opt HTTPClientOptions) *HTTPClient {
	discordPublicKey, err := hex.DecodeString(opt.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	client := HTTPClient{
		BaseClient: NewBaseClient(BaseClientOptions{
			Token:                      opt.Token,
			DefaultInteractionContexts: opt.DefaultInteractionContexts,
			PreCommandHook:             opt.PreCommandHook,
			PostCommandHook:            opt.PostCommandHook,
			ComponentHandler:           opt.ComponentHandler,
			ModalHandler:               opt.ModalHandler,
			Logger:                     opt.Logger,
		}),
		PublicKey: discordPublicKey,
		bufferPool: &sync.Pool{
			New: func() any {
				b := new(bytes.Buffer)
				b.Grow(1024 * 32) // 32KB is plenty for most interactions
				return b
			},
		},
	}

	if opt.Trace {
		w := client.traceLogger.Writer()
		if w == nil || w == io.Discard {
			client.traceLogger.SetOutput(os.Stdout)
		}
		client.tracef("HTTP Client tracing enabled.")
	}

	return &client
}

func (m *HTTPClient) tracef(format string, v ...any) {
	m.traceLogger.Printf("[(HTTP) CLIENT] "+format, v...)
}

// This handler already runs in dedicated goroutine (from std http server behavior).
// Due to default HTTP server behavior, this goroutine cannot block for more than 3s,
// to achieve that we use client interaction responder trick.
func (client *HTTPClient) DiscordRequestHandler(w http.ResponseWriter, r *http.Request) {
	rawData, verified := client.verifyRequest(r, client.PublicKey, MAX_REQUEST_BODY_SIZE)
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var interaction Interaction
	if err := json.Unmarshal(rawData, &interaction); err != nil {
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	interaction.BaseClient = client.BaseClient
	interaction.HTTPClient = client

	// Buffered channel ensures the handler doesn't block if the HTTP request times out.
	responseCh := make(chan []byte, 1)
	interaction.responder = func(res Response) error {
		body, err := json.Marshal(res)
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

		// Move to extra goroutine in case modal handler needs more than 3s.
		go client.commandInteractionHandler(CommandInteraction{
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

		// Move to extra goroutine in case modal handler needs more than 3s.
		go client.componentInteractionHandler(ComponentInteraction{
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

		// Move to extra goroutine in case modal handler needs more than 3s.
		go client.modalInteractionHandler(ModalInteraction{
			Interaction: &interaction,
			Data:        data,
		}, responseCh)

		client.awaitResponse(w, responseCh)
		return
	}
}

// Verifies incoming request if it's from Discord. Returns the body bytes if verification was successful.
func (client *HTTPClient) verifyRequest(r *http.Request, key ed25519.PublicKey, maxSize int64) ([]byte, bool) {
	signature := r.Header.Get("X-Signature-Ed25519")
	if signature == "" {
		return nil, false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, false
	}

	if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
		return nil, false
	}

	timestamp := r.Header.Get("X-Signature-Timestamp")
	if timestamp == "" {
		return nil, false
	}

	buf := client.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer client.bufferPool.Put(buf)

	buf.WriteString(timestamp)

	defer r.Body.Close()
	_, err = buf.ReadFrom(io.LimitReader(r.Body, maxSize))
	if err != nil {
		return nil, false
	}

	if ed25519.Verify(key, buf.Bytes(), sig) {
		// We copy the body bytes because we're returning the buffer to the pool.
		bodyBytes := make([]byte, buf.Len()-len(timestamp))
		copy(bodyBytes, buf.Bytes()[len(timestamp):])
		return bodyBytes, true
	}

	return nil, false
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
	itx, command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Received command interaction but there's no matching command! (requested \"%s\")", itx.Data.Name)
		responseCh <- bodyUnknownCommandResponse
		return
	}

	client.tracef("Received command interaction - moved to target command's handler.")

	if client.preCommandHandler != nil && !client.preCommandHandler(command, &itx) {
		return
	}

	command.SlashCommandHandler(&itx)

	if client.postCommandHandler != nil {
		client.postCommandHandler(command, &itx)
	}
}

func (client *HTTPClient) autoCompleteInteractionHandler(interaction CommandInteraction) []CommandOptionChoice {
	itx, command, available := client.handleInteraction(interaction)
	if !available || command.AutoCompleteHandler == nil {
		client.tracef("Dropped auto complete interaction. You see this trace message because client received slash command's auto complete interaction but there's no defined handler for it.")
		return nil
	}

	client.tracef("Received slash command's auto complete interaction - moved to target (sub) command auto complete handler.")
	return command.AutoCompleteHandler(itx)
}

func (client *HTTPClient) componentInteractionHandler(interaction ComponentInteraction, responseCh chan []byte) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		client.tracef("Received component interaction with matching custom ID for static handler - moved to registered handler.")
		fn(interaction)
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
		return
	}

	client.tracef("Dropped component interaction. You see this trace message because client received component interaction but there's no defined handler for it.")
}

func (client *HTTPClient) modalInteractionHandler(interaction ModalInteraction, responseCh chan []byte) {
	if fn, ok := client.staticModals.Get(interaction.Data.CustomID); ok {
		client.tracef("Received modal interaction with matching custom ID for static handler - moved to registered handler.")
		fn(interaction)
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
		return
	}

	client.tracef("Dropped modal interaction. You see this trace message because client received modal interaction but there's no defined handler for it.")
}
