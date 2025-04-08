package command

import (
	tempest "github.com/amatsagu/tempest"
)

type AutoCompleteSlashCommand struct{}

func (cmd AutoCompleteSlashCommand) Data() tempest.Command {
	return tempest.Command{
		Name:        "auto-complete",
		Description: "Sends text completion recommendations as user types.",
		Options: []tempest.CommandOption{
			{
				Name:         "suggestion",
				Description:  "Type to get recommendation.",
				Type:         tempest.STRING_OPTION_TYPE,
				Required:     true,
				AutoComplete: true,
			},
		},
	}
}

func (cmd AutoCompleteSlashCommand) AutoCompleteHandler(itx *tempest.CommandInteraction) []tempest.Choice {
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
}

func (cmd AutoCompleteSlashCommand) CommandHandler(itx tempest.CommandInteraction) {
	value, _ := itx.GetOptionValue("suggestion")
	itx.SendLinearReply("Received: "+value.(string), false)
}
