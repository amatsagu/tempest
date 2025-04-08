package command

import (
	"fmt"

	tempest "github.com/amatsagu/tempest"
)

type AddSlashCommand struct{}

func (cmd AddSlashCommand) Data() tempest.Command {
	return tempest.Command{
		Name:        "add",
		Description: "Adds 2 numbers.",
		Options: []tempest.CommandOption{
			{
				Name:        "first",
				Description: "First number to add.",
				Type:        tempest.NUMBER_OPTION_TYPE,
				Required:    true,
			},
			{
				Name:        "second",
				Description: "Second number to add.",
				Type:        tempest.NUMBER_OPTION_TYPE,
				Required:    true,
			},
		},
	}
}

func (cmd AddSlashCommand) AutoCompleteHandler(itx *tempest.CommandInteraction) []tempest.Choice {
	return nil
}

func (cmd AddSlashCommand) CommandHandler(itx tempest.CommandInteraction) {
	a, _ := itx.GetOptionValue("first")
	b, _ := itx.GetOptionValue("second")
	// ^ There's no need to check second bool value if option exists because they come from required field.

	af := a.(float64)
	bf := b.(float64)

	itx.SendLinearReply(fmt.Sprintf("Result: %.2f", af+bf), false)
}
