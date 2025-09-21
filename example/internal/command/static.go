package command

import (
	"fmt"
	"log"
	"qord/api"
)

var Static api.Command = api.Command{
	Name:        "static",
	Description: "Sends example message with static component.",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		itx.SendReply(api.ResponseMessageData{
			Content: "Example message",
			Components: []api.LayoutComponent{
				api.ActionRowComponent{
					Type: api.ACTION_ROW_COMPONENT_TYPE,
					Components: []api.InteractiveComponent{
						api.ButtonComponent{
							Type:     api.BUTTON_COMPONENT_TYPE,
							CustomID: "button-hello",
							Style:    api.SECONDARY_BUTTON_STYLE,
							Label:    "Click me!",
						},
					},
				},
			},
		}, false, nil)
	},
}

// This function will be used at every button click, there's no max time limit.
func HelloStatic(itx api.ComponentInteraction) {
	err := itx.AcknowledgeWithMessage(api.ResponseMessageData{
		Content: fmt.Sprintf("Hello <@%d>!", itx.Member.User.ID),
	}, false)

	if err != nil {
		log.Println("failed to acknowledge static component", err)
		return
	}
}
