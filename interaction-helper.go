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
	if itx.Member != nil && itx.Member.User != nil {
		return itx.Member.User
	}
	return itx.User
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

// Returns pointer to user if present in resolved data. It'll return empty struct if there's no resolved user.
func (r *InteractionDataResolved) ResolveUser(id Snowflake) User {
	if r == nil {
		return User{}
	}
	return r.Users[id]
}

// Returns pointer to member if present in resolved data and binds member.user. It'll return empty struct if there's no resolved member.
func (r *InteractionDataResolved) ResolveMember(id Snowflake) Member {
	if r == nil {
		return Member{}
	}
	member, available := r.Members[id]
	if available {
		user := r.Users[id]
		member.User = &user
		return member
	}
	return Member{}
}

// Returns pointer to guild role if present in resolved data. It'll return empty struct if there's no resolved role.
func (r *InteractionDataResolved) ResolveRole(id Snowflake) (Role, bool) {
	if r == nil {
		return Role{}, false
	}
	role, ok := r.Roles[id]
	return role, ok
}

// Returns pointer to partial channel if present in resolved data. It'll return empty struct if there's no resolved partial channel.
func (r *InteractionDataResolved) ResolveChannel(id Snowflake) PartialChannel {
	if r == nil {
		return PartialChannel{}
	}
	return r.Channels[id]
}

// Returns pointer to message if present in resolved data. It'll return empty struct if there's no resolved message.
func (r *InteractionDataResolved) ResolveMessage(id Snowflake) Message {
	if r == nil {
		return Message{}
	}
	return r.Messages[id]
}

// Returns pointer to attachment if present in resolved data. It'll return empty struct if there's no resolved attachment.
func (r *InteractionDataResolved) ResolveAttachment(id Snowflake) Attachment {
	if r == nil {
		return Attachment{}
	}
	return r.Attachments[id]
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
	err := itx.responder(Response{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})

	if err == nil {
		itx.responded = true
	}

	return err
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

// Acknowledges the interaction by updating the message on which the component was attached.
func (itx *ComponentInteraction) AcknowledgeWithUpdate(reply ResponseMessageData) error {
	return itx.responder(Response{
		Type: UPDATE_MESSAGE_RESPONSE_TYPE,
		Data: &reply,
	})
}

func (itx *ComponentInteraction) AcknowledgeWithLinearUpdate(content string) error {
	return itx.AcknowledgeWithUpdate(ResponseMessageData{
		Content: content,
	})
}

// Retrieves the contents of the first [TextInputComponent] inside the modal (at any depth) with the given customID.
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

// Retrieves the uploaded file attachment IDs of the first [FileUploadComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetFileUploadValues(customID string) []Snowflake {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetFileUploadValues was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return nil
	}

	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if upload, ok := label.Component.(FileUploadComponent); ok && upload.CustomID == customID {
				return upload.Values
			}
		}
	}

	return nil
}

// Retrieves the resolved [Attachment] objects of the first [FileUploadComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists or no attachments were resolved, nil is returned instead.
func (itx *ModalInteraction) GetFileUploadAttachments(customID string) []Attachment {
	ids := itx.GetFileUploadValues(customID)
	if len(ids) == 0 || itx.Data.Resolved == nil || len(itx.Data.Resolved.Attachments) == 0 {
		return nil
	}

	attachments := make([]Attachment, 0, len(ids))
	for _, id := range ids {
		if attachment, ok := itx.Data.Resolved.Attachments[id]; ok {
			attachments = append(attachments, attachment)
		}
	}

	if len(attachments) == 0 {
		return nil
	}

	return attachments
}

// Retrieves the selected value of the first [RadioGroupComponent] inside the modal (at any depth) with the given customID.
// The second return value indicates whether a value was selected.
func (itx *ModalInteraction) GetRadioGroupValue(customID string) (string, bool) {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetRadioGroupValue was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return "", false
	}

	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if radio, ok := label.Component.(RadioGroupComponent); ok && radio.CustomID == customID {
				if radio.Value != nil {
					return *radio.Value, true
				}
				return "", false
			}
		}
	}

	return "", false
}

// Retrieves the selected values of the first [CheckboxGroupComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetCheckboxGroupValues(customID string) []string {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetCheckboxGroupValues was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return nil
	}

	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if group, ok := label.Component.(CheckboxGroupComponent); ok && group.CustomID == customID {
				return group.Values
			}
		}
	}

	return nil
}

// Retrieves the state of the first [CheckboxComponent] inside the modal (at any depth) with the given customID.
// The second return value indicates whether the component was found.
func (itx *ModalInteraction) GetCheckboxValue(customID string) (bool, bool) {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetCheckboxValue was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return false, false
	}

	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if checkbox, ok := label.Component.(CheckboxComponent); ok && checkbox.CustomID == customID {
				return checkbox.Value, true
			}
		}
	}

	return false, false
}

// Retrieves the selected values of the first [StringSelectComponent] inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetStringSelectValues(customID string) []string {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetStringSelectValues was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return nil
	}

	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if selectMenu, ok := label.Component.(StringSelectComponent); ok && selectMenu.CustomID == customID {
				return selectMenu.Values
			}
		}
	}

	return nil
}

// Retrieves the selected snowflake IDs of the first [SelectComponent] (user, role, mentionable, or channel select) inside the modal (at any depth) with the given customID.
//
// If no such component exists, nil is returned instead.
func (itx *ModalInteraction) GetSelectValues(customID string) []Snowflake {
	if customID == "" {
		// TODO: Display warning even if tracing is disabled
		itx.BaseClient.tracef(
			"Warning: ModalInteraction.GetSelectValues was called with an empty customID, " +
				"which is invalid and will never appear inside a component.",
		)
		return nil
	}

	for _, row := range itx.Data.Components {
		if label, ok := row.(LabelComponent); ok {
			if selectMenu, ok := label.Component.(SelectComponent); ok && selectMenu.CustomID == customID {
				return selectMenu.Values
			}
		}
	}

	return nil
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

// Acknowledges the interaction by updating the message on which the component was attached.
func (itx *ModalInteraction) AcknowledgeWithUpdate(response ResponseMessageData) error {
	return itx.responder(Response{
		Type: UPDATE_MESSAGE_RESPONSE_TYPE,
		Data: &response,
	})
}

func (itx *ModalInteraction) AcknowledgeWithLinearUpdate(content string) error {
	return itx.AcknowledgeWithUpdate(ResponseMessageData{
		Content: content,
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
