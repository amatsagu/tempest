package command

import (
	"example-bot/internal/logger"
	"fmt"

	tempest "github.com/Amatsagu/Tempest"
)

var Static tempest.Command = tempest.Command{
	Name:        "static",
	Description: "Sends example message with static component.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		itx.SendReply(tempest.ResponseMessageData{
			Content: "Example message",
			Components: []*tempest.ComponentRow{
				{
					Type: tempest.ROW_COMPONENT_TYPE,
					Components: []*tempest.Component{
						{
							CustomID: "button-hello",
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							Style:    uint8(tempest.SECONDARY_BUTTON_STYLE),
							Label:    "Click me",
						},
					},
				},
			},
		}, false)
	},
}

// This function will be used at every button click, there's no max time limit.
func HelloStatic(itx tempest.ComponentInteraction) {
	err := itx.AcknowledgeWithMessage(tempest.ResponseMessageData{
		Content: fmt.Sprintf("Hello <@%d>!", itx.Member.User.ID),
	}, false)

	if err != nil {
		logger.Error.Println(err)
		logger.Warn.Printf("%+v\n", itx)
		return
	}
}
