package command

import (
	"time"

	tempest "github.com/amatsagu/tempest"
)

var Swap tempest.Command = tempest.Command{
	Name:        "swap",
	Description: "Sends example embed and replaces it with plain text after 2 seconds.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
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
			// Define new, empty slice to include empty array when marshalled to JSON.
			// Discord looks for empty objects or arrays to define intentional lack of value.
			Embeds: make([]tempest.Embed, 0),
		}, false)
	},
}
