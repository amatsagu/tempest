package commands

import (
	"strings"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

// colorCommand holds the known colors; names is a slice so autocomplete stays ordered.
type colorCommand struct {
	names  []string
	values map[string]uint32
}

func newColorCommand() *colorCommand {
	return &colorCommand{
		names: []string{"red", "green", "blue", "purple", "teal"},
		values: map[string]uint32{
			"red":    0xED4245,
			"green":  0x57F287,
			"blue":   0x5865F2,
			"purple": 0xEB459E,
			"teal":   0x1ABC9C,
		},
	}
}

func (c *colorCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:        "color",
		Description: "Show a color, with autocomplete on the name.",
		Options: []tempest.CommandOption{
			{
				Type:         tempest.STRING_OPTION_TYPE,
				Name:         "name",
				Description:  "Which color to show.",
				Required:     true,
				AutoComplete: true,
			},
		},
		AutoCompleteHandler: c.autocomplete,
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

func (c *colorCommand) autocomplete(itx tempest.CommandInteraction) []tempest.CommandOptionChoice {
	typed := focusedString(itx)
	typed = strings.ToLower(typed)

	choices := make([]tempest.CommandOptionChoice, 0, len(c.names))
	for _, name := range c.names {
		if strings.HasPrefix(name, typed) {
			choices = append(choices, tempest.CommandOptionChoice{Name: name, Value: name})
		}
	}
	return choices
}

func (c *colorCommand) handle(itx *tempest.CommandInteraction) error {
	name, _ := util.StringOption(itx, "name")

	card := util.Embed(name, "Here is your color.")
	if value, ok := c.values[name]; ok {
		card.Color = value
	}

	return itx.SendReply(tempest.ResponseMessageData{Embeds: []tempest.Embed{card}}, false, nil)
}

// focusedString reads the focused autocomplete value without the panic in
// GetFocusedValue for malformed payloads.
func focusedString(itx tempest.CommandInteraction) string {
	for _, option := range itx.Data.Options {
		if !option.Focused {
			continue
		}

		value, _ := option.Value.(string)
		return value
	}
	return ""
}
