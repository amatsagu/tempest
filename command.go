package tempest

import "github.com/amatsagu/tempest/discord"

// Command wraps [discord.Command] with Tempest's interaction handlers.
type Command struct {
	SlashCommandHandler func(itx *CommandInteraction)                              `json:"-"` // Custom handler for slash command interactions. It's a Tempest specific field. It receives pointer to CommandInteraction as it's being used with pre & post client hooks.
	AutoCompleteHandler func(itx CommandInteraction) []discord.CommandOptionChoice `json:"-"` // Custom handler for auto complete interactions. It's a Tempest specific field.
	discord.Command
}
