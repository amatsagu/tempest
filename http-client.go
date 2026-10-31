package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	json "encoding/json/v2"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type HTTPClient struct {
	*BaseClient
	sessionPool *sync.Pool
	timerPool   *sync.Pool
	readerPool  *sync.Pool
	bufferPool  *sync.Pool
	PublicKey   ed25519.PublicKey
}

type HTTPClientOptions struct {
	PublicKey string
	BaseClientOptions
	Trace bool // Whether to enable basic logging for the client actions.
}

func NewHTTPClient(opt HTTPClientOptions) *HTTPClient {
	discordPublicKey, err := hex.DecodeString(opt.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}
	if len(discordPublicKey) != ed25519.PublicKeySize {
		panic("discord public key must be exactly 32 bytes")
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
			RestOptions:                opt.RestOptions,
			Context:                    opt.Context,
		}),
		PublicKey: discordPublicKey,
		sessionPool: &sync.Pool{
			New: func() any {
				return &httpSession{
					responseCh: make(chan responsePayload, 1),
				}
			},
		},
		timerPool: &sync.Pool{
			New: func() any {
				t := time.NewTimer(time.Hour)
				t.Stop()
				return t
			},
		},
		readerPool: &sync.Pool{
			New: func() any {
				return new(io.LimitedReader)
			},
		},
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
	if !m.trace {
		return
	}

	m.traceLogger.Printf("[(HTTP) CLIENT] "+format, v...)
}

type httpSession struct {
	responseCh   chan responsePayload
	responseSent atomic.Bool
	timedOut     atomic.Bool
}

type responsePayload struct {
	buf  *bytes.Buffer
	data []byte
}

func (client *HTTPClient) acquireHTTPSession() *httpSession {
	if client.sessionPool == nil {
		return &httpSession{
			responseCh: make(chan responsePayload, 1),
		}
	}
	s := client.sessionPool.Get().(*httpSession)
	s.responseSent.Store(false)
	s.timedOut.Store(false)
	return s
}

func (client *HTTPClient) releaseHTTPSession(s *httpSession) {
	if client.sessionPool != nil {
		client.sessionPool.Put(s)
	}
}

