package tempest

import (
	"encoding/json"
	"errors"
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

	_, err := itx.Client.Rest.Request("POST", "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", Response{
		Type: DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE,
		Data: &ResponseData{
			Flags: flags,
		},
	})

	return err
}

// Acknowledges the interaction with a message. Set ephemeral = true to make message visible only to target.
func (itx CommandInteraction) SendReply(content ResponseData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := itx.Client.Rest.Request("POST", "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE,
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

	_, err := itx.Client.Rest.Request("POST", "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE,
		Data: &ResponseData{
			Content:    content,
			Embeds:     make([]*Embed, 1),
			Components: make([]*Component, 1),
			Flags:      flags,
		},
	})

	return err
}

func (itx CommandInteraction) EditReply(content ResponseData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := itx.Client.Rest.Request("PATCH", "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/@original", content)
	return err
}

func (itx CommandInteraction) DeleteReply() error {
	_, err := itx.Client.Rest.Request("DELETE", "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/@original", nil)
	return err
}

// Create a followup message for an Interaction.
func (itx CommandInteraction) SendFollowUp(content ResponseData, ephemeral bool) (Message, error) {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	raw, err := itx.Client.Rest.Request("POST", "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token, content)
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

// Edits a followup message for an Interaction.
func (itx CommandInteraction) EditFollowUp(messageID Snowflake, content ResponseData) error {
	_, err := itx.Client.Rest.Request("PATCH", "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
	return err
}

// Deletes a followup message for an Interaction. It does not support ephemeral followups.
func (itx CommandInteraction) DeleteFollowUp(messageID Snowflake, content ResponseData) error {
	_, err := itx.Client.Rest.Request("DELETE", "/webhooks/"+itx.Client.ApplicationID.String()+"/"+itx.Token+"/messages/"+messageID.String(), content)
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

	panic("auto complete interaction had no option with \"focused\" field. This error should never happen")
}

func (itx CommandInteraction) SendModal(modal ResponseModalData) error {
	_, err := itx.Client.Rest.Request("POST", "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", ResponseModal{
		Type: MODAL_RESPONSE,
		Data: &modal,
	})
	return err
}

// Use that if you need to make a call that is not already supported by Tempest.
func (itx Interaction) SendCustomCallback(method string, callback Response) error {
	_, err := itx.Client.Rest.Request(method, "/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback", callback)
	return err
}
