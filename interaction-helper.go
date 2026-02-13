package tempest

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Returns whether this interaction already was responded to.
func (itx *Interaction) Responded() bool {
	return itx.responded
}

// Returns user data either from member or from user (depending if interaction was used in a server).
func (itx *Interaction) BaseUser() *User {
	if itx.GuildID == 0 {
		return itx.User
	} else {
		return itx.Member.User
	}
}

// Returns value of any type. Check second value to check whether option was provided or not (true if yes).
func (itx *CommandInteraction) GetOptionValue(name string) (any, bool) {
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
func (itx *CommandInteraction) ResolveUser(id Snowflake) User {
	return itx.Data.Resolved.Users[id]
}

// Returns pointer to member if present in interaction.data.resolved and binds member.user. It'll return empty struct if there's no resolved member.
func (itx *CommandInteraction) ResolveMember(id Snowflake) Member {
	member, available := itx.Data.Resolved.Members[id]
	if available {
		user := itx.Data.Resolved.Users[id]
		member.User = &user
		return member
	}
	return Member{}
}

// Returns pointer to guild role if present in interaction.data.resolved. It'll return empty struct if there's no resolved role.
func (itx *CommandInteraction) ResolveRole(id Snowflake) (Role, bool) {
	role, ok := itx.Data.Resolved.Roles[id]
	return role, ok
}

// Returns pointer to partial channel if present in interaction.data.resolved.  It'll return empty struct if there's no resolved partial channel.
func (itx *CommandInteraction) ResolveChannel(id Snowflake) PartialChannel {
	return itx.Data.Resolved.Channels[id]
}

// Returns pointer to message if present in interaction.data.resolved.  It'll return empty struct if there's no resolved message.
func (itx *CommandInteraction) ResolveMessage(id Snowflake) Message {
	return itx.Data.Resolved.Messages[id]
}

// Returns pointer to attachment if present in interaction.data.resolved.  It'll return empty struct if there's no resolved attachment.
func (itx *CommandInteraction) ResolveAttachment(id Snowflake) Attachment {
	return itx.Data.Resolved.Attachments[id]
}

// Use to let user/member know that bot is processing command.
// Make ephemeral = true to make notification visible only to target.
func (itx *CommandInteraction) Defer(ephemeral bool) error {
	if itx.deferred || itx.responded {
		return errors.New("interaction has already been responded to or deferred")
	}

	var flags MessageFlags = 0
	if ephemeral {
		flags = EPHEMERAL_MESSAGE_FLAG
	}

	err := itx.responder(Response{
		Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{Flags: flags},
	})

	if err == nil {
		itx.deferred = true
	}

	return err
}

// Acknowledges the interaction with a message. Set ephemeral = true to make message visible only to target.
func (itx *CommandInteraction) SendReply(reply ResponseMessageData, ephemeral bool, files []File) error {
	if ephemeral {
		reply.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	if itx.responded {
		return errors.New("interaction has already been responded to")
	}

	endpoint := "/webhooks/" + itx.ApplicationID.String() + "/" + itx.Token

	if itx.deferred {
		_, err := itx.BaseClient.Rest.RequestWithFiles(http.MethodPatch, endpoint+"/messages/@original", reply, files)
		if err == nil {
			itx.responded = true
		}
		return err
	}

	if len(files) > 0 {
		err := itx.responder(Response{
			Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
			Data: &ResponseMessageData{Flags: reply.Flags},
		})
		if err != nil {
			return err
		}

		itx.deferred = true // Manually set state

		_, err = itx.BaseClient.Rest.RequestWithFiles(http.MethodPatch, endpoint+"/messages/@original", reply, files)
		if err == nil {
			itx.responded = true
		}

		return err
	}

	err := itx.responder(Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &reply,
	})

	if err == nil {
		itx.responded = true
	}

	return err
}

func (itx *CommandInteraction) SendLinearReply(content string, ephemeral bool) error {
	return itx.SendReply(ResponseMessageData{
		Content: content,
	}, ephemeral, nil)
}

func (itx *CommandInteraction) SendModal(modal ResponseModalData) error {
	// TODO: Add validation for modal data (1-5 components, title length, non-empty customID)
	return itx.responder(Response{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})
}

func (itx *CommandInteraction) EditReply(content ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	_, err := itx.BaseClient.Rest.Request(http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content)
	return err
}

func (itx *CommandInteraction) EditLinearReply(content string, ephemeral bool) error {
	return itx.EditReply(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *CommandInteraction) DeleteReply() error {
	_, err := itx.BaseClient.Rest.Request(http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil)
	return err
}

func (itx *CommandInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error) {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	raw, err := itx.BaseClient.Rest.Request(http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content)
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

func (itx *CommandInteraction) SendLinearFollowUp(content string, ephemeral bool) (Message, error) {
	return itx.SendFollowUp(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *CommandInteraction) SendFollowUpWithFiles(content ResponseMessageData, ephemeral bool, files []File) (Message, error) {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	raw, err := itx.BaseClient.Rest.RequestWithFiles(http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content, files)
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

func (itx *CommandInteraction) SendLinearFollowUpWithFiles(content string, ephemeral bool, files []File) (Message, error) {
	return itx.SendFollowUpWithFiles(ResponseMessageData{
		Content: content,
	}, ephemeral, files)
}

func (itx *CommandInteraction) EditFollowUp(messageID Snowflake, content ResponseMessageData) error {
	_, err := itx.BaseClient.Rest.Request(http.MethodPatch, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
	return err
}

func (itx *CommandInteraction) EditLinearFollowUp(messageID Snowflake, content string) error {
	return itx.EditFollowUp(messageID, ResponseMessageData{
		Content: content,
	})
}

func (itx *CommandInteraction) DeleteFollowUp(messageID Snowflake) error {
	_, err := itx.BaseClient.Rest.Request(http.MethodDelete, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), nil)
	return err
}

// Warning! This method is only for handling auto complete interaction which is a part of command logic.
// Returns option name and its value of triggered option. Option name is always of string type but you'll need to check type of value.
func (itx *CommandInteraction) GetFocusedValue() (string, any) {
	for _, option := range itx.Data.Options {
		if option.Focused {
			return option.Name, option.Value
		}
	}

	panic("auto complete interaction had no option with \"focused\" field. This error should never happen with correctly defined slash command")
}

// Sends to discord info that this component was handled successfully without sending anything more.
func (itx *ComponentInteraction) Acknowledge() error {
	return itx.responder(Response{
		Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE,
	})
}

func (itx *ComponentInteraction) AcknowledgeWithMessage(reply ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		reply.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	return itx.responder(Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &reply,
	})
}

func (itx *ComponentInteraction) AcknowledgeWithLinearMessage(content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *ComponentInteraction) AcknowledgeWithModal(modal ResponseModalData) error {
	return itx.responder(Response{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})
}

// GetInputValue retrieves the contents of the first [TextInputComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, an empty string is returned instead.
func (itx *ModalInteraction) GetInputValue(customID string) string {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetInputValue was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return ""
	}

	// Currently, text inputs can only legally be placed inside LabelComponents, so we only need to check there
	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if input, ok := label.Component.(TextInputComponent); ok && input.CustomID == customID {
				return input.Value
			}
		}
	}

	return ""
}

// Sends to discord info that this component was handled successfully without sending anything more.
func (itx *ModalInteraction) Acknowledge() error {
	return itx.responder(Response{
		Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE,
	})
}

func (itx *ModalInteraction) AcknowledgeWithMessage(response ResponseMessageData, ephemeral bool) error {
	if ephemeral {
		response.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	return itx.responder(Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &response,
	})
}

func (itx *ModalInteraction) AcknowledgeWithLinearMessage(content string, ephemeral bool) error {
	return itx.AcknowledgeWithMessage(ResponseMessageData{
		Content: content,
	}, ephemeral)
}

func (itx *ModalInteraction) AcknowledgeWithModal(modal ResponseModalData) error {
	return itx.responder(Response{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})
}

// Used to let user/member know that the bot is processing the modal submission,
// which will close the modal dialog.
// Set ephemeral = true to make notification visible only to the submitter.
func (itx *ModalInteraction) Defer(ephemeral bool) error {
	var flags MessageFlags = 0

	if ephemeral {
		flags = EPHEMERAL_MESSAGE_FLAG
	}

	_, err := itx.BaseClient.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{
			Flags: flags,
		},
	})

	if err == nil {
		itx.deferred = true
	}

	return err
}

// Used after defering a modal submission to send a message to the user/member.
// Set ephemeral = true to make message visible only to the submitter.
func (itx *ModalInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error) {
	if ephemeral {
		content.Flags |= EPHEMERAL_MESSAGE_FLAG
	}

	raw, err := itx.BaseClient.Rest.Request(http.MethodPost, "/webhooks/"+itx.ApplicationID.String()+"/"+itx.Token, content)
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
func (itx *ModalInteraction) SendLinearFollowUp(content string, ephemeral bool) (Message, error) {
	return itx.SendFollowUp(ResponseMessageData{
		Content: content,
	}, ephemeral)
}
