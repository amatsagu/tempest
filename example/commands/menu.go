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
				itx.SendFollowUp(tempest.ResponseData{Content: "Terminated listener (timeout)."}, false)
				break
			}

			response := fmt.Sprintf(`Member "%s" clicked "%s" button!`, citx.Member.User.Username, citx.Data.CustomId)
			if citx.Member.User.Id == itx.Member.User.Id {
				response += "\nSince it's you, this listener will be now terminated."
				stopFunction() // <== Terminates listener before reaching timeout. Use it if it hits desired target.
				itx.EditReply(tempest.ResponseData{Content: response}, false)
				break
			}

			itx.EditReply(tempest.ResponseData{Content: response}, false)
		}
	},
}
