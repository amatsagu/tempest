package tempest

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"net/http"

	"github.com/amatsagu/tempest/discord"
)

// Reports whether an initial response has already been sent for this interaction.
func (ic *interactionContext) Responded() bool {
	return ic.responded
}

func (ic *interactionContext) param(name string) (string, bool) {
	inline := min(int(ic.paramCount), maxInlinePatternParams)
	for i := range inline {
		if ic.params[i].name == name {
			return ic.params[i].value, true
		}
	}

	for _, p := range ic.extraParams {
		if p.name == name {
			return p.value, true
		}
	}

	return "", false
}

// Returns the matched wildcard value from the registered custom_id pattern (or "" if absent).
func (itx *ComponentInteraction) PathValue(name string) string {
	v, _ := itx.param(name)
	return v
}

// Returns the matched wildcard value from the registered custom_id pattern and whether it exists.
func (itx *ComponentInteraction) Param(name string) (string, bool) {
	return itx.param(name)
}

// Returns the matched wildcard value from the registered custom_id pattern (or "" if absent).
func (itx *ModalInteraction) PathValue(name string) string {
	v, _ := itx.param(name)
	return v
}

// Returns the matched wildcard value from the registered custom_id pattern and whether it exists.
func (itx *ModalInteraction) Param(name string) (string, bool) {
	return itx.param(name)
}

func (ic *interactionContext) respond(ctx context.Context, itx *discord.Interaction, res *Response) error {
	if ic.Client == nil {
		return errors.New("no client attached to interaction")
	}

	if ctx == nil {
		ctx = ic.Client.baseCtx
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Fast HTTP inline response path: writes directly back on the webhook HTTP request.
	if ic.session != nil && len(res.Files) == 0 {
		buf := ic.Client.acquireBuffer()
		if err := json.MarshalWrite(buf, res); err != nil {
			ic.Client.releaseBuffer(buf)
			return err
		}

		if ic.session.tryRespond(buf.Bytes(), buf) {
			return nil
		}

		ic.Client.releaseBuffer(buf)
		if ic.session.state.Load() == sessionTimedOut {
			return ErrAckTimeout
		}
		return ErrAlreadyResponded
	}

	// Gateway (no HTTP session) sends the initial response via Discord's interaction callback endpoint.
	endpoint := "/interactions/" + itx.ID.String() + "/" + itx.Token + "/callback"
	if len(res.Files) > 0 {
		return ic.Client.Rest.RequestWithFiles(ctx, http.MethodPost, endpoint, res, res.Files, nil)
	}

	return ic.Client.Rest.Request(ctx, http.MethodPost, endpoint, res, nil)
}

func (ic *interactionContext) webhookBase(itx *discord.Interaction) string {
	return "/webhooks/" + itx.ApplicationID.String() + "/" + itx.Token
}

func (ic *interactionContext) editOriginal(ctx context.Context, itx *discord.Interaction, content discord.ResponseMessageData, files []File) error {
	endpoint := ic.webhookBase(itx) + "/messages/@original"
	if len(files) > 0 {
		return ic.Client.Rest.RequestWithFiles(ctx, http.MethodPatch, endpoint, content, files, nil)
	}

	return ic.Client.Rest.Request(ctx, http.MethodPatch, endpoint, content, nil)
}

func (ic *interactionContext) deleteOriginal(ctx context.Context, itx *discord.Interaction) error {
	return ic.Client.Rest.Request(ctx, http.MethodDelete, ic.webhookBase(itx)+"/messages/@original", nil, nil)
}

func (ic *interactionContext) sendFollowUp(ctx context.Context, itx *discord.Interaction, content discord.ResponseMessageData, ephemeral bool, files []File) (discord.Message, error) {
	if ephemeral {
		content.Flags |= discord.EphemeralMessageFlag
	}

	var res discord.Message
	var err error
	if len(files) > 0 {
		err = ic.Client.Rest.RequestWithFiles(ctx, http.MethodPost, ic.webhookBase(itx), content, files, &res)
	} else {
		err = ic.Client.Rest.Request(ctx, http.MethodPost, ic.webhookBase(itx), content, &res)
	}
	if err != nil {
		return discord.Message{}, err
	}

	return res, nil
}

func (ic *interactionContext) editFollowUp(ctx context.Context, itx *discord.Interaction, messageID discord.MessageID, content discord.ResponseMessageData) error {
	return ic.Client.Rest.Request(ctx, http.MethodPatch, ic.webhookBase(itx)+"/messages/"+messageID.String(), content, nil)
}

func (ic *interactionContext) deleteFollowUp(ctx context.Context, itx *discord.Interaction, messageID discord.MessageID) error {
	return ic.Client.Rest.Request(ctx, http.MethodDelete, ic.webhookBase(itx)+"/messages/"+messageID.String(), nil, nil)
}

// --- CommandInteraction response helpers ---

func (itx *CommandInteraction) Defer(ctx context.Context, ephemeral bool) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	var flags discord.MessageFlags
	if ephemeral {
		flags = discord.EphemeralMessageFlag
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.DeferredChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{Flags: flags},
	})
	if err == nil {
		itx.deferred = true
	}

	return err
}

