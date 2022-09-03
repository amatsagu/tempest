package commands

import (
	"fmt"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

var Menu tempest.Command = tempest.Command{
	Name:        "menu",
	Description: "Creates message with example button menu.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		firstButtonId := itx.Id.String() + "_first" // Some unique id to filter for later. It's recommended to use id or token of interaction because it's always unique.
		secondButtonId := itx.Id.String() + "_second"

		itx.SendReply(tempest.ResponseData{
			Content: "Example message",
			Components: []*tempest.Component{
				{
					Type: tempest.COMPONENT_ROW,
					Components: []*tempest.Component{
						{
							CustomId: firstButtonId,
							Type:     tempest.COMPONENT_BUTTON,
							Style:    tempest.BUTTON_PRIMARY,
							Label:    "First button",
						},
						{
							CustomId: secondButtonId,
							Type:     tempest.COMPONENT_BUTTON,
							Style:    tempest.BUTTON_SECONDARY,
							Label:    "Second button",
						},
					},
				},
			},
		}, false)

		channel, stopFunction := itx.Client.AwaitComponent([]string{firstButtonId, secondButtonId}, time.Minute*2)
		for {
			citx := <-channel
			if citx == nil {
				itx.SendFollowUp(tempest.ResponseData{Content: "Nobody clicked button within last 2 minutes!"}, false)
				return
			}

			if citx.Member.User.Id == itx.Member.User.Id {
				itx.SendLinearReply(fmt.Sprintf(`Detected that you clicked "%s" button! Terminating listener!`, citx.Data.CustomId), false)
				stopFunction() // <== Terminates listener before reaching timeout.
			} else {
				itx.SendLinearReply(fmt.Sprintf(`Member "%s" clicked "%s" button!`, citx.Member.User.Username, citx.Data.CustomId), false)
			}
		}
	},
}
