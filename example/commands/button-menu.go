package commands

import (
	"fmt"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

var ButtonMenu tempest.Command = tempest.Command{
	Name:        "button-menu",
	Description: "Creates message with example button menu.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		firstButtonID := itx.ID.String() + "_first" // Some unique id to filter for later. It's recommended to use id or token of interaction because it's always unique.
		secondButtonID := itx.ID.String() + "_second"

		itx.SendReply(tempest.ResponseData{
			Content: "Example message",
			Components: []*tempest.Component{
				{
					Type: tempest.COMPONENT_ROW,
					Components: []*tempest.Component{
						{
							CustomID: firstButtonID,
							Type:     tempest.COMPONENT_BUTTON,
							Style:    tempest.BUTTON_PRIMARY,
							Label:    "First button",
						},
						{
							CustomID: secondButtonID,
							Type:     tempest.COMPONENT_BUTTON,
							Style:    tempest.BUTTON_SECONDARY,
							Label:    "Second button",
						},
					},
				},
			},
		}, false)

		channel, stopFunction := itx.Client.AwaitComponent([]string{firstButtonID, secondButtonID}, time.Minute*2)
		for {
			citx := <-channel
			if citx == nil {
				itx.SendFollowUp(tempest.ResponseData{Content: "Terminated listener (timeout)."}, false)
				break
			}

			response := fmt.Sprintf(`Member "%s" clicked "%s" button!`, citx.Member.User.Username, citx.Data.CustomID)
			if citx.Member.User.ID == itx.Member.User.ID {
				response += "\nSince it's you, this listener will be now terminated."
				stopFunction() // <== Terminates listener before reaching timeout. Use it if it hits desired target.
				itx.EditReply(tempest.ResponseData{Content: response}, false)
				break
			}

			itx.EditReply(tempest.ResponseData{Content: response}, false)
		}
	},
}
