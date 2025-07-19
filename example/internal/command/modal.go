package command

import (
	"log"

	tempest "github.com/amatsagu/tempest"
)

var Modal tempest.Command = tempest.Command{
	Name:        "modal",
	Description: "Sends example message with static modal.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		err := itx.SendModal(tempest.ResponseModalData{
			CustomID: "my-modal",
			Title:    "Hello modal!",
			Components: []tempest.LayoutComponent{
				tempest.ActionRowComponent{
					Type: tempest.ACTION_ROW_COMPONENT_TYPE,
					Components: []tempest.InteractiveComponent{
						tempest.TextInputComponent{
							Type:     tempest.TEXT_INPUT_COMPONENT_TYPE,
							CustomID: "example-test-input",
							Style:    tempest.SHORT_TEXT_INPUT_STYLE,
							Label:    "Tell me something you like",
						},
					},
				},
			},
		})

		if err != nil {
			log.Println("failed to send modal", err)
		}
	},
}

func HelloModal(itx tempest.ModalInteraction) {
	value := itx.GetInputValue("example-text-input")
	if value == "" {
		itx.AcknowledgeWithLinearMessage("Oh, how about trying pizza?", false)
	} else {
		itx.AcknowledgeWithLinearMessage("I see! So you like "+value+"...", false)
	}
}
