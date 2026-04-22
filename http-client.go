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
	rawData, cleanup, verified := client.verifyRequest(r, client.PublicKey, MAX_REQUEST_BODY_SIZE)
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer cleanup()

	var extractor InteractionTypeExtractor
	if err := json.Unmarshal(rawData, &extractor); err != nil {
		client.tracef("Received interaction event but failed to extract type: %v", err)
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	// Buffered channel ensures the handler doesn't block if the HTTP request times out.
	responseCh := make(chan []byte, 1)
	responderFn := func(res Response) error {
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

	switch extractor.Type {
	case PING_INTERACTION_TYPE:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		if _, err := w.Write(bodyPingResponse); err != nil {
			client.tracef("Failed to write ping response: %v.", err)
		}
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal(rawData, &interaction); err != nil {
			client.tracef("Received command interaction event but failed to parse its data: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.HTTPClient = client
		interaction.responder = responderFn

		go client.commandInteractionHandler(interaction, responseCh)
		client.awaitResponse(w, responseCh)

		return
	case MESSAGE_COMPONENT_INTERACTION_TYPE:
		var interaction ComponentInteraction
		if err := json.Unmarshal(rawData, &interaction); err != nil {
			client.tracef("Received component interaction event but failed to parse its data: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.HTTPClient = client
		interaction.responder = responderFn

		go client.componentInteractionHandler(interaction, responseCh)
		client.awaitResponse(w, responseCh)

		return
	case APPLICATION_COMMAND_AUTO_COMPLETE_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal(rawData, &interaction); err != nil {
			client.tracef("Received auto complete interaction event but failed to parse its data: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.HTTPClient = client
		interaction.responder = responderFn

		choices := client.autoCompleteInteractionHandler(interaction)

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
		if _, err := w.Write(body); err != nil {
			client.tracef("Failed to write auto complete response: %v.", err)
		}
		return
	case MODAL_SUBMIT_INTERACTION_TYPE:
		var interaction ModalInteraction
		if err := json.Unmarshal(rawData, &interaction); err != nil {
			client.tracef("Received modal interaction event but failed to parse its data: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		interaction.BaseClient = client.BaseClient
		interaction.HTTPClient = client
		interaction.responder = responderFn

		go client.modalInteractionHandler(interaction, responseCh)
		client.awaitResponse(w, responseCh)

		return
	}
}

// Verifies incoming request if it's from Discord. Returns the body bytes if verification was successful.
func (client *HTTPClient) verifyRequest(r *http.Request, key ed25519.PublicKey, maxSize int64) ([]byte, func(), bool) {
	signature := r.Header.Get("X-Signature-Ed25519")
	if signature == "" {
		return nil, nil, false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, nil, false
	}

	if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
		return nil, nil, false
	}

	timestamp := r.Header.Get("X-Signature-Timestamp")
	if timestamp == "" {
		return nil, nil, false
	}

	buf := client.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	buf.WriteString(timestamp)

	defer func() {
		if err := r.Body.Close(); err != nil {
			client.tracef("failed to close request body: %v", err)
		}
	}()
	_, err = buf.ReadFrom(io.LimitReader(r.Body, maxSize))
	if err != nil {
		client.bufferPool.Put(buf)
		return nil, nil, false
	}

	if ed25519.Verify(key, buf.Bytes(), sig) {
		return buf.Bytes()[len(timestamp):], func() {
			if int64(buf.Cap()) <= maxSize {
				client.bufferPool.Put(buf)
			}
		}, true
	}

	if int64(buf.Cap()) <= maxSize {
		client.bufferPool.Put(buf)
	}
	return nil, nil, false
}

func (client *HTTPClient) awaitResponse(w http.ResponseWriter, ch chan []byte) {
	select {
	case response := <-ch:
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		if _, err := w.Write(response); err != nil {
			client.tracef("failed to write response: %v", err)
		}
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

	handler, isQueued := client.queuedComponents.Get(interaction.Data.CustomID)
	if isQueued && time.Now().After(handler.Expire) {
		isQueued = false
		client.queuedComponents.Delete(interaction.Data.CustomID)
		if handler.OnTimeout != nil {
			go handler.OnTimeout()
		}
	}

	hasGlobal := client.componentHandler != nil

	if isQueued {
		client.tracef("Received component interaction with matching custom ID for dynamic handler - moved to listener.")
		select {
		case responseCh <- bodyAcknowledgeResponse:
		default:
		}

		interaction.deferred = true
		handler.Handler(&interaction)
		return
	}

	if hasGlobal {
		client.tracef("Received component interaction - moved to defined component handler.")
		client.componentHandler(&interaction)
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

	handler, isQueued := client.queuedModals.Get(interaction.Data.CustomID)
	if isQueued && time.Now().After(handler.Expire) {
		isQueued = false
		client.queuedModals.Delete(interaction.Data.CustomID)
		if handler.OnTimeout != nil {
			go handler.OnTimeout()
		}
	}

	hasGlobal := client.modalHandler != nil

	if isQueued {
		client.tracef("Received modal interaction with matching custom ID for dynamic handler - moved to listener.")
		select {
		case responseCh <- bodyAcknowledgeResponse:
		default:
		}

		interaction.deferred = true
		handler.Handler(&interaction)
		return
	}

	if hasGlobal {
		client.tracef("Received modal interaction - moved to defined modal handler.")
		client.modalHandler(&interaction)
		return
	}

	client.tracef("Dropped modal interaction. You see this trace message because client received modal interaction but there's no defined handler for it.")
}
