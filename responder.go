package tempest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest/discord"
)

// Errors returned when a handler answers an interaction itself.
var (
	// ErrResponded means the interaction's response was already sent.
	ErrResponded = errors.New("tempest: interaction response was already sent")
	// ErrNoResponder means there is no transport around to answer through.
	ErrNoResponder = errors.New("tempest: no responder is installed for this interaction")
)

const (
	responderNone uint8 = iota
	responderHTTP
	responderGateway
)

const (
	responderOpen uint32 = iota
	responderSent
	responderClosed
)

// responder carries a transport's ability to deliver an interaction response
// while the handler is still running, over the shortest available path. It is
// pooled: the transport acquires one, dispatch installs it for the handler, and
// the transport releases it once the reply's owner is decided.
type responder struct {
	ctx    context.Context
	token  string
	ch     chan dispatchResult
	client *Client
	// firstReplyAt is when the first response was actually delivered, zero until
	// a first reply goes out.
	firstReplyAt time.Time
	response     discord.Response
	id           discord.Snowflake
	appID        discord.AppID
	state        atomic.Uint32
	kind         uint8
	typ          discord.InteractionType
	shard        uint16
}

var responderPool = sync.Pool{New: func() any { return new(responder) }}

func acquireResponder() *responder {
	r := responderPool.Get().(*responder)
	r.state.Store(responderOpen)
	return r
}

func releaseResponder(r *responder) {
	*r = responder{}
	responderPool.Put(r)
}

// Send implements discord.Responder for the raw first-response path used by Defer
// and Respond. The nil receiver is deliberate: a handler that answers early with no
// responder gets an error instead of a panic.
func (r *responder) Send(response discord.Response) error {
	if r == nil {
		return ErrNoResponder
	}

	if !r.state.CompareAndSwap(responderOpen, responderSent) {
		return ErrResponded
	}

	r.response = response

	err := r.deliverFirst(response)
	if err != nil {
		// Let the handler's return value go out instead.
		r.state.CompareAndSwap(responderSent, responderOpen)
		return err
	}
	r.firstReplyAt = time.Now()
	return nil
}

// Reply implements discord.Responder. With no first reply yet it sends the initial
// response over the transport's short path; afterwards it edits the original, so
// Defer-then-Reply replaces the acknowledgement rather than failing. Files are
// uploaded by acknowledging with a deferred response and then editing the original
// with multipart, which is the only shape Discord accepts attachments in.
func (r *responder) Reply(data discord.ResponseMessageData, ephemeral, update bool, files []discord.File) error {
	if r == nil {
		return ErrNoResponder
	}

	if r.typ == discord.InteractionApplicationCommandAutocomplete {
		return errors.New("tempest: autocomplete must answer with choices")
	}

	if !r.state.CompareAndSwap(responderOpen, responderSent) {
		if r.state.Load() != responderSent {
			return ErrResponded
		}
		return r.editResponse(data, files)
	}

	typ := discord.ResponseChannelMessageWithSource
	if update {
		typ = discord.ResponseUpdateMessage
	}

	if ephemeral {
		data.Flags |= discord.MessageFlagEphemeral
	}

	if len(files) > 0 {
		deferredType := discord.ResponseDeferredChannelMessageWithSource
		if update {
			deferredType = discord.ResponseDeferredUpdateMessage
		}

		deferred := discord.Response{Type: deferredType}
		if ephemeral {
			deferred.Data = discord.ResponseMessageData{Flags: discord.MessageFlagEphemeral}
		}

		r.response = deferred
		if err := r.deliverFirst(deferred); err != nil {
			r.state.CompareAndSwap(responderSent, responderOpen)
			return err
		}
		r.firstReplyAt = time.Now()
		return r.editResponse(data, files)
	}

	response := discord.Response{Type: typ, Data: data}
	r.response = response

	if err := r.deliverFirst(response); err != nil {
		r.state.CompareAndSwap(responderSent, responderOpen)
		return err
	}
	r.firstReplyAt = time.Now()
	return nil
}

// Responded reports whether the initial response was already sent.
func (r *responder) Responded() bool { return r.responded() }

// FollowUp sends another message after the initial response.
func (r *responder) FollowUp(data discord.ResponseMessageData, ephemeral bool, files []discord.File) (*discord.Message, error) {
	if r == nil || r.client == nil {
		return nil, ErrNoResponder
	}
	return r.client.followUp(r.ctx, r.appID, r.token, data, ephemeral, files)
}

// EditFollowUp edits a follow-up message.
func (r *responder) EditFollowUp(id discord.MessageID, data discord.ResponseMessageData, files []discord.File) error {
	if r == nil || r.client == nil {
		return ErrNoResponder
	}
	return r.client.editFollowUp(r.ctx, r.appID, r.token, id, data, files)
}

// DeleteFollowUp deletes a follow-up message.
func (r *responder) DeleteFollowUp(id discord.MessageID) error {
	if r == nil || r.client == nil {
		return ErrNoResponder
	}
	return r.client.deleteFollowUp(r.ctx, r.appID, r.token, id)
}

func (r *responder) deliverFirst(response discord.Response) error {
	switch r.kind {
	case responderHTTP:
		r.ch <- dispatchResult{Response: response}
		return nil
	case responderGateway:
		return r.client.answer(r.ctx, r.id, r.token, response)
	default:
		return ErrNoResponder
	}
}

func (r *responder) editResponse(data discord.ResponseMessageData, files []discord.File) error {
	if r.client == nil {
		return ErrNoResponder
	}
	return r.client.editOriginal(r.ctx, r.appID, r.token, data, files...)
}

func (r *responder) responded() bool {
	return r != nil && r.state.Load() == responderSent
}

func (r *responder) firstReplyAfter(start time.Time) time.Duration {
	if r != nil && !r.firstReplyAt.IsZero() {
		return r.firstReplyAt.Sub(start)
	}
	return time.Since(start)
}

func (r *responder) deliver(result dispatchResult) {
	if r == nil {
		return
	}

	if !r.state.CompareAndSwap(responderOpen, responderSent) {
		return
	}
	r.ch <- result
}

func (r *responder) install(itx *discord.Interaction) {
	if r == nil {
		return
	}
	r.id, r.token, r.appID = itx.ID, itx.Token, itx.ApplicationID
	r.typ = itx.Type
	itx.ShardID = r.shard
	itx.SetResponder(r)

	if r.client != nil {
		itx.SetRuntime(r.client)
	}
}

func (r *responder) uninstall(itx *discord.Interaction) {
	if r == nil {
		return
	}
	itx.SetResponder(nil)
	itx.SetRuntime(nil)
}

// close claims the reply for the transport, reporting true when the transport
// should send the handler's outcome.
func (r *responder) close() bool {
	if r == nil {
		return true
	}
	return r.state.CompareAndSwap(responderOpen, responderClosed)
}

// effective is the response the completion hook should report: whatever a
// handler sent, or the fallback the runtime produced.
func (r *responder) effective(returned discord.Response) discord.Response {
	if r.responded() {
		return r.response
	}
	return returned
}
