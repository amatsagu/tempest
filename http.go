package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest/discord"
)

const (
	sessionPending uint32 = iota
	sessionResponded
	sessionTimedOut
)

const semaphoreWaitBudget = 250 * time.Millisecond

var (
	bodyPingResponse           = fmt.Appendf(nil, `{"type":%d}`, discord.PongResponseType)
	bodyUnknownCommandResponse = fmt.Appendf(nil, `{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.","flags":%d}}`, discord.ChannelMessageWithSourceResponseType, discord.EphemeralMessageFlag)

	emptyReader    = bytes.NewReader(nil)
	jsonReaderPool = sync.Pool{
		New: func() any {
			return bytes.NewReader(nil)
		},
	}
	jsonDecoderPool = sync.Pool{
		New: func() any {
			return jsontext.NewDecoder(emptyReader)
		},
	}

	errInvalidJSONObject = errors.New("expected json object")
	errFieldNotFound     = errors.New("type field not found")
)

type responsePayload struct {
	buf  *bytes.Buffer
	data []byte
}

type httpSession struct {
	client     *Client
	responseCh chan responsePayload
	state      atomic.Uint32
	refs       atomic.Int32
}

func (c *Client) acquireSession() *httpSession {
	s := c.sessionPool.Get().(*httpSession)
	s.client = c
	s.state.Store(sessionPending)
	s.refs.Store(2)

	select {
	case old := <-s.responseCh:
		if old.buf != nil {
			c.releaseBuffer(old.buf)
		}
	default:
	}

	return s
}

func (s *httpSession) releaseRef() {
	if s.refs.Add(-1) != 0 {
		return
	}

	select {
	case leftover := <-s.responseCh:
		if leftover.buf != nil {
			s.client.releaseBuffer(leftover.buf)
		}
	default:
	}

	c := s.client
	s.client = nil
	if c != nil && c.sessionPool != nil {
		c.sessionPool.Put(s)
	}
}

func (s *httpSession) tryRespond(data []byte, buf *bytes.Buffer) bool {
	if !s.state.CompareAndSwap(sessionPending, sessionResponded) {
		return false
	}

	s.responseCh <- responsePayload{data: data, buf: buf}
	return true
}

func (s *httpSession) finishEmpty() {
	if s.state.CompareAndSwap(sessionPending, sessionResponded) {
		s.responseCh <- responsePayload{}
	}
}

func (s *httpSession) hasResponded() bool {
	return s.state.Load() == sessionResponded
}

type mergedValueCtx struct {
	context.Context
	vals context.Context
}

func (m mergedValueCtx) Value(key any) any {
	if v := m.vals.Value(key); v != nil {
		return v
	}
	return m.Context.Value(key)
}

func (c *Client) interactionContext(rCtx context.Context, s *httpSession) (context.Context, context.CancelFunc, func()) {
	parent := c.baseCtx
	if rCtx != nil && rCtx != context.Background() && rCtx != context.TODO() {
		parent = mergedValueCtx{Context: c.baseCtx, vals: rCtx}
	}

	ctx, cancel := context.WithCancel(parent)
	if rCtx == nil || rCtx.Done() == nil {
		return ctx, cancel, nil
	}

	stop := context.AfterFunc(rCtx, func() {
		if !s.hasResponded() {
			cancel()
		}
	})

	return ctx, cancel, func() { stop() }
}

func (c *Client) acquireTimer(d time.Duration) *time.Timer {
	if c.timerPool == nil {
		return time.NewTimer(d)
	}

	t := c.timerPool.Get().(*time.Timer)
	t.Reset(d)
	return t
}

func (c *Client) releaseTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}

	if c.timerPool != nil {
		c.timerPool.Put(t)
	}
}

func (c *Client) acquireLimitedReader(r io.Reader, limit int64) *io.LimitedReader {
	if c.readerPool == nil {
		return &io.LimitedReader{R: r, N: limit}
	}

	lr := c.readerPool.Get().(*io.LimitedReader)
	lr.R = r
	lr.N = limit
	return lr
}

