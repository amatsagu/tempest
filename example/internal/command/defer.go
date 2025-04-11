package command

import (
	"time"

	tempest "github.com/amatsagu/tempest"
)

var Defer tempest.Command = tempest.Command{
	Name:        "defer",
	Description: "Defer command for 2s, then sends reply message (follow up).",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		itx.Defer(false)
		time.Sleep(time.Second * 2)

		itx.SendFollowUp(tempest.ResponseMessageData{
			Content: "Hello after delay!",
		}, false)
	},
}
