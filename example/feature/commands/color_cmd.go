package commands

import (
	"context"
	"strings"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
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

func (c *colorCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "color",
		Description: "Show a color, with autocomplete on the name.",
		Options: []discord.CommandOption{
			{
				Type:         discord.OptionString,
				Name:         "name",
				Description:  "Which color to show.",
				Required:     true,
				Autocomplete: true,
			},
		},
	}
}

func (c *colorCommand) autocomplete(_ context.Context, itx *discord.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	typed, _ := itx.Data.StringOption("name")
	typed = strings.ToLower(typed)

	choices := make([]discord.CommandOptionChoice, 0, len(c.names))
	for _, name := range c.names {
		if strings.HasPrefix(name, typed) {
			choices = append(choices, discord.CommandOptionChoice{Name: name, Value: name})
		}
	}
	return choices, nil
}

func (c *colorCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	name, _ := itx.Data.StringOption("name")

	card := builder.Embed(name, "Here is your color.")
	if value, ok := c.values[name]; ok {
		card.Color = value
	}

	return itx.Reply(discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false)
}
