package command

import (
	"fmt"

	"github.com/amatsagu/tempest/api"
)

var Add api.Command = api.Command{
	Name:        "add",
	Description: "Adds 2 numbers.",
	Options: []api.CommandOption{
		{
			Name:        "first",
			Description: "First number to add.",
			Type:        api.INTEGER_OPTION_TYPE,
			Required:    true,
		},
		{
			Name:        "second",
			Description: "Second number to add.",
			Type:        api.INTEGER_OPTION_TYPE,
			Required:    true,
		},
	},
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		var first, second int32

		for _, option := range itx.Data.Options {

			// Discord returns numbers as floats no matter the preference (I guess json thing)

			switch option.Name {
			case "first":
				first = int32(option.Value.(float64))
			case "second":
				first = int32(option.Value.(float64))
			}
		}

		itx.SendLinearReply(fmt.Sprintf("Result: %d", first+second), false)
	},
}
