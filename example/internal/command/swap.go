package command

import (
	"time"

	tempest "github.com/amatsagu/tempest"
)

type SwapSlashCommand struct{}

func (cmd SwapSlashCommand) Data() tempest.Command {
	return tempest.Command{
		Name:        "swap",
		Description: "Sends example embed and replaces it with plain text after 2 seconds.",
	}
}

func (cmd SwapSlashCommand) AutoCompleteHandler(itx *tempest.CommandInteraction) []tempest.Choice {
	return nil
}

func (cmd SwapSlashCommand) CommandHandler(itx tempest.CommandInteraction) {
	itx.SendReply(tempest.ResponseMessageData{
		Content: "Example message",
		Embeds: []tempest.Embed{
			{
				Title:       "Example embed title",
				Description: "Example embed description",
			},
		},
	}, false, nil)

	time.Sleep(time.Second * 2)

	itx.EditReply(tempest.ResponseMessageData{
		Content: "Modified hello message!",
		// Provide single nill value for any field that you wish to signal it's empty.
		// Discord API requires specifically [] empty array as value when clearing embeds, components, etc. but it's hard to achieve with std encoding/json.
		// Tempest will replace all [null] with [] in stringified json. Different libraries may resolve that by using custom marshallers, json libs, etc.
		Embeds: []tempest.Embed{{}},
	}, false)
}