func (itx *CommandInteraction) SendReply(ctx context.Context, reply discord.ResponseMessageData, ephemeral bool, files []File) error {
	if ephemeral {
		reply.Flags |= discord.EphemeralMessageFlag
	}
	if itx.responded {
		return ErrAlreadyResponded
	}

	if itx.deferred {
		err := itx.editOriginal(ctx, &itx.Interaction, reply, files)
		if err == nil {
			itx.responded = true
		}
		return err
	}

	if len(files) > 0 && itx.session != nil {
		if err := itx.respond(ctx, &itx.Interaction, &Response{
			Type: discord.DeferredChannelMessageWithSourceResponseType,
			Data: &discord.ResponseMessageData{Flags: reply.Flags},
		}); err != nil {
			return err
		}

		itx.deferred = true
		err := itx.editOriginal(ctx, &itx.Interaction, reply, files)
		if err == nil {
			itx.responded = true
		}
		return err
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type:  discord.ChannelMessageWithSourceResponseType,
		Data:  &reply,
		Files: files,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *CommandInteraction) SendLinearReply(ctx context.Context, content string, ephemeral bool) error {
	return itx.SendReply(ctx, discord.ResponseMessageData{Content: content}, ephemeral, nil)
}

func (itx *CommandInteraction) SendModal(ctx context.Context, modal discord.ResponseModalData) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.ModalResponseType,
		Data: &modal,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *CommandInteraction) EditReply(ctx context.Context, content discord.ResponseMessageData, ephemeral bool, files []File) error {
	if ephemeral {
		content.Flags |= discord.EphemeralMessageFlag
	}

	return itx.editOriginal(ctx, &itx.Interaction, content, files)
}

func (itx *CommandInteraction) EditLinearReply(ctx context.Context, content string, ephemeral bool) error {
	return itx.EditReply(ctx, discord.ResponseMessageData{Content: content}, ephemeral, nil)
}

func (itx *CommandInteraction) DeleteReply(ctx context.Context) error {
	return itx.deleteOriginal(ctx, &itx.Interaction)
}

func (itx *CommandInteraction) SendFollowUp(ctx context.Context, content discord.ResponseMessageData, ephemeral bool) (discord.Message, error) {
	return itx.sendFollowUp(ctx, &itx.Interaction, content, ephemeral, nil)
}

func (itx *CommandInteraction) SendLinearFollowUp(ctx context.Context, content string, ephemeral bool) (discord.Message, error) {
	return itx.SendFollowUp(ctx, discord.ResponseMessageData{Content: content}, ephemeral)
}

