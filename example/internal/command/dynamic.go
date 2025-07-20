package command

import (
	"context"
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
			Components: []tempest.LayoutComponent{
				tempest.ActionRowComponent{
					Type: tempest.ACTION_ROW_COMPONENT_TYPE,
					Components: []tempest.InteractiveComponent{
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		signalChan, cleanupFunc, err := itx.Client.AwaitComponent([]string{uniqueButtonID})
		if err != nil {
			log.Println("failed to create component listener:", err)
			itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to create component listener."}, false)
			return
		}
		defer cleanupFunc()

		var counter uint64 = 0

	counterLoop: // At least in our case, use label to clearly exit infinite loop where appropriate.
		for {
			select {
			case citx, open := <-signalChan:
				if !open {
					break counterLoop
				}

				if citx.Member.User.ID != itx.Member.User.ID {
					continue
				}

				counter++
				if row, ok := msg.Components[0].(tempest.ActionRowComponent); ok {
					if btn, ok := row.Components[0].(tempest.ButtonComponent); ok {
						btn.Label = strconv.FormatUint(counter, 10)
						row.Components[0] = btn
						msg.Components[0] = row
					}
				}

				err = itx.EditReply(msg, false)
				if err != nil {
					log.Println("failed to edit response", err)
					itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to edit response."}, false)
					return
				}

				//break counterLoop
			case <-ctx.Done():
				// timeout or cancellation (we already defer cleanup higher)

				err = itx.EditReply(tempest.ResponseMessageData{
					Content: "Reached timeout or cancellation of context",
				}, false)

				if err != nil {
					log.Println("failed to edit response", err)
					itx.SendFollowUp(tempest.ResponseMessageData{Content: "Failed to edit response."}, false)
				}

				break counterLoop
			}
		}

		// Any code after for loop would run just fine...
	},
}
