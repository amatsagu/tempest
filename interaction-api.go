package tempest

import (
	"encoding/json"
	"errors"
)

type CommandInteraction Interaction

type ResponseWithFollow struct {
	*ResponseData
	Wait bool `json:"wait"`
}

// Use to let user/member know that bot is processing command.
// Make ephemeral = true to make notification visible only to target.
func (ctx CommandInteraction) Defer(ephemeral bool) error {
	var flags uint64 = 0

	if ephemeral {
		flags = 64
	}

	_, err := ctx.client.Rest.Request("PUT", "/interactions/"+ctx.Id.String()+"/"+ctx.Token+"/callback", Response{
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

	_, err := ctx.client.Rest.Request("POST", "/interactions/"+ctx.Id.String()+"/"+ctx.Token+"/callback", Response{
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

	_, err := ctx.client.Rest.Request("POST", "/interactions/"+ctx.Id.String()+"/"+ctx.Token+"/callback", Response{
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

	_, err := ctx.client.Rest.Request("PATCH", "/webhooks/"+ctx.client.ApplicationId.String()+"/"+ctx.Token+"/messages/@original", content)

	if err != nil {
		return err
	}
	return nil
}

func (ctx CommandInteraction) DeleteReply() error {
	_, err := ctx.client.Rest.Request("DELETE", "/webhooks/"+ctx.client.ApplicationId.String()+"/"+ctx.Token+"/messages/@original", nil)

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

	raw, err := ctx.client.Rest.Request("POST", "/webhooks/"+ctx.client.ApplicationId.String()+"/"+ctx.Token, content)
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
	_, err := ctx.client.Rest.Request("PATCH", "/webhooks/"+ctx.client.ApplicationId.String()+"/"+ctx.Token+"/messages/"+messageId.String(), content)
	if err != nil {
		return err
	}
	return nil
}

// Deletes a followup message for an Interaction. It does not support ephemeral followups.
func (ctx CommandInteraction) DeleteFollowUp(messageId Snowflake, content ResponseData) error {
	_, err := ctx.client.Rest.Request("DELETE", "/webhooks/"+ctx.client.ApplicationId.String()+"/"+ctx.Token+"/messages/"+messageId.String(), content)
	if err != nil {
		return err
	}
	return nil
}
