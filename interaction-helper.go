package tempest

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Returns value of any type. Check second value to check whether option was provided or not (true if yes).
func (itx CommandInteraction) GetOptionValue(name string) (any, bool) {
	options := itx.Data.Options
	if len(options) == 0 {
		return nil, false
	}

	for _, option := range options {
		if option.Name == name {
			return option.Value, true
		}
	}

	return nil, false
}

// Returns pointer to user if present in interaction.data.resolved. It'll return empty struct if there's no resolved user.
func (itx CommandInteraction) ResolveUser(id Snowflake) User {
	return itx.Data.Resolved.Users[id]
}

// Returns pointer to member if present in interaction.data.resolved and binds member.user. It'll return empty struct if there's no resolved member.
func (itx CommandInteraction) ResolveMember(id Snowflake) Member {
	member, available := itx.Data.Resolved.Members[id]
	if available {
		user := itx.Data.Resolved.Users[id]
		member.User = &user
		return member
	}
	return Member{}
}

// Returns pointer to guild role if present in interaction.data.resolved. It'll return empty struct if there's no resolved role.
func (itx CommandInteraction) ResolveRole(id Snowflake) (Role, bool) {
	role, ok := itx.Data.Resolved.Roles[id]
	return role, ok
}

// Returns pointer to partial channel if present in interaction.data.resolved.  It'll return empty struct if there's no resolved partial channel.
func (itx CommandInteraction) ResolveChannel(id Snowflake) PartialChannel {
	return itx.Data.Resolved.Channels[id]
}

// Returns pointer to message if present in interaction.data.resolved.  It'll return empty struct if there's no resolved message.
func (itx CommandInteraction) ResolveMessage(id Snowflake) Message {
	return itx.Data.Resolved.Messages[id]
}

// Returns pointer to attachment if present in interaction.data.resolved.  It'll return empty struct if there's no resolved attachment.
func (itx CommandInteraction) ResolveAttachment(id Snowflake) Attachment {
	return itx.Data.Resolved.Attachments[id]
}

