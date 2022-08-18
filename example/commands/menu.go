package commands

import (
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

		itx.Client.AwaitComponent(tempest.QueueComponent{
			CustomIds: []string{firstButtonId, secondButtonId},
			TargetId:  itx.Member.User.Id,
			Handler: func(btx *tempest.Interaction) {
				if btx == nil {
					itx.SendFollowUp(tempest.ResponseData{Content: "You haven't clicked button within last 5 minutes!"}, false)
					return
				}

				itx.SendFollowUp(tempest.ResponseData{
					Content: "Successfully clicked button within 5 minutes! Button Component id: " + btx.Data.CustomId,
				}, false)
			},
		}, time.Minute*5)
	},
}
