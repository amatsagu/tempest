package command

import (
	"log"
	"strconv"
	"time"

	tempest "github.com/amatsagu/tempest"
)

var Dynamic tempest.Command = tempest.Command{
	Name:        "dynamic",
	Description: "Same as static but awaits button (impurity).",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		uniqueButtonID := "button-hello-dynamic-" + itx.ID.String()

		msg := tempest.ResponseMessageData{
			Content: "Click below button *(only you can do it)*:",
			Components: []tempest.ComponentRow{
				{
					Type: tempest.ROW_COMPONENT_TYPE,
					Components: []tempest.Component{
						{
							CustomID: uniqueButtonID,
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							Style:    uint8(tempest.SECONDARY_BUTTON_STYLE),
							Label:    "0",
						},
					},
				},
			},
		}

		itx.SendReply(msg, false, nil)
		signalChannel, stopFunction, err := itx.Client.AwaitComponent([]string{uniqueButtonID}, time.Minute*1)
		if err != nil {
			log.Println("failed to create component listener", err)
			itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to create component listener."}, false)
			return
		}

		var counter uint64 = 0
		for {
			citx := <-signalChannel
			// Default, "0 value" struct is returned when something fails or it gets timed out.
			// It's ineffcient and annoying to compare whole structs so instead just check for string Token that is always defined for any valid interaction.
			if citx != nil {
				log.Println("component listener channel closed")
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
				log.Println("failed to edit response", err)
				itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to edit response."}, false)
				return
			}
		}
	},
}
