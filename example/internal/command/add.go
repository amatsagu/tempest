package command

import (
	"fmt"

	tempest "github.com/amatsagu/tempest"
)

var Add tempest.Command = tempest.Command{
	Name:        "add",
	Description: "Adds 2 numbers.",
	Options: []tempest.CommandOption{
		{
			Name:        "first",
			Description: "First number to add.",
			Type:        tempest.INTEGER_OPTION_TYPE,
			Required:    true,
		},
		{
			Name:        "second",
			Description: "Second number to add.",
			Type:        tempest.INTEGER_OPTION_TYPE,
			Required:    true,
		},
	},
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		a, _ := itx.GetOptionValue("first")
		b, _ := itx.GetOptionValue("second")
		// ^ There's no need to check second bool value if option exists because we set them as required on lines 15 & 21.

		// A & B values are json numbers (f32), make Go compiler see them as float64 and then cast to integers:
		af := int32(a.(float64))
		bf := int32(b.(float64))

		itx.SendLinearReply(fmt.Sprintf("Result: %d", af+bf), false)
	},
}
