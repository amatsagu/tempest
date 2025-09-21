package command

import (
	"qord/api"
	"time"
)

var Defer api.Command = api.Command{
	Name:        "defer",
	Description: "Defer command for 2s, then sends reply message (follow up).",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		itx.Defer(false)
		time.Sleep(time.Second * 2)

		itx.SendFollowUp(api.ResponseMessageData{
			Content: "Hello after delay!",
		}, false)
	},
}
