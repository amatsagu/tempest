package commands

import (
	"context"
	"strings"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"

	"github.com/amatsagu/tempest-example/util"
)

type colorCommand struct {
	values map[string]uint32
	names  []string // ordered for deterministic autocomplete
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
		Options: []discord.CommandOption{
			{
				Type:         discord.StringOptionType,
				Name:         "name",
				Description:  "Which color to show.",
				Required:     true,
				AutoComplete: true,
			},
		},
		AutoCompleteHandler: c.autocomplete,
		SlashCommandHandler: c.handle,
	}
}

func (c *colorCommand) autocomplete(_ context.Context, itx *tempest.CommandInteraction) ([]discord.CommandOptionChoice, error) {
	typed := strings.ToLower(util.FocusedString(itx))

	choices := make([]discord.CommandOptionChoice, 0, len(c.names))
	for _, name := range c.names {
		if strings.HasPrefix(name, typed) {
			choices = append(choices, discord.CommandOptionChoice{Name: name, Value: name})
		}
	}
	return choices, nil
}

func (c *colorCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	name, _ := util.StringOption(itx, "name")

	card := util.Embed(name, "Here is your color.")
	if value, ok := c.values[name]; ok {
		card.Color = value
	}

	return itx.SendReply(ctx, discord.ResponseMessageData{Embeds: []discord.Embed{card}}, false, nil)
}
