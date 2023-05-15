package tempest

import (
	"errors"
	"net/http"

	"github.com/sugawarayuuta/sonnet"
)

// Returns value of any type. Check second value to check whether option was provided or not (true if yes).
// Use this method when working with Command-like interactions.
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

// Returns pointer to user if present in interaction.data.resolved. It'll return <nil> if there's no resolved user.
func (itx CommandInteraction) ResolveUser(id Snowflake) *User {
	return itx.Data.Resolved.Users[id]
}

// Returns pointer to member if present in interaction.data.resolved and binds member.user. It'll return <nil> if there's no resolved member.
func (itx CommandInteraction) ResolveMember(id Snowflake) *Member {
	member, available := itx.Data.Resolved.Members[id]
	if available {
		member.User = itx.Data.Resolved.Users[id]
		return member
	}
	return nil
}

// Returns pointer to guild role if present in interaction.data.resolved. It'll return <nil> if there's no resolved role.
func (itx CommandInteraction) ResolveRole(id Snowflake) *Role {
	return itx.Data.Resolved.Roles[id]
}

// Use to let user/member know that bot is processing command.
// Make ephemeral = true to make notification visible only to target.
func (itx CommandInteraction) Defer(ephemeral bool) error {
	var flags uint64 = 0

	if ephemeral {
		flags = 64
	}

	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{
			Flags: flags,
		},
	})

	return err
}

// Acknowledges the interaction with a message. Set ephemeral = true to make message visible only to target.
func (itx CommandInteraction) SendReply(content ResponseMessageData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &content,
	})

	return err
}

// Use that for simple text messages that won't be modified.
func (itx CommandInteraction) SendLinearReply(content string, ephemeral bool) error {
	var flags uint64 = 0

	if ephemeral {
		flags = 64
	}

	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{
			Content: content,
			Flags:   flags,
		},
	})

	return err
}

func (itx CommandInteraction) EditReply(content ResponseMessageData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := itx.Client.Rest.Request(http.MethodDelete, "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content)
	return err
}

func (itx CommandInteraction) DeleteReply() error {
	_, err := itx.Client.Rest.Request(http.MethodDelete, "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil)
	return err
}

// Create a followup message for an Interaction.
func (itx CommandInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error) {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	raw, err := itx.Client.Rest.Request(http.MethodPatch, "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token, content)
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = sonnet.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// Edits a followup message for an Interaction.
func (itx CommandInteraction) EditFollowUp(messageID Snowflake, content ResponseMessage) error {
	_, err := itx.Client.Rest.Request(http.MethodPatch, "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
	return err
}

// Deletes a followup message for an Interaction. It does not support ephemeral followups.
func (itx CommandInteraction) DeleteFollowUp(messageID Snowflake, content ResponseMessage) error {
	_, err := itx.Client.Rest.Request(http.MethodDelete, "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
	return err
}

// Returns option name and its value of triggered option. Option name is always of string type but you'll need to check type of value.
func (itx AutoCompleteInteraction) GetFocusedValue() (string, any) {
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
	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE,
	})
	return err
}

func (itx ComponentInteraction) AcknowledgeWithMessage(content ResponseMessageData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseMessage{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &content,
	})
	return err
}

func (itx ComponentInteraction) AcknowledgeWithModal(modal ResponseModalData) error {
	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseModal{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})
	return err
}

func (itx CommandInteraction) SendModal(modal ResponseModalData) error {
	_, err := itx.Client.Rest.Request(http.MethodPost, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseModal{
		Type: MODAL_RESPONSE_TYPE,
		Data: &modal,
	})
	return err
}