// Use to let user/member know that bot is processing command.
// Make ephemeral = true to make notification visible only to target.
func (itx *CommandInteraction) Defer(ephemeral bool) error {
	var flags MessageFlags = 0

	if ephemeral {
		flags = EPHEMERAL_MESSAGE_FLAG
	}

	body, err := json.Marshal(ResponseMessage{
		Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{Flags: flags},
	})
	if err != nil {
		return err
	}

	// Send response via channel for fast HTTP return
	if itx.responseChan != nil {
		select {
		case itx.responseChan <- body:
			itx.deferred = true
			return nil
		default:
			// Channel full, fall back to direct write
		}
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	_, werr := itx.w.Write(body)

	if werr == nil {
		itx.deferred = true
	}

	return werr
}

// Acknowledges the interaction with a message. Set ephemeral = true to make message visible only to target.
func (itx *CommandInteraction) SendReply(reply ResponseMessageData, ephemeral bool, files []File) error {
	if ephemeral {
		reply.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	// If files are present, defer fast and then edit original with files
	if len(files) > 0 {
		if !itx.deferred && !itx.responded {
			if derr := itx.Defer(ephemeral); derr != nil {
				return derr
			}
		}
		_, err := itx.Client.Rest.RequestWithFiles(http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", reply, files)
		return err
	}

	// If we've already responded once, send a follow-up
	if itx.responded {
		_, err := itx.Client.Rest.Request(http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, reply)
		return err
	}

	// First response goes through the initial HTTP response
	body, err := json.Marshal(ResponseMessage{Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE, Data: &reply})
	if err != nil {
		return err
	}

	// Send response via channel for fast HTTP return
	if itx.responseChan != nil {
		select {
		case itx.responseChan <- body:
			itx.responded = true
			return nil
		default:
			// Channel full, fall back to direct write
		}
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	_, werr := itx.w.Write(body)

	if werr == nil {
		itx.responded = true
	}

	return werr
}

func (itx *CommandInteraction) SendLinearReply(content string, ephemeral bool) error {
	return itx.SendReply(ResponseMessageData{
		Content: content,
	}, ephemeral, nil)
}

func (itx *CommandInteraction) SendModal(modal ResponseModalData) error {
	body, err := json.Marshal(ResponseModal{Type: MODAL_RESPONSE_TYPE, Data: &modal})
	if err != nil {
		return err
	}

	// Send response via channel for fast HTTP return
	if itx.responseChan != nil {
		select {
		case itx.responseChan <- body:
			itx.responded = true
			return nil
		default:
			// Channel full, fall back to direct write
		}
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	_, werr := itx.w.Write(body)
	if werr == nil {
		itx.responded = true
	}
	return werr
}

func (itx CommandInteraction) EditReply(content ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	_, err := itx.Client.Rest.Request(http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content)
	return err
}

func (itx CommandInteraction) EditLinearReply(content string, ephemeral bool) error {
	return itx.EditReply(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx CommandInteraction) DeleteReply() error {
	_, err := itx.Client.Rest.Request(http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil)
	return err
}

func (itx CommandInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error) {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	raw, err := itx.Client.Rest.Request(http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content)
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (itx CommandInteraction) SendLinearFollowUp(content string, ephemeral bool) (Message, error) {
	return itx.SendFollowUp(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx CommandInteraction) EditFollowUp(messageID Snowflake, content ResponseMessageData) error {
	_, err := itx.Client.Rest.Request(http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
	return err
}

func (itx CommandInteraction) EditLinearFollowUp(messageID Snowflake, content string) error {
	return itx.EditFollowUp(messageID, ResponseMessageData{
		Content: content,
	})
}

func (itx CommandInteraction) DeleteFollowUp(messageID Snowflake, content ResponseMessage) error {
	_, err := itx.Client.Rest.Request(http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
	return err
}

// Warning! This method is only for handling auto complete interaction which is a part of command logic.
// Returns option name and its value of triggered option. Option name is always of string type but you'll need to check type of value.
func (itx CommandInteraction) GetFocusedValue() (string, any) {
	options := itx.Data.Options

	for _, option := range options {
		if option.Focused {
			return option.Name, option.Value
		}
	}

	panic("auto complete interaction had no option with \"focused\" field. This error should never happen with correctly defined slash command")
}

// Sends to discord info that this component was handled successfully without sending anything more.
func (itx ComponentInteraction) Acknowledge() error {
	body, err := json.Marshal(ResponseMessage{
		Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE,
	})

	if err != nil {
		return err
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	itx.w.Write(body)
	return err
}

func (itx ComponentInteraction) AcknowledgeWithMessage(reply ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		reply.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	body, err := json.Marshal(ResponseMessage{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &reply,
	})

	if err != nil {
		return err
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	itx.w.Write(body)
	return err
}

func (itx ComponentInteraction) AcknowledgeWithLinearMessage(content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx ComponentInteraction) AcknowledgeWithModal(modal ResponseModalData) error {
	body, err := json.Marshal(ResponseModal{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})

	if err != nil {
		return err
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	itx.w.Write(body)
	return err
}

// Returns value of any type. It will return empty string on no value or empty value.
func (itx ModalInteraction) GetInputValue(customID string) string {
	rows := itx.Data.Components
	if len(rows) == 0 {
		return ""
	}

	for _, row := range rows {
		for _, component := range row.(ActionRowComponent).Components {
			scmp := component.(TextInputComponent)
			if scmp.CustomID == customID {
				return scmp.Value
			}
		}
	}

	return ""
}

// Sends to discord info that this component was handled successfully without sending anything more.
func (itx ModalInteraction) Acknowledge() error {
	body, err := json.Marshal(ResponseMessage{
		Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE,
	})

	if err != nil {
		return err
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	itx.w.Write(body)
	return err
}

func (itx ModalInteraction) AcknowledgeWithMessage(response ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		response.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	body, err := json.Marshal(ResponseMessage{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &response,
	})

	if err != nil {
		return err
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	itx.w.Write(body)
	return err
}

func (itx ModalInteraction) AcknowledgeWithLinearMessage(content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx ModalInteraction) AcknowledgeWithModal(modal ResponseModalData) error {
	body, err := json.Marshal(ResponseModal{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})

	if err != nil {
		return err
	}

	itx.w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
	itx.w.Write(body)
	return err
}

// Used to let user/member know that the bot is processing the modal submission,
// which will close the modal dialog.
// Set ephemeral = true to make notification visible only to the submitter.
func (itx ModalInteraction) Defer(ephemeral bool) error {
	var flags MessageFlags = 0

	if ephemeral {
		flags = EPHEMERAL_MESSAGE_FLAG
	}

	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{
			Flags: flags,
		},
	})

	return err
}

// Used after defering a modal submission to send a message to the user/member.
// Set ephemeral = true to make message visible only to the submitter.
func (itx ModalInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error) {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	raw, err := itx.Client.Rest.Request(http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content)
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// Used after defering a modal submission to send a message to the user/member.
// Set ephemeral = true to make message visible only to the submitter.
func (itx ModalInteraction) SendLinearFollowUp(content string, ephemeral bool) (Message, error) {
	return itx.SendFollowUp(ResponseMessageData{
		Content: content,
	}, ephemeral)
}
