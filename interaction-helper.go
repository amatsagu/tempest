package tempest

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"net/http"

	"github.com/amatsagu/tempest/discord"
)

// Returns whether this interaction already was responded to.
func (itx *interactionContext) Responded() bool {
	return itx.responded
}

func (itx *interactionContext) respond(res *Response) error {
	if itx.HTTPClient != nil && itx.session != nil {
		if err := itx.requestContext().Err(); err != nil {
			return err
		}

		buf := itx.HTTPClient.acquireBuffer()
		if err := json.MarshalWrite(buf, res); err != nil {
			itx.HTTPClient.releaseBuffer(buf)
			return err
		}

		itx.session.responseSent.Store(true)
		select {
		case itx.session.responseCh <- responsePayload{data: buf.Bytes(), buf: buf}:
			return nil
		default:
			itx.HTTPClient.releaseBuffer(buf)
			return errors.New("initial interaction response already sent")
		}
	}

	return errors.New("no client attached to interaction")
}

// Returns the narrowest lifecycle available to this interaction.
// HTTP interactions carry a request-specific context; otherwise fall back to
// the REST client lifecycle context.
func (itx *interactionContext) requestContext() context.Context {
	if itx.ctx != nil {
		return itx.ctx
	}

	if itx.BaseClient != nil {
		return itx.BaseClient.context()
	}

	return context.Background()
}

// Use to let user/member know that bot is processing command.
// Make ephemeral = true to make notification visible only to target.
func (itx *CommandInteraction) Defer(ephemeral bool) error {
	if itx.deferred || itx.responded {
		return errors.New("interaction has already been responded to or deferred")
	}

	var flags discord.MessageFlags = 0
	if ephemeral {
		flags = discord.EphemeralMessageFlag
	}

	err := itx.respond(&Response{
		Type: discord.DeferredChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{Flags: flags},
	})

	if err == nil {
		itx.deferred = true
	}

	return err
}

// Acknowledges the interaction with a message. Set ephemeral = true to make message visible only to target.
func (itx *CommandInteraction) SendReply(reply discord.ResponseMessageData, ephemeral bool, files []File) error {
	if ephemeral {
		reply.Flags |= discord.EphemeralMessageFlag
	}

	if itx.responded {
		return errors.New("interaction has already been responded to")
	}

	endpoint := "/webhooks/" + itx.ApplicationID.String() + "/" + itx.Token

	if itx.deferred {
		err := itx.BaseClient.Rest.RequestWithFilesContext(itx.requestContext(), http.MethodPatch, endpoint+"/messages/@original", reply, files, nil)
		if err == nil {
			itx.responded = true
		}
		return err
	}

	if len(files) > 0 {
		err := itx.respond(&Response{
			Type: discord.DeferredChannelMessageWithSourceResponseType,
			Data: &discord.ResponseMessageData{Flags: reply.Flags},
		})
		if err != nil {
			return err
		}

		itx.deferred = true // Manually set state

		err = itx.BaseClient.Rest.RequestWithFilesContext(itx.requestContext(), http.MethodPatch, endpoint+"/messages/@original", reply, files, nil)
		if err == nil {
			itx.responded = true
		}

		return err
	}

	err := itx.respond(&Response{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &reply,
	})

	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *CommandInteraction) SendLinearReply(content string, ephemeral bool) error {
	return itx.SendReply(discord.ResponseMessageData{
		Content: content,
	}, ephemeral, nil)
}

func (itx *CommandInteraction) SendModal(modal discord.ResponseModalData) error {
	err := itx.respond(&Response{
		Type: discord.ModalResponseType,
		Data: &modal,
	})

	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *CommandInteraction) EditReply(content discord.ResponseMessageData, ephemeral bool, files []File) error {
	if ephemeral {
		content.Flags |= discord.EphemeralMessageFlag
	}

	if len(files) > 0 {
		return itx.BaseClient.Rest.RequestWithFilesContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content, files, nil)
	}

	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content, nil)
}

func (itx *CommandInteraction) EditLinearReply(content string, ephemeral bool) error {
	return itx.EditReply(discord.ResponseMessageData{
		Content: content,
	}, ephemeral, nil)
}

func (itx *CommandInteraction) DeleteReply() error {
	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil, nil)
}

func (itx *CommandInteraction) SendFollowUp(content discord.ResponseMessageData, ephemeral bool) (discord.Message, error) {
	if ephemeral {
		content.Flags |= discord.EphemeralMessageFlag
	}

	var res discord.Message
	err := itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content, &res)
	if err != nil {
		return discord.Message{}, err
	}

	return res, nil
}

func (itx *CommandInteraction) SendLinearFollowUp(content string, ephemeral bool) (discord.Message, error) {
	return itx.SendFollowUp(discord.ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *CommandInteraction) SendFollowUpWithFiles(content discord.ResponseMessageData, ephemeral bool, files []File) (discord.Message, error) {
	if ephemeral {
		content.Flags |= discord.EphemeralMessageFlag
	}

	var res discord.Message
	err := itx.BaseClient.Rest.RequestWithFilesContext(itx.requestContext(), http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content, files, &res)
	if err != nil {
		return discord.Message{}, err
	}

	return res, nil
}

func (itx *CommandInteraction) SendLinearFollowUpWithFiles(content string, ephemeral bool, files []File) (discord.Message, error) {
	return itx.SendFollowUpWithFiles(discord.ResponseMessageData{
		Content: content,
	}, ephemeral, files)
}

func (itx *CommandInteraction) EditFollowUp(messageID discord.MessageID, content discord.ResponseMessageData) error {
	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content, nil)
}

