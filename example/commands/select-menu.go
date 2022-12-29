package commands

import (
	"fmt"
	"strings"
	"time"

	tempest "github.com/Amatsagu/Tempest"
)

var SelectMenu tempest.Command = tempest.Command{
	Name:        "select-menu",
	Description: "Creates message with example select menu.",
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		handlerId := itx.ID.String()

		itx.SendReply(tempest.ResponseData{
			Content: "Example message",
			Components: []*tempest.Component{
				{
					Type: tempest.COMPONENT_ROW,
					Components: []*tempest.Component{
						{
							CustomID:    handlerId,
							Type:        tempest.COMPONENT_SELECT_MENU,
							Placeholder: "Choose a class (or 2 classes)",
							MinValues:   1,
							MaxValues:   2,
							Options: []*tempest.SelectMenuOption{
								{
									Label:       "Warrior",
									Description: "Nice and classy",
									Value:       "warrior",
								},
								{
									Label:       "Rogue",
									Description: "Sneak n stab",
									Value:       "rogue",
								},
								{
									Label:       "Mage",
									Description: "Turn 'em into a sheep",
									Value:       "mage",
								},
							},
						},
					},
				},
			},
		}, false)

		channel, stopFunction := itx.Client.AwaitComponent([]string{handlerId}, time.Minute*2)
		for {
			citx := <-channel
			if citx == nil {
				itx.SendFollowUp(tempest.ResponseData{Content: "Terminated listener (timeout)."}, false)
				break
			}

			// HOW TO FIND VAlUES?
			// Interaction.Data.Values contains all values in string form. In case it was for example channel or role - you'll receive only its numeric id.
			// Check Interaction.Data.Resolved.Channels and Interaction.Data.Resolved.Roles respectively to find their struct slices.

			response := fmt.Sprintf(`Member "%s" selected %s!`, citx.Member.User.Username, strings.Join(citx.Data.Values, ", "))
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
