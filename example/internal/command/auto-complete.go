package command

import (
	tempest "github.com/Amatsagu/Tempest"
)

var AutoComplete tempest.Command = tempest.Command{
	Name:        "auto-complete",
	Description: "Adds 2 numbers.",
	Options: []tempest.CommandOption{
		{
			Name:         "suggestion",
			Description:  "First number to add.",
			Type:         tempest.STRING_OPTION_TYPE,
			Required:     true,
			AutoComplete: true,
		},
	},
	AutoCompleteHandler: func(itx tempest.AutoCompleteInteraction) []tempest.Choice {
		examples := []tempest.Choice{
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

		// Do some logic that returns slice of choices...

		return examples
	},
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		value, _ := itx.GetOptionValue("suggestion")
		itx.SendLinearReply("Received: "+value.(string), false)
	},
}