// This handler already runs in dedicated goroutine (from std http server behavior).
// Due to default HTTP server behavior, this goroutine cannot block for more than 3s,
// to achieve that we use client interaction responder trick.
func (client *HTTPClient) DiscordRequestHandler(w http.ResponseWriter, r *http.Request) {
	rawData, buf, verified := client.verifyRequest(r, client.PublicKey, MAX_REQUEST_BODY_SIZE)
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer client.releaseBuffer(buf)

	itxType, err := peekInteractionType(rawData)
	if err != nil {
		client.tracef("Received interaction event but failed to extract type: %v", err)
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	switch itxType {
	case PING_INTERACTION_TYPE:
		w.Header().Set("Content-Type", CONTENT_TYPE_JSON)
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
		client.serveCommand(w, r, &interaction)
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
		client.serveComponent(w, r, &interaction)
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
		interaction.ctx = r.Context()

		choices := client.autoCompleteInteractionHandler(&interaction)

		buf := client.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		res := ResponseAutoComplete{
			Type: AUTOCOMPLETE_RESPONSE_TYPE,
			Data: &ResponseAutoCompleteData{
				Choices: choices,
			},
		}
		if err := json.MarshalWrite(buf, &res); err != nil {
			client.releaseBuffer(buf)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", CONTENT_TYPE_JSON)
		if _, err := w.Write(buf.Bytes()); err != nil {
			client.tracef("Failed to write auto complete response: %v.", err)
		}
		client.releaseBuffer(buf)

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
		client.serveModal(w, r, &interaction)
		return
	}
}

func (client *HTTPClient) serveCommand(w http.ResponseWriter, r *http.Request, interaction *CommandInteraction) {
	session := client.acquireHTTPSession()
	ctx, cancel, stop := client.interactionContext(r, &session.responseSent)
	interaction.ctx = ctx
	interaction.session = session

	go func() {
		if cancel != nil {
			defer cancel()
		}
		if stop != nil {
			defer stop()
		}
		defer func() {
			if !session.responseSent.Load() {
				select {
				case session.responseCh <- responsePayload{}:
				default:
				}
			}
		}()
		client.commandInteractionHandler(interaction)
	}()

	client.awaitResponse(r.Context(), w, session)
}

func (client *HTTPClient) serveComponent(w http.ResponseWriter, r *http.Request, interaction *ComponentInteraction) {
	session := client.acquireHTTPSession()
	ctx, cancel, stop := client.interactionContext(r, &session.responseSent)
	interaction.ctx = ctx
	interaction.session = session

	go func() {
		if cancel != nil {
			defer cancel()
		}
		if stop != nil {
			defer stop()
		}
		defer func() {
			if !session.responseSent.Load() {
				select {
				case session.responseCh <- responsePayload{}:
				default:
				}
			}
		}()
		client.componentInteractionHandler(interaction)
	}()

	client.awaitResponse(r.Context(), w, session)
}

func (client *HTTPClient) serveModal(w http.ResponseWriter, r *http.Request, interaction *ModalInteraction) {
	session := client.acquireHTTPSession()
	ctx, cancel, stop := client.interactionContext(r, &session.responseSent)
	interaction.ctx = ctx
	interaction.session = session

	go func() {
		if cancel != nil {
			defer cancel()
		}

		if stop != nil {
			defer stop()
		}

		defer func() {
			if !session.responseSent.Load() {
				select {
				case session.responseCh <- responsePayload{}:
				default:
				}
			}
		}()

		client.modalInteractionHandler(interaction)
	}()

	client.awaitResponse(r.Context(), w, session)
}

func (client *HTTPClient) interactionContext(r *http.Request, responseSent *atomic.Bool) (context.Context, context.CancelFunc, func()) {
	// Keep request values for tracing/user code, but do not let ServeHTTP
	// returning cancel follow-up work. Request cancellation still cancels the
	// interaction until its initial response has been sent.
	reqCtx := r.Context()
	if reqCtx.Done() == nil {
		return reqCtx, nil, nil
	}

	ctx, cancel := context.WithCancel(context.WithoutCancel(reqCtx))
	stop := context.AfterFunc(reqCtx, func() {
		if !responseSent.Load() {
			cancel()
		}
	})

	return ctx, cancel, func() { stop() }
}

func (client *HTTPClient) acquireTimer(d time.Duration) *time.Timer {
	if client.timerPool == nil {
		return time.NewTimer(d)
	}
	t := client.timerPool.Get().(*time.Timer)
	t.Reset(d)
	return t
}

func (client *HTTPClient) releaseTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	if client.timerPool != nil {
		client.timerPool.Put(t)
	}
}

func (client *HTTPClient) awaitResponse(reqCtx context.Context, w http.ResponseWriter, session *httpSession) {
	timer := client.acquireTimer(2500 * time.Millisecond)
	defer client.releaseTimer(timer)

	done := reqCtx.Done()
	if done == nil {
		select {
		case payload := <-session.responseCh:
			if len(payload.data) > 0 {
				client.writeResponse(w, payload.data)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}

			if payload.buf != nil {
				client.releaseBuffer(payload.buf)
			}

			if !session.timedOut.Load() {
				client.releaseHTTPSession(session)
			}
		case <-timer.C:
			session.timedOut.Store(true)
			w.WriteHeader(http.StatusNoContent)
		}

		return
	}

	select {
	case payload := <-session.responseCh:
		if len(payload.data) > 0 {
			client.writeResponse(w, payload.data)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}

		if payload.buf != nil {
			client.releaseBuffer(payload.buf)
		}

		if !session.timedOut.Load() {
			client.releaseHTTPSession(session)
		}
	case <-timer.C:
		session.timedOut.Store(true)
		w.WriteHeader(http.StatusNoContent)
	case <-done:
		session.timedOut.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (client *HTTPClient) writeResponse(w http.ResponseWriter, response []byte) {
	w.Header().Set("Content-Type", CONTENT_TYPE_JSON)
	if _, err := w.Write(response); err != nil {
		client.tracef("failed to write response: %v", err)
	}
}

func (client *HTTPClient) acquireLimitedReader(r io.Reader, limit int64) *io.LimitedReader {
	if client.readerPool == nil {
		return &io.LimitedReader{R: r, N: limit}
	}
	lr := client.readerPool.Get().(*io.LimitedReader)
	lr.R = r
	lr.N = limit
	return lr
}

func (client *HTTPClient) releaseLimitedReader(lr *io.LimitedReader) {
	if client.readerPool != nil {
		lr.R = nil
		client.readerPool.Put(lr)
	}
}

// Verifies incoming request if it's from Discord. Returns the body bytes and buffer if verification was successful.
func (client *HTTPClient) verifyRequest(r *http.Request, key ed25519.PublicKey, maxSize int64) (rawData []byte, buf *bytes.Buffer, ok bool) {
	signature := r.Header.Get("X-Signature-Ed25519")
	if len(signature) != ed25519.SignatureSize*2 {
		return nil, nil, false
	}

	var sig [ed25519.SignatureSize]byte
	if !decodeHex(sig[:], signature) {
		return nil, nil, false
	}

	if sig[ed25519.SignatureSize-1]&224 != 0 {
		return nil, nil, false
	}

	timestamp := r.Header.Get("X-Signature-Timestamp")
	if timestamp == "" {
		return nil, nil, false
	}

	buf = client.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	buf.WriteString(timestamp)

	defer func() {
		if err := r.Body.Close(); err != nil {
			client.tracef("failed to close request body: %v", err)
		}
	}()

	lr := client.acquireLimitedReader(r.Body, maxSize)
	_, err := buf.ReadFrom(lr)
	client.releaseLimitedReader(lr)
	if err != nil {
		client.releaseBuffer(buf)
		return nil, nil, false
	}

	if ed25519.Verify(key, buf.Bytes(), sig[:]) {
		return buf.Bytes()[len(timestamp):], buf, true
	}

	client.releaseBuffer(buf)
	return nil, nil, false
}

func (client *HTTPClient) releaseBuffer(buf *bytes.Buffer) {
	if buf != nil && int64(buf.Cap()) <= MAX_REQUEST_BODY_SIZE {
		client.bufferPool.Put(buf)
	}
}

func decodeHex(dst []byte, src string) bool {
	if len(src) != len(dst)*2 {
		return false
	}

	for i := range dst {
		high, ok := hexNibble(src[i*2])
		if !ok {
			return false
		}

		low, ok := hexNibble(src[i*2+1])
		if !ok {
			return false
		}
		dst[i] = high<<4 | low
	}

	return true
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

func (client *HTTPClient) commandInteractionHandler(interaction *CommandInteraction) {
	command, available := client.handleInteraction(interaction)
	if !available {
		client.tracef("Received command interaction (ID = %s) but there's no matching command! (requested \"%s\")", interaction.ID.String(), interaction.Data.Name)
		interaction.session.responseSent.Store(true)
		select {
		case interaction.session.responseCh <- responsePayload{data: bodyUnknownCommandResponse}:
		default:
		}
		return
	}

	client.tracef("Received command interaction (ID = %s, Command = \"%s\") - moved to target command's handler.", interaction.ID.String(), interaction.Data.Name)

	if client.preCommandHandler != nil && !client.preCommandHandler(command, interaction) {
		return
	}

	if client.trace {
		start := time.Now()
		command.SlashCommandHandler(interaction)
		client.tracef("Command %s execution took %v", command.Name, time.Since(start))
	} else {
		command.SlashCommandHandler(interaction)
	}

	if client.postCommandHandler != nil {
		client.postCommandHandler(command, interaction)
	}
}

func (client *HTTPClient) autoCompleteInteractionHandler(interaction *CommandInteraction) []CommandOptionChoice {
	command, available := client.handleInteraction(interaction)
	if !available || command.AutoCompleteHandler == nil {
		client.tracef("Dropped auto complete interaction (ID = %s). You see this trace message because client received slash command's auto complete interaction but there's no defined handler for it.", interaction.ID.String())
		return nil
	}

	client.tracef("Received slash command's auto complete interaction (ID = %s, Command = \"%s\") - moved to target (sub) command auto complete handler.", interaction.ID.String(), interaction.Data.Name)
	return command.AutoCompleteHandler(*interaction)
}

func (client *HTTPClient) componentInteractionHandler(interaction *ComponentInteraction) {
	if fn, ok := client.staticComponents.Get(interaction.Data.CustomID); ok {
		client.tracef("Received component interaction (ID = %s, CustomID = \"%s\") with matching custom ID for static handler - moved to registered handler.", interaction.ID.String(), interaction.Data.CustomID)
		fn(*interaction)
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
		client.tracef("Received component interaction (ID = %s, CustomID = \"%s\") with matching custom ID for dynamic handler - moved to listener.", interaction.ID.String(), interaction.Data.CustomID)

		interaction.session.responseSent.Store(true)
		select {
		case interaction.session.responseCh <- responsePayload{data: bodyAcknowledgeResponse}:
		default:
		}

		interaction.deferred = true
		handler.Handler(interaction)
		return
	}

	if hasGlobal {
		client.tracef("Received component interaction (ID = %s, CustomID = \"%s\") - moved to defined component handler.", interaction.ID.String(), interaction.Data.CustomID)
		client.componentHandler(interaction)
		return
	}

	client.tracef("Dropped component interaction (ID = %s, CustomID = \"%s\"). You see this trace message because client received component interaction but there's no defined handler for it.", interaction.ID.String(), interaction.Data.CustomID)
}

func (client *HTTPClient) modalInteractionHandler(interaction *ModalInteraction) {
	if fn, ok := client.staticModals.Get(interaction.Data.CustomID); ok {
		client.tracef("Received modal interaction (ID = %s, CustomID = \"%s\") with matching custom ID for static handler - moved to registered handler.", interaction.ID.String(), interaction.Data.CustomID)
		fn(*interaction)
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
		client.tracef("Received modal interaction (ID = %s, CustomID = \"%s\") with matching custom ID for dynamic handler - moved to listener.", interaction.ID.String(), interaction.Data.CustomID)

		interaction.session.responseSent.Store(true)
		select {
		case interaction.session.responseCh <- responsePayload{data: bodyAcknowledgeResponse}:
		default:
		}

		interaction.deferred = true
		handler.Handler(interaction)
		return
	}

	if hasGlobal {
		client.tracef("Received modal interaction (ID = %s, CustomID = \"%s\") - moved to defined modal handler.", interaction.ID.String(), interaction.Data.CustomID)
		client.modalHandler(interaction)
		return
	}

	client.tracef("Dropped modal interaction (ID = %s, CustomID = \"%s\"). You see this trace message because client received modal interaction but there's no defined handler for it.", interaction.ID.String(), interaction.Data.CustomID)
}