func (itx *CommandInteraction) EditLinearFollowUp(messageID discord.MessageID, content string) error {
	return itx.EditFollowUp(messageID, discord.ResponseMessageData{
		Content: content,
	})
}

func (itx *CommandInteraction) DeleteFollowUp(messageID discord.MessageID) error {
	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), nil, nil)
}

// Sends to discord info that this component was handled successfully without sending anything more.
func (itx *ComponentInteraction) Acknowledge() error {
	return itx.respond(&Response{
		Type: discord.DeferredUpdateMessageResponseType,
	})
}

func (itx *ComponentInteraction) AcknowledgeWithMessage(reply discord.ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		reply.Flags |= discord.EphemeralMessageFlag
	}

	return itx.respond(&Response{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &reply,
	})
}

func (itx *ComponentInteraction) AcknowledgeWithLinearMessage(content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(discord.ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *ComponentInteraction) AcknowledgeWithModal(modal discord.ResponseModalData) error {
	return itx.respond(&Response{
		Type: discord.ModalResponseType,
		Data: &modal,
	})
}

// Acknowledges the interaction by updating the message on which the component was attached.
func (itx *ComponentInteraction) AcknowledgeWithUpdate(reply discord.ResponseMessageData) error {
	return itx.respond(&Response{
		Type: discord.UpdateMessageResponseType,
		Data: &reply,
	})
}

func (itx *ComponentInteraction) AcknowledgeWithLinearUpdate(content string) error {
	return itx.AcknowledgeWithUpdate(discord.ResponseMessageData{
		Content: content,
	})
}

// Edits the original message on which the component was attached.
// This is typically used after itx.Acknowledge() (which defers the message update via [discord.DeferredUpdateMessageResponseType]).
// Passing files = nil or an empty slice is handled properly without multipart encoding.
func (itx *ComponentInteraction) EditOriginal(content discord.ResponseMessageData, files []File) error {
	if len(files) > 0 {
		return itx.BaseClient.Rest.RequestWithFilesContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content, files, nil)
	}

	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content, nil)
}

// Edits the original message on which the component was attached with plain text content.
func (itx *ComponentInteraction) EditLinearOriginal(content string) error {
	return itx.EditOriginal(discord.ResponseMessageData{
		Content: content,
	}, nil)
}

// Deletes the original message on which the component was attached.
func (itx *ComponentInteraction) DeleteOriginal() error {
	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil, nil)
}

// Sends to discord info that this component was handled successfully without sending anything more.
func (itx *ModalInteraction) Acknowledge() error {
	return itx.respond(&Response{
		Type: discord.DeferredUpdateMessageResponseType,
	})
}

func (itx *ModalInteraction) AcknowledgeWithMessage(response discord.ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		response.Flags |= discord.EphemeralMessageFlag
	}

	return itx.respond(&Response{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &response,
	})
}

func (itx *ModalInteraction) AcknowledgeWithLinearMessage(content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(discord.ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *ModalInteraction) AcknowledgeWithModal(modal discord.ResponseModalData) error {
	return itx.respond(&Response{
		Type: discord.ModalResponseType,
		Data: &modal,
	})
}

// Acknowledges the interaction by updating the message on which the component was attached.
func (itx *ModalInteraction) AcknowledgeWithUpdate(response discord.ResponseMessageData) error {
	return itx.respond(&Response{
		Type: discord.UpdateMessageResponseType,
		Data: &response,
	})
}

func (itx *ModalInteraction) AcknowledgeWithLinearUpdate(content string) error {
	return itx.AcknowledgeWithUpdate(discord.ResponseMessageData{
		Content: content,
	})
}

// Edits the original message/response after deferring the modal submission or updating in place.
// Passing files = nil or an empty slice is handled properly without multipart encoding.
func (itx *ModalInteraction) EditOriginal(response discord.ResponseMessageData, files []File) error {
	if len(files) > 0 {
		return itx.BaseClient.Rest.RequestWithFilesContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", response, files, nil)
	}

	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", response, nil)
}

// Edits the original message/response with plain text content.
func (itx *ModalInteraction) EditLinearOriginal(content string) error {
	return itx.EditOriginal(discord.ResponseMessageData{
		Content: content,
	}, nil)
}

// Deletes the original message/response.
func (itx *ModalInteraction) DeleteOriginal() error {
	return itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil, nil)
}

// Used to let user/member know that the bot is processing the modal submission,
// which will close the modal dialog.
// Set ephemeral = true to make notification visible only to the submitter.
func (itx *ModalInteraction) Defer(ephemeral bool) error {
	var flags discord.MessageFlags = 0

	if ephemeral {
		flags = discord.EphemeralMessageFlag
	}

	err := itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", discord.ResponseMessage{
		Type: discord.DeferredChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{
			Flags: flags,
		},
	}, nil)

	if err == nil {
		itx.deferred = true
	}

	return err
}

// Used after defering a modal submission to send a message to the user/member.
// Set ephemeral = true to make message visible only to the submitter.
func (itx *ModalInteraction) SendFollowUp(content discord.ResponseMessageData, ephemeral bool) (discord.Message, error) {
	if ephemeral {
		content.Flags |= discord.EphemeralMessageFlag
	}

	var res discord.Message
	err := itx.BaseClient.Rest.RequestContext(itx.requestContext(), http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content, &res)
	if err != nil {
		return discord.Message{}, err
	}

	return res, nil
}

// Used after defering a modal submission to send a message to the user/member.
// Set ephemeral = true to make message visible only to the submitter.
func (itx *ModalInteraction) SendLinearFollowUp(content string, ephemeral bool) (discord.Message, error) {
	return itx.SendFollowUp(discord.ResponseMessageData{
		Content: content,
	}, ephemeral)
}
