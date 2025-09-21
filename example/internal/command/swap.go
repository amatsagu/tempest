package command

import (
	"time"

	"github.com/amatsagu/qord/api"
)

var Swap api.Command = api.Command{
	Name:        "swap",
	Description: "Sends example embed and replaces it with plain text after 2 seconds.",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		itx.SendReply(api.ResponseMessageData{
			Content: "Example message",
			Embeds: []api.Embed{
				{
					Title:       "Example embed title",
					Description: "Example embed description",
				},
			},
		}, false, nil)

		time.Sleep(time.Second * 2)

		itx.EditReply(api.ResponseMessageData{
			Content: "Modified hello message!",
			// Define new, empty slice to include empty array when marshalled to JSON.
			// Discord looks for empty objects or arrays to define intentional lack of value.
			Embeds: make([]api.Embed, 0),
		}, false)
	},
}
