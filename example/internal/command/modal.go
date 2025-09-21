package command

import (
	"log"

	"github.com/amatsagu/tempest/api"
)

var Modal api.Command = api.Command{
	Name:        "modal",
	Description: "Sends example message with static modal.",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		err := itx.SendModal(api.ResponseModalData{
			CustomID: "my-modal",
			Title:    "Hello modal!",
			Components: []api.LayoutComponent{
				api.ActionRowComponent{
					Type: api.ACTION_ROW_COMPONENT_TYPE,
					Components: []api.InteractiveComponent{
						api.TextInputComponent{
							Type:     api.TEXT_INPUT_COMPONENT_TYPE,
							CustomID: "example-test-input",
							Style:    api.SHORT_TEXT_INPUT_STYLE,
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

func HelloModal(itx api.ModalInteraction) {
	var value string
	// if row, ok := itx.Data.Components[0].(api.ActionRowComponent); ok {
	// 	if textInput, ok := row.Components[0].(api.TextInputComponent); ok {
	//      // This if check is an example, not required in this example as it has only 1 component in total.
	// 		if textInput.CustomID == "example-test-input" {
	// 			value = textInput.Value
	// 		}
	// 	}
	// }

	// This is just an example that lib provides helped function to traverse component trees.
	// In case you work with interaction that you know has just 1-2 interactive components, use higher commented code for better performance.
	textInput, ok := api.FindInteractiveComponent(
		itx.Data.Components,
		func(cmp api.TextInputComponent) bool { return cmp.CustomID == "example-test-input" },
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
