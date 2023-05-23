package command

import (
	"example-bot/logger"
	"fmt"
	"strconv"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

var Dynamic tempest.Command = tempest.Command{
	Name:        "dynamic",
	Description: "Same as static but awaits button (impurity).",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		msg := tempest.ResponseMessageData{
			Content: "Click below burtton *(only you can do it)*:",
			Components: []*tempest.ComponentRow{
				{
					Type: tempest.ROW_COMPONENT_TYPE,
					Components: []*tempest.Component{
						{
							CustomID: "button-hello-dynamic",
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							Style:    uint8(tempest.SECONDARY_BUTTON_STYLE),
							Label:    "0",
						},
					},
				},
			},
		}

		itx.SendReply(msg, false)
		signalChannel, stopFunction, err := itx.Client.AwaitComponent([]string{"button-hello-dynamic"}, time.Minute*1)
		if err != nil {
			logger.Error.Println(err)
			itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to create component listener."}, false)
			return
		}

		interaction := <-signalChannel
		err = interaction.AcknowledgeWithMessage(tempest.ResponseMessageData{
			Content: fmt.Sprintf("Hello <@%d>!", interaction.Member.User.ID),
		}, false)

		if err != nil {
			panic(err)
		}

		stopFunction()

		var counter uint64 = 0
		for {
			citx := <-signalChannel
			if citx == nil {
				stopFunction()
				return
			}

			if citx.Member.User.ID != itx.Member.User.ID {
				continue
			}

			counter++
			msg.Components[0].Components[0].Label = strconv.FormatUint(counter, 10)
			err = itx.EditReply(msg, false)
			if err != nil {
				logger.Error.Println(err)
				itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to edit response."}, false)
				return
			}
		}
	},
}
