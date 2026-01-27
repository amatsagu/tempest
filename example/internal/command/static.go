package command

import (
	"fmt"
	"log"

	tempest "github.com/amatsagu/tempest"
)

var Static tempest.Command = tempest.Command{
	Name:        "static",
	Description: "Sends example message with static component.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		itx.SendReply(tempest.ResponseMessageData{
			Content: "Example message",
			Components: []tempest.MessageComponent{
				tempest.ActionRowComponent{
					Type: tempest.ACTION_ROW_COMPONENT_TYPE,
					Components: []tempest.InteractiveComponent{
						tempest.ButtonComponent{
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							CustomID: "button-hello",
							Style:    tempest.SECONDARY_BUTTON_STYLE,
							Label:    "Click me!",
						},
					},
				},
			},
		}, false, nil)
	},
}

// This function will be used at every button click, there's no max time limit.
func HelloStatic(itx tempest.ComponentInteraction) {
	err := itx.AcknowledgeWithMessage(tempest.ResponseMessageData{
		Content: fmt.Sprintf("Hello <@%d>!", itx.Member.User.ID),
	}, false)

	if err != nil {
		log.Println("failed to acknowledge static component", err)
		return
	}
}
