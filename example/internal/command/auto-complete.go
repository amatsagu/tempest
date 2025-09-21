package command

import "github.com/amatsagu/qord/api"

var AutoComplete api.Command = api.Command{
	Name:        "auto-complete",
	Description: "Shows example of auto-complete.",
	Options: []api.CommandOption{
		{
			Name:         "suggestion",
			Description:  "Selector for one of the options.",
			Type:         api.STRING_OPTION_TYPE,
			Required:     true,
			AutoComplete: true,
		},
	},
	AutoCompleteHandler: func(itx api.CommandInteraction) []api.CommandOptionChoice {
		examples := []api.CommandOptionChoice{
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
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		value, _ := itx.GetOptionValue("suggestion")
		itx.SendLinearReply("Received: "+value.(string), false)
	},
}
