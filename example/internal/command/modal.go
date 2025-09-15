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
	var value string
	// if row, ok := itx.Data.Components[0].(tempest.ActionRowComponent); ok {
	// 	if textInput, ok := row.Components[0].(tempest.TextInputComponent); ok {
	//      // This if check is an example, not required in this example as it has only 1 component in total.
	// 		if textInput.CustomID == "example-test-input" {
	// 			value = textInput.Value
	// 		}
	// 	}
	// }

	// This is just an example that lib provides helped function to traverse component trees.
	// In case you work with interaction that you know has just 1-2 interactive components, use higher commented code for better performance.
	textInput, ok := tempest.FindInteractiveComponent(
		itx.Data.Components,
		func(cmp tempest.TextInputComponent) bool { return cmp.CustomID == "example-test-input" },
	)

	if ok {
		value = textInput.Value
	}

	if value == "" {
		itx.AcknowledgeWithLinearMessage("Oh, how about trying pizza?", false)
	} else {
		itx.AcknowledgeWithLinearMessage("I see! So you like "+value+"...", false)
	}
}
