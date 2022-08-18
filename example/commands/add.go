package commands

import (
	"fmt"

	tempest "github.com/Amatsagu/Tempest"
)

var Add tempest.Command = tempest.Command{
	Name:        "add",
	Description: "Adds 2 numbers.",
	Options: []tempest.Option{
		{
			Name:        "first",
			Description: "First number to add.",
			Type:        tempest.OPTION_NUMBER,
			Required:    true,
		},
		{
			Name:        "second",
			Description: "Second number to add.",
			Type:        tempest.OPTION_NUMBER,
			Required:    true,
		},
	},
	SlashCommandHandler: func(itx tempest.CommandInteraction) {
		a, _ := itx.GetOptionValue("first")
		b, _ := itx.GetOptionValue("second")
		// ^ There's no need to check second bool value if option exists because we set them as required on lines 15 & 21.

		// A & B values are json numbers, make Go compiler see them as float64:
		af := a.(float64)
		bf := b.(float64)

		itx.SendLinearReply(fmt.Sprintf("Result: %.2f", af+bf), false)
	},
}
