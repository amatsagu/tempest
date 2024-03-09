package command

import (
	"log/slog"

	tempest "github.com/Amatsagu/Tempest"
)

var Modal tempest.Command = tempest.Command{
	Name:        "modal",
	Description: "Sends example message with static modal.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		err := itx.SendModal(tempest.ResponseModalData{
			CustomID: "my-modal",
			Title:    "Hello modal!",
			Components: []tempest.ComponentRow{
				{
					Type: tempest.ROW_COMPONENT_TYPE,
					Components: []tempest.Component{
						{
							CustomID: "example-text-input",
							Type:     tempest.TEXT_INPUT_COMPONENT_TYPE,
							Style:    uint8(tempest.SHORT_TEXT_INPUT_STYLE),
							Label:    "Tell me something you like",
						},
					},
				},
			},
		})

		if err != nil {
			slog.Error("failed to send modal", err)
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
