package discord

import "errors"

// Responder delivers interaction responses; a runtime installs one on the
// interaction before the handler runs, so it can answer through the interaction.
type Responder interface {
	Send(Response) error
	Reply(data ResponseMessageData, ephemeral, update bool, files []File) error
	FollowUp(data ResponseMessageData, ephemeral bool, files []File) (*Message, error)
	EditFollowUp(id MessageID, data ResponseMessageData, files []File) error
	DeleteFollowUp(id MessageID) error
	Responded() bool
}

var errNoResponder = errors.New("discord: no responder installed; this interaction is not running under a runtime")

var errAutocomplete = errors.New("discord: autocomplete must answer with choices and cannot be deferred or replied to")

// SetResponder installs the responder used by Defer, Reply, Update and Respond; it
// is cleared after the handler so nothing can be sent once the interaction is over.
func (i *Interaction) SetResponder(r Responder) { i.responder = r }

// Responded reports whether the initial response was already sent.
func (i *Interaction) Responded() bool {
	return i.responder != nil && i.responder.Responded()
}

// BaseUser returns the invoking user, taken from the member when the interaction
// happened in a server.
func (i *Interaction) BaseUser() *User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

// Defer acknowledges the interaction without a message, buying time for slow work;
// it goes out immediately. Autocomplete cannot be deferred and reports an error.
func (i *Interaction) Defer(ephemeral bool) error {
	if i.responder == nil {
		return errNoResponder
	}

	if i.Type == InteractionApplicationCommandAutocomplete {
		return errAutocomplete
	}

	response := Response{Type: ResponseDeferredUpdateMessage}
	if i.Type == InteractionApplicationCommand {
		response.Type = ResponseDeferredChannelMessageWithSource
	}

	if ephemeral {
		response.Data = ResponseMessageData{Flags: MessageFlagEphemeral}
	}

	return i.responder.Send(response)
}

// Reply answers with a message: the first response before anything has been sent,
// or an edit of the original afterwards. Files turn it into a deferred response
// followed by an edit, because Discord only accepts attachment bytes as multipart.
func (i *Interaction) Reply(data ResponseMessageData, ephemeral bool, files ...File) error {
	if i.responder == nil {
		return errNoResponder
	}
	return i.responder.Reply(data, ephemeral, false, files)
}

// Update edits the component's own message as the first response, or the original
// afterwards.
func (i *Interaction) Update(data ResponseMessageData, files ...File) error {
	if i.responder == nil {
		return errNoResponder
	}
	return i.responder.Reply(data, false, true, files)
}

// Respond sends an arbitrary response immediately; unlike Reply it never swaps to
// an edit, so a second Respond reports ErrResponded.
func (i *Interaction) Respond(r Response) error {
	if i.responder == nil {
		return errNoResponder
	}
	return i.responder.Send(r)
}

// FollowUp sends another message after the initial response, with files when given.
func (i *Interaction) FollowUp(data ResponseMessageData, ephemeral bool, files ...File) (*Message, error) {
	if i.responder == nil {
		return nil, errNoResponder
	}
	return i.responder.FollowUp(data, ephemeral, files)
}

// EditFollowUp edits a follow-up message previously returned by FollowUp.
func (i *Interaction) EditFollowUp(id MessageID, data ResponseMessageData, files ...File) error {
	if i.responder == nil {
		return errNoResponder
	}
	return i.responder.EditFollowUp(id, data, files)
}

// DeleteFollowUp deletes a follow-up message previously returned by FollowUp.
func (i *Interaction) DeleteFollowUp(id MessageID) error {
	if i.responder == nil {
		return errNoResponder
	}
	return i.responder.DeleteFollowUp(id)
}