func (itx *CommandInteraction) SendFollowUpWithFiles(ctx context.Context, content discord.ResponseMessageData, ephemeral bool, files []File) (discord.Message, error) {
	return itx.sendFollowUp(ctx, &itx.Interaction, content, ephemeral, files)
}

func (itx *CommandInteraction) SendLinearFollowUpWithFiles(ctx context.Context, content string, ephemeral bool, files []File) (discord.Message, error) {
	return itx.SendFollowUpWithFiles(ctx, discord.ResponseMessageData{Content: content}, ephemeral, files)
}

func (itx *CommandInteraction) EditFollowUp(ctx context.Context, messageID discord.MessageID, content discord.ResponseMessageData) error {
	return itx.editFollowUp(ctx, &itx.Interaction, messageID, content)
}

func (itx *CommandInteraction) EditLinearFollowUp(ctx context.Context, messageID discord.MessageID, content string) error {
	return itx.EditFollowUp(ctx, messageID, discord.ResponseMessageData{Content: content})
}

func (itx *CommandInteraction) DeleteFollowUp(ctx context.Context, messageID discord.MessageID) error {
	return itx.deleteFollowUp(ctx, &itx.Interaction, messageID)
}

// --- ComponentInteraction response helpers ---

func (itx *ComponentInteraction) Acknowledge(ctx context.Context) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.DeferredUpdateMessageResponseType,
	})
	if err == nil {
		itx.deferred = true
	}

	return err
}

func (itx *ComponentInteraction) Defer(ctx context.Context, ephemeral bool) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	var flags discord.MessageFlags
	if ephemeral {
		flags = discord.EphemeralMessageFlag
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.DeferredChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{Flags: flags},
	})
	if err == nil {
		itx.deferred = true
	}

	return err
}

func (itx *ComponentInteraction) AcknowledgeWithMessage(ctx context.Context, reply discord.ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		reply.Flags |= discord.EphemeralMessageFlag
	}
	if itx.responded {
		return ErrAlreadyResponded
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &reply,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *ComponentInteraction) AcknowledgeWithLinearMessage(ctx context.Context, content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(ctx, discord.ResponseMessageData{Content: content}, ephemeral)
}

func (itx *ComponentInteraction) AcknowledgeWithModal(ctx context.Context, modal discord.ResponseModalData) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.ModalResponseType,
		Data: &modal,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *ComponentInteraction) AcknowledgeWithUpdate(ctx context.Context, reply discord.ResponseMessageData) error {
	if itx.responded {
		return ErrAlreadyResponded
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.UpdateMessageResponseType,
		Data: &reply,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *ComponentInteraction) AcknowledgeWithLinearUpdate(ctx context.Context, content string) error {
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{Content: content})
}

func (itx *ComponentInteraction) EditOriginal(ctx context.Context, content discord.ResponseMessageData, files []File) error {
	return itx.editOriginal(ctx, &itx.Interaction, content, files)
}

func (itx *ComponentInteraction) EditLinearOriginal(ctx context.Context, content string) error {
	return itx.EditOriginal(ctx, discord.ResponseMessageData{Content: content}, nil)
}

func (itx *ComponentInteraction) DeleteOriginal(ctx context.Context) error {
	return itx.deleteOriginal(ctx, &itx.Interaction)
}

func (itx *ComponentInteraction) SendFollowUp(ctx context.Context, content discord.ResponseMessageData, ephemeral bool) (discord.Message, error) {
	return itx.sendFollowUp(ctx, &itx.Interaction, content, ephemeral, nil)
}

func (itx *ComponentInteraction) SendLinearFollowUp(ctx context.Context, content string, ephemeral bool) (discord.Message, error) {
	return itx.SendFollowUp(ctx, discord.ResponseMessageData{Content: content}, ephemeral)
}

func (itx *ComponentInteraction) SendFollowUpWithFiles(ctx context.Context, content discord.ResponseMessageData, ephemeral bool, files []File) (discord.Message, error) {
	return itx.sendFollowUp(ctx, &itx.Interaction, content, ephemeral, files)
}

