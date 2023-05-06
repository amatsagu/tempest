package command

import (
	"log"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

var Modal tempest.Command = tempest.Command{
	Name:        "modal",
	Description: "Sends example modal interaction.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		firstInputID := itx.ID.String() + "_first" // Some unique id to filter for later. It's recommended to use id or token of interaction because it's always unique.

		itx.SendModal(tempest.ResponseModalData{
			CustomID: itx.ID.String(),
			Title:    "Example modal",
			Components: []*tempest.Component{
				{
					Type: tempest.COMPONENT_ROW,
					Components: []*tempest.Component{
						{
							CustomID: firstInputID,
							Type:     tempest.COMPONENT_TEXT_INPUT,
							Style:    tempest.BUTTON_PRIMARY,
							Label:    "Example label",
						},
					},
				},
			},
		})

		channel, stopFunction := itx.Client.AwaitComponent([]string{itx.ID.String(), firstInputID}, time.Minute*5)
		citx := <-channel
		if citx == nil {
			itx.SendFollowUp(tempest.ResponseData{Content: "Terminated listener (timeout)."}, false)
			return
		}

		log.Printf("%+v\n", citx)
	},
}
