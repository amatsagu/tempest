package command

import (
	"log"
	"strconv"
	"time"

	tempest "github.com/amatsagu/tempest"
)

// Tip: This example would be nearly identical for handling dynamic modals.

var Dynamic tempest.Command = tempest.Command{
	Name:        "dynamic",
	Description: "Same as static but awaits button (impurity).",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		uniqueButtonID := "btn-dynamic-" + itx.ID.String()

		msg := tempest.ResponseMessageData{
			Content: "Click below button *(only you can do it)*:",
			Components: []tempest.MessageComponent{
				tempest.ActionRowComponent{
					Type: tempest.ACTION_ROW_COMPONENT_TYPE,
					Components: []tempest.ActionRowChildComponent{
						tempest.ButtonComponent{
							Type:     tempest.BUTTON_COMPONENT_TYPE,
							Style:    tempest.SECONDARY_BUTTON_STYLE,
							Label:    "0",
							CustomID: uniqueButtonID,
						},
					},
				},
			},
		}

		itx.SendReply(msg, false, nil)

		// In real world - you'll probably have some sort of master context instead default background to gracefully control app/bot lifecycles.
		var counter uint64 = 0

		err := itx.BaseClient.AwaitComponent([]string{uniqueButtonID}, time.Minute*2, func(citx *tempest.ComponentInteraction) bool {
			if citx.Member.User.ID != itx.Member.User.ID {
				return true // Ignore click from other user, keep listening
			}

			counter++
			if row, ok := msg.Components[0].(tempest.ActionRowComponent); ok {
				if btn, ok := row.Components[0].(tempest.ButtonComponent); ok {
					btn.Label = strconv.FormatUint(counter, 10)
					row.Components[0] = btn
					msg.Components[0] = row
				}
			}

			err := itx.EditReply(msg, false)
			if err != nil {
				log.Println("failed to edit response", err)
				itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to edit response."}, false)
				return false // Stop listening on error
			}

			return true // Continue listening
		}, func() {
			// This runs when timeout is reached
			err := itx.EditReply(tempest.ResponseMessageData{
				Content: "Reached timeout (button disabled).",
			}, false)

			if err != nil {
				log.Println("failed to edit response", err)
				itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to edit response."}, false)
			}
		})

		if err != nil {
			log.Println("failed to create component listener:", err)
			itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to create component listener."}, false)
			return
		}

		// Any code after for loop would run just fine...
	},
}
