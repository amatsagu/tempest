package command

import (
	"fmt"

	tempest "github.com/amatsagu/tempest"
)

var AutoComplete tempest.Command = tempest.Command{
	Name:        "auto-complete",
	Description: "Shows example of auto-complete.",
	Options: []tempest.CommandOption{
		{
			Name:         "suggestion",
			Description:  "Selector for one of the options.",
			Type:         tempest.STRING_OPTION_TYPE,
			Required:     true,
			AutoComplete: true,
		},
	},
	AutoCompleteHandler: func(itx tempest.CommandInteraction) []tempest.CommandOptionChoice {
		examples := []tempest.CommandOptionChoice{
			{
				Name:  "Select first option!",
				Value: "first option",
			},
			{
				Name:  "Select second option!",
				Value: "second option",
			},
			{
				Name:  "Or maybe third?",
				Value: "third option",
			},
		}

		fieldName, value := itx.GetFocusedValue()
		fmt.Printf("Received autocomplete field \"%s\": %+v\n", fieldName, value)

		// Do some logic that returns slice of choices...

		return examples
	},
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		value, _ := itx.GetOptionValue("suggestion")
		itx.SendLinearReply("Received: "+value.(string), false)
	},
}
