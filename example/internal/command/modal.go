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
			Components: []tempest.ModalComponent{
				tempest.LabelComponent{
					Type:  tempest.LABEL_COMPONENT_TYPE,
					Label: "Tell me something you like",
					Component: tempest.TextInputComponent{
						Type:     tempest.TEXT_INPUT_COMPONENT_TYPE,
						CustomID: "example-test-input",
						Style:    tempest.SHORT_TEXT_INPUT_STYLE,
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
	if row, ok := itx.Data.Components[0].(tempest.LabelComponent); ok {
		if textInput, ok := row.Component.(tempest.TextInputComponent); ok {
			// This if check is technically redundant since we already know what the 1st text input field contains, but
			// illustrates that Discord will send component Custom IDs back verbatim (so you could use them to encode state).
			if textInput.CustomID == "example-test-input" {
				value = textInput.Value
			}
		}
	}

	// NB: The below commented code is an alternate means of retrieving the component's value using a builtin helper.
	// FindInteractiveComponent is mostly useful for larger, more complex component trees where manually checking individual components would be infeasible;
	// users with simpler component hierarchies should prefer the manual approach for higher performance.

	// if textInput, ok := tempest.FindInteractiveComponent(
	// 	itx.Data.Components,
	// 	func(cmp tempest.TextInputComponent) bool { return cmp.CustomID == "example-test-input" },
	// ); ok {
	// 	value = textInput.Value
	// }

	if value == "" {
		itx.AcknowledgeWithLinearMessage("Oh, how about trying pizza?", false)
	} else {
		itx.AcknowledgeWithLinearMessage("I see! So you like "+value+"...", false)
	}
}
