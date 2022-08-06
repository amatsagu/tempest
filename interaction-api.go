package tempest

import (
	"encoding/json"
	"errors"
)

type CommandInteraction Interaction
type AutoCompleteInteraction Interaction

// Returns value of any type. Check second value to check whether option was provided or not (true if yes).
// Use this method when working with Command-like interactions.
func (ctx CommandInteraction) GetOptionValue(name string) (any, bool) {
	options := ctx.Data.Options
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

// Use to let user/member know that bot is processing command.
// Make ephemeral = true to make notification visible only to target.
func (ctx CommandInteraction) Defer(ephemeral bool) error {
	var flags uint64 = 0

	if ephemeral {
		flags = 64
	}

	_, err := ctx.Client.Rest.Request("PUT", "/interactions/"+ctx.Id.String()+"/"+ctx.Token+"/callback", Response{
		Type: ACKNOWLEDGE_WITH_SOURCE_RESPONSE,
		Data: ResponseData{
			Flags: flags,
		},
	})

	if err != nil {
		return err
	}
	return nil
}

// Acknowledges the interaction with a message. Set ephemeral = true to make message visible only to target.
func (ctx CommandInteraction) SendReply(content ResponseData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := ctx.Client.Rest.Request("POST", "/interactions/"+ctx.Id.String()+"/"+ctx.Token+"/callback", Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE,
		Data: content,
	})

	if err != nil {
		return err
	}
	return nil
}

// Use that for simple text messages that won't be modified.
func (ctx CommandInteraction) SendLinearReply(content string, ephemeral bool) error {
	var flags uint64 = 0

	if ephemeral {
		flags = 64
	}

	_, err := ctx.Client.Rest.Request("POST", "/interactions/"+ctx.Id.String()+"/"+ctx.Token+"/callback", Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE,
		Data: ResponseData{
			Content: content,
			Flags:   flags,
		},
	})

	if err != nil {
		return err
	}
	return nil
}

func (ctx CommandInteraction) EditReply(content ResponseData, ephemeral bool) error {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	_, err := ctx.Client.Rest.Request("PATCH", "/webhooks/"+ctx.Client.ApplicationId.String()+"/"+ctx.Token+"/messages/@original", content)

	if err != nil {
		return err
	}
	return nil
}

func (ctx CommandInteraction) DeleteReply() error {
	_, err := ctx.Client.Rest.Request("DELETE", "/webhooks/"+ctx.Client.ApplicationId.String()+"/"+ctx.Token+"/messages/@original", nil)

	if err != nil {
		return err
	}
	return nil
}

// Create a followup message for an Interaction.
func (ctx CommandInteraction) SendFollowUp(content ResponseData, ephemeral bool) (Message, error) {
	if ephemeral && content.Flags == 0 {
		content.Flags = 64
	}

	raw, err := ctx.Client.Rest.Request("POST", "/webhooks/"+ctx.Client.ApplicationId.String()+"/"+ctx.Token, content)
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
func (ctx CommandInteraction) EditFollowUp(messageId Snowflake, content ResponseData) error {
	_, err := ctx.Client.Rest.Request("PATCH", "/webhooks/"+ctx.Client.ApplicationId.String()+"/"+ctx.Token+"/messages/"+messageId.String(), content)
	if err != nil {
		return err
	}
	return nil
}

// Deletes a followup message for an Interaction. It does not support ephemeral followups.
func (ctx CommandInteraction) DeleteFollowUp(messageId Snowflake, content ResponseData) error {
	_, err := ctx.Client.Rest.Request("DELETE", "/webhooks/"+ctx.Client.ApplicationId.String()+"/"+ctx.Token+"/messages/"+messageId.String(), content)
	if err != nil {
		return err
	}
	return nil
}

// Returns option name and its value of triggered option. Option name is always of string type but you'll need to check type of value.
func (ctx AutoCompleteInteraction) GetFocusedValue() (string, any) {
	options := ctx.Data.Options

	for _, option := range options {
		if option.Focused {
			return option.Name, option.Value
		}
	}

	panic("auto complete interaction had no option with \"focused\" field. This error should never happen")
}