func (itx *ComponentInteraction) EditFollowUp(ctx context.Context, messageID discord.MessageID, content discord.ResponseMessageData) error {
	return itx.editFollowUp(ctx, &itx.Interaction, messageID, content)
}

func (itx *ComponentInteraction) DeleteFollowUp(ctx context.Context, messageID discord.MessageID) error {
	return itx.deleteFollowUp(ctx, &itx.Interaction, messageID)
}

// --- ModalInteraction response helpers ---

func (itx *ModalInteraction) Acknowledge(ctx context.Context) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.DeferredUpdateMessageResponseType,
	})
	if err == nil {
		itx.deferred = true
	}

	return err
}

func (itx *ModalInteraction) Defer(ctx context.Context, ephemeral bool) error {
	if itx.deferred || itx.responded {
		return ErrAlreadyDeferred
	}

	var flags discord.MessageFlags
	if ephemeral {
		flags = discord.EphemeralMessageFlag
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.DeferredChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{Flags: flags},
	})
	if err == nil {
		itx.deferred = true
	}

	return err
}

func (itx *ModalInteraction) AcknowledgeWithMessage(ctx context.Context, response discord.ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		response.Flags |= discord.EphemeralMessageFlag
	}
	if itx.responded {
		return ErrAlreadyResponded
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &response,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *ModalInteraction) AcknowledgeWithLinearMessage(ctx context.Context, content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(ctx, discord.ResponseMessageData{Content: content}, ephemeral)
}

func (itx *ModalInteraction) AcknowledgeWithUpdate(ctx context.Context, response discord.ResponseMessageData) error {
	if itx.responded {
		return ErrAlreadyResponded
	}

	err := itx.respond(ctx, &itx.Interaction, &Response{
		Type: discord.UpdateMessageResponseType,
		Data: &response,
	})
	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *ModalInteraction) AcknowledgeWithLinearUpdate(ctx context.Context, content string) error {
	return itx.AcknowledgeWithUpdate(ctx, discord.ResponseMessageData{Content: content})
}

func (itx *ModalInteraction) EditOriginal(ctx context.Context, response discord.ResponseMessageData, files []File) error {
	return itx.editOriginal(ctx, &itx.Interaction, response, files)
}

func (itx *ModalInteraction) EditLinearOriginal(ctx context.Context, content string) error {
	return itx.EditOriginal(ctx, discord.ResponseMessageData{Content: content}, nil)
}

func (itx *ModalInteraction) DeleteOriginal(ctx context.Context) error {
	return itx.deleteOriginal(ctx, &itx.Interaction)
}

func (itx *ModalInteraction) SendFollowUp(ctx context.Context, content discord.ResponseMessageData, ephemeral bool) (discord.Message, error) {
	return itx.sendFollowUp(ctx, &itx.Interaction, content, ephemeral, nil)
}

func (itx *ModalInteraction) SendLinearFollowUp(ctx context.Context, content string, ephemeral bool) (discord.Message, error) {
	return itx.SendFollowUp(ctx, discord.ResponseMessageData{Content: content}, ephemeral)
}

func (itx *ModalInteraction) SendFollowUpWithFiles(ctx context.Context, content discord.ResponseMessageData, ephemeral bool, files []File) (discord.Message, error) {
	return itx.sendFollowUp(ctx, &itx.Interaction, content, ephemeral, files)
}

func (itx *ModalInteraction) EditFollowUp(ctx context.Context, messageID discord.MessageID, content discord.ResponseMessageData) error {
	return itx.editFollowUp(ctx, &itx.Interaction, messageID, content)
}

func (itx *ModalInteraction) DeleteFollowUp(ctx context.Context, messageID discord.MessageID) error {
	return itx.deleteFollowUp(ctx, &itx.Interaction, messageID)
}