func (c *Client) releaseLimitedReader(lr *io.LimitedReader) {
	if c.readerPool != nil {
		lr.R = nil
		c.readerPool.Put(lr)
	}
}

func (c *Client) acquireBuffer() *bytes.Buffer {
	if c.bufferPool == nil {
		b := new(bytes.Buffer)
		b.Grow(32 * 1024)
		return b
	}

	buf := c.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (c *Client) releaseBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	if c.bufferPool != nil && buf.Cap() <= maxPooledBodyBufferCap {
		buf.Reset()
		c.bufferPool.Put(buf)
	}
}

// Handles Discord HTTP interaction webhooks with Ed25519 signature verification
// and inline initial response writing. Implements [http.Handler].
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.closed.Load() {
		http.Error(w, "service shutting down", http.StatusServiceUnavailable)
		return
	}

	rawData, buf, verified := c.verifyRequest(r, c.PublicKey, maxRequestBodySize)
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer c.releaseBuffer(buf)

	itxType, err := peekInteractionType(rawData)
	if err != nil {
		c.tracef("Failed to peek interaction type: %v", err)
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	switch itxType {
	case discord.PingInteractionType:
		c.writeJSON(w, bodyPingResponse)

	case discord.ApplicationCommandInteractionType:
		itx := new(CommandInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			c.tracef("Failed to unmarshal command interaction: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		itx.Client = c
		c.serveAsyncHTTP(w, r, &itx.interactionContext, func(ctx context.Context) {
			_ = c.dispatchCommand(ctx, itx)
		})

	case discord.MessageComponentInteractionType:
		itx := new(ComponentInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			c.tracef("Failed to unmarshal component interaction: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		itx.Client = c
		c.serveAsyncHTTP(w, r, &itx.interactionContext, func(ctx context.Context) {
			_ = c.dispatchComponent(ctx, itx)
		})

	case discord.ApplicationCommandAutoCompleteInteractionType:
		itx := new(CommandInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			c.tracef("Failed to unmarshal autocomplete interaction: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		itx.Client = c
		c.serveAutocompleteHTTP(w, r.Context(), itx)

	case discord.ModalSubmitInteractionType:
		itx := new(ModalInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			c.tracef("Failed to unmarshal modal interaction: %v", err)
			http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
			return
		}

		itx.Client = c
		c.serveAsyncHTTP(w, r, &itx.interactionContext, func(ctx context.Context) {
			_ = c.dispatchModal(ctx, itx)
		})

	default:
		http.Error(w, "bad request - unsupported interaction type", http.StatusBadRequest)
	}
}

// Processes a raw Discord INTERACTION_CREATE JSON payload without Ed25519 signature checks.
// Intended for external Gateway addons; initial responses are sent via [Client.Rest].
func (c *Client) HandleInteraction(ctx context.Context, rawData []byte) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	if ctx == nil {
		ctx = c.baseCtx
	}

	itxType, err := peekInteractionType(rawData)
	if err != nil {
		return fmt.Errorf("failed to inspect interaction payload: %w", err)
	}

	switch itxType {
	case discord.PingInteractionType:
		return nil

	case discord.ApplicationCommandInteractionType:
		itx := new(CommandInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			return fmt.Errorf("failed to unmarshal command interaction: %w", err)
		}

		itx.Client = c
		if err := c.acquireSlot(ctx); err != nil {
			return err
		}

		c.wg.Add(1)
		defer c.wg.Done()
		defer c.releaseSlot()
		return c.dispatchCommand(ctx, itx)

	case discord.MessageComponentInteractionType:
		itx := new(ComponentInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			return fmt.Errorf("failed to unmarshal component interaction: %w", err)
		}

		itx.Client = c
		if err := c.acquireSlot(ctx); err != nil {
			return err
		}

		c.wg.Add(1)
		defer c.wg.Done()
		defer c.releaseSlot()
		return c.dispatchComponent(ctx, itx)

	case discord.ApplicationCommandAutoCompleteInteractionType:
		itx := new(CommandInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			return fmt.Errorf("failed to unmarshal autocomplete interaction: %w", err)
		}

		itx.Client = c
		if err := c.acquireSlot(ctx); err != nil {
			return err
		}

		c.wg.Add(1)
		defer c.wg.Done()
		defer c.releaseSlot()

		choices, err := c.runAutoComplete(ctx, itx)
		if err != nil {
			return err
		}

		res := discord.ResponseAutoComplete{
			Type: discord.AutocompleteResponseType,
			Data: &discord.ResponseAutoCompleteData{Choices: choices},
		}
		return c.Rest.Request(ctx, http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", &res, nil)

	case discord.ModalSubmitInteractionType:
		itx := new(ModalInteraction)
		if err := json.Unmarshal(rawData, itx); err != nil {
			return fmt.Errorf("failed to unmarshal modal interaction: %w", err)
		}

		itx.Client = c
		if err := c.acquireSlot(ctx); err != nil {
			return err
		}

		c.wg.Add(1)
		defer c.wg.Done()
		defer c.releaseSlot()
		return c.dispatchModal(ctx, itx)

	default:
		return fmt.Errorf("unsupported interaction type %d", itxType)
	}
}

func (c *Client) serveAsyncHTTP(w http.ResponseWriter, r *http.Request, ic *interactionContext, fn func(context.Context)) {
	if !c.tryAcquireSlotHTTP(r.Context()) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	session := c.acquireSession()
	ic.session = session

	handlerCtx, cancel, stop := c.interactionContext(r.Context(), session)

	c.wg.Go(func() {
		defer c.releaseSlot()
		defer session.releaseRef()
		defer session.finishEmpty()
		defer cancel()
		if stop != nil {
			defer stop()
		}

		fn(handlerCtx)
	})

	c.awaitResponse(r.Context(), w, session)
}

func (c *Client) serveAutocompleteHTTP(w http.ResponseWriter, reqCtx context.Context, itx *CommandInteraction) {
	if !c.tryAcquireSlotHTTP(reqCtx) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	c.wg.Add(1)
	defer c.wg.Done()
	defer c.releaseSlot()

	ctx := reqCtx
	if ctx == nil || ctx == context.Background() || ctx == context.TODO() {
		ctx = c.baseCtx
	} else {
		ctx = mergedValueCtx{Context: c.baseCtx, vals: reqCtx}
	}

	choices, err := c.runAutoComplete(ctx, itx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	buf := c.acquireBuffer()
	defer c.releaseBuffer(buf)

	res := discord.ResponseAutoComplete{
		Type: discord.AutocompleteResponseType,
		Data: &discord.ResponseAutoCompleteData{Choices: choices},
	}
	if err := json.MarshalWrite(buf, &res); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	c.writeJSON(w, buf.Bytes())
}

func (c *Client) writeSessionPayload(w http.ResponseWriter, payload responsePayload) {
	if len(payload.data) > 0 {
		c.writeJSON(w, payload.data)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}

	if payload.buf != nil {
		c.releaseBuffer(payload.buf)
	}
}

func (c *Client) awaitResponse(reqCtx context.Context, w http.ResponseWriter, session *httpSession) {
	defer session.releaseRef()

	timer := c.acquireTimer(c.ackTimeout)
	defer c.releaseTimer(timer)

	var done <-chan struct{}
	if reqCtx != nil {
		done = reqCtx.Done()
	}

	select {
	case payload := <-session.responseCh:
		c.writeSessionPayload(w, payload)

	case <-timer.C:
		if session.state.CompareAndSwap(sessionPending, sessionTimedOut) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		c.writeSessionPayload(w, <-session.responseCh)

	case <-done:
		if session.state.CompareAndSwap(sessionPending, sessionTimedOut) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		c.writeSessionPayload(w, <-session.responseCh)
	}
}

func (c *Client) acquireSlot(ctx context.Context) error {
	if c.sem == nil {
		return nil
	}

	select {
	case c.sem <- struct{}{}:
		return nil
	default:
	}

	select {
	case c.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.baseCtx.Done():
		return ErrClientClosed
	}
}

func (c *Client) tryAcquireSlotHTTP(reqCtx context.Context) bool {
	if c.sem == nil {
		return true
	}

	select {
	case c.sem <- struct{}{}:
		return true
	default:
	}

	timer := c.acquireTimer(semaphoreWaitBudget)
	defer c.releaseTimer(timer)

	var done <-chan struct{}
	if reqCtx != nil {
		done = reqCtx.Done()
	}

	select {
	case c.sem <- struct{}{}:
		return true
	case <-timer.C:
		return false
	case <-done:
		return false
	case <-c.baseCtx.Done():
		return false
	}
}

func (c *Client) releaseSlot() {
	if c.sem != nil {
		<-c.sem
	}
}

func (c *Client) dispatchCommand(ctx context.Context, itx *CommandInteraction) error {
	cmd, ok := c.resolveCommand(itx)
	if !ok {
		c.tracef("Unknown command %q (interaction ID = %s).", itx.Data.Name, itx.ID.String())
		if itx.session != nil {
			itx.session.tryRespond(bodyUnknownCommandResponse, nil)
		} else {
			_ = c.Rest.Request(ctx, http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", &discord.ResponseMessage{
				Type: discord.ChannelMessageWithSourceResponseType,
				Data: &discord.ResponseMessageData{
					Content: "Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.",
					Flags:   discord.EphemeralMessageFlag,
				},
			}, nil)
		}
		return fmt.Errorf("%w: %s", ErrUnknownCommand, itx.Data.Name)
	}

	if cmd.SlashCommandHandler == nil {
		return fmt.Errorf("command %q has nil SlashCommandHandler", itx.Data.Name)
	}

	c.tracef("Dispatching command %q (interaction ID = %s).", itx.Data.Name, itx.ID.String())

	if c.hooks.PreCommand != nil && !c.hooks.PreCommand(ctx, cmd, itx) {
		return nil
	}

	err := invokeSafe(func() error {
		if c.trace {
			start := time.Now()
			defer func() {
				c.tracef("Command %q completed in %s.", cmd.Name, time.Since(start))
			}()
		}
		return cmd.SlashCommandHandler(ctx, itx)
	})
	if err != nil {
		c.tracef("Command %q failed (interaction ID = %s): %v", itx.Data.Name, itx.ID.String(), err)
		if c.hooks.CommandError != nil {
			c.hooks.CommandError(ctx, cmd, itx, err)
		}
	}

	if c.hooks.PostCommand != nil {
		c.hooks.PostCommand(ctx, cmd, itx)
	}

	return err
}

func (c *Client) runAutoComplete(ctx context.Context, itx *CommandInteraction) ([]discord.CommandOptionChoice, error) {
	cmd, ok := c.resolveCommand(itx)
	if !ok || cmd.AutoCompleteHandler == nil {
		c.tracef("Dropped autocomplete for %q (interaction ID = %s).", itx.Data.Name, itx.ID.String())
		return nil, nil
	}

	c.tracef("Dispatching autocomplete for %q (interaction ID = %s).", itx.Data.Name, itx.ID.String())

	var choices []discord.CommandOptionChoice
	err := invokeSafe(func() error {
		var handlerErr error
		choices, handlerErr = cmd.AutoCompleteHandler(ctx, itx)
		return handlerErr
	})
	if err != nil {
		c.tracef("Autocomplete %q failed (interaction ID = %s): %v", itx.Data.Name, itx.ID.String(), err)
		if c.hooks.CommandError != nil {
			c.hooks.CommandError(ctx, cmd, itx, err)
		}
	}

	return choices, err
}

func (c *Client) dispatchComponent(ctx context.Context, itx *ComponentInteraction) error {
	handler, params, extra, n, ok := c.components.match(itx.Data.CustomID)
	if !ok {
		c.tracef("Dropped unhandled component %q (interaction ID = %s).", itx.Data.CustomID, itx.ID.String())
		return fmt.Errorf("%w: %s", ErrUnknownComponent, itx.Data.CustomID)
	}

	itx.params = params
	itx.extraParams = extra
	itx.paramCount = n

	c.tracef("Dispatching component %q (interaction ID = %s).", itx.Data.CustomID, itx.ID.String())

	if c.hooks.PreComponent != nil && !c.hooks.PreComponent(ctx, itx) {
		return nil
	}

	err := invokeSafe(func() error {
		return handler(ctx, itx)
	})
	if err != nil {
		c.tracef("Component %q failed (interaction ID = %s): %v", itx.Data.CustomID, itx.ID.String(), err)
		if c.hooks.ComponentError != nil {
			c.hooks.ComponentError(ctx, itx, err)
		}
	}

	if c.hooks.PostComponent != nil {
		c.hooks.PostComponent(ctx, itx)
	}

	return err
}

func (c *Client) dispatchModal(ctx context.Context, itx *ModalInteraction) error {
	handler, params, extra, n, ok := c.modals.match(itx.Data.CustomID)
	if !ok {
		c.tracef("Dropped unhandled modal %q (interaction ID = %s).", itx.Data.CustomID, itx.ID.String())
		return fmt.Errorf("%w: %s", ErrUnknownModal, itx.Data.CustomID)
	}

	itx.params = params
	itx.extraParams = extra
	itx.paramCount = n

	c.tracef("Dispatching modal %q (interaction ID = %s).", itx.Data.CustomID, itx.ID.String())

	if c.hooks.PreModal != nil && !c.hooks.PreModal(ctx, itx) {
		return nil
	}

	err := invokeSafe(func() error {
		return handler(ctx, itx)
	})
	if err != nil {
		c.tracef("Modal %q failed (interaction ID = %s): %v", itx.Data.CustomID, itx.ID.String(), err)
		if c.hooks.ModalError != nil {
			c.hooks.ModalError(ctx, itx, err)
		}
	}

	if c.hooks.PostModal != nil {
		c.hooks.PostModal(ctx, itx)
	}

	return err
}

func invokeSafe(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if rErr, ok := r.(error); ok {
				err = fmt.Errorf("handler panic: %w", rErr)
			} else {
				err = fmt.Errorf("handler panic: %v", r)
			}
		}
	}()

	return fn()
}

func (c *Client) writeJSON(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	if _, err := w.Write(data); err != nil {
		c.tracef("Failed to write response: %v", err)
	}
}

func (c *Client) verifyRequest(r *http.Request, key ed25519.PublicKey, maxSize int64) (rawData []byte, buf *bytes.Buffer, ok bool) {
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

	buf = c.acquireBuffer()
	buf.WriteString(timestamp)

	defer func() {
		if err := r.Body.Close(); err != nil {
			c.tracef("Failed to close request body: %v", err)
		}
	}()

	lr := c.acquireLimitedReader(r.Body, maxSize)
	_, err := buf.ReadFrom(lr)
	c.releaseLimitedReader(lr)
	if err != nil {
		c.releaseBuffer(buf)
		return nil, nil, false
	}

	if ed25519.Verify(key, buf.Bytes(), sig[:]) {
		return buf.Bytes()[len(timestamp):], buf, true
	}

	c.releaseBuffer(buf)
	return nil, nil, false
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

func peekInteractionType(data []byte) (discord.InteractionType, error) {
	r := jsonReaderPool.Get().(*bytes.Reader)
	r.Reset(data)
	dec := jsonDecoderPool.Get().(*jsontext.Decoder)
	dec.Reset(r)

	defer func() {
		jsonReaderPool.Put(r)
		jsonDecoderPool.Put(dec)
	}()

	tok, err := dec.ReadToken()
	if err != nil || tok.Kind() != '{' {
		return 0, errInvalidJSONObject
	}

	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return 0, err
		}

		if keyTok.Kind() == '"' && keyTok.String() == "type" {
			valTok, err := dec.ReadToken()
			if err != nil {
				return 0, err
			}
			if valTok.Kind() != '0' {
				return 0, errInvalidJSONObject
			}
			u, err := valTok.Uint()
			if err != nil {
				return 0, err
			}
			return discord.InteractionType(u), nil
		}

		if err := dec.SkipValue(); err != nil {
			return 0, err
		}
	}

	return 0, errFieldNotFound
}
