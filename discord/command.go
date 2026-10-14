package discord

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-types
type CommandType uint8

const (
	CommandChatInput CommandType = iota + 1
	CommandUser
	CommandMessage
	CommandPrimaryEntryPoint
)

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-type
type OptionType uint8

const (
	OptionSubCommand OptionType = iota + 1
	OptionSubCommandGroup
	OptionString
	OptionInteger
	OptionBoolean
	OptionUser
	OptionChannel
	OptionRole
	OptionMentionable
	OptionNumber
	OptionAttachment
)

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-entry-point-command-handler-types
type CommandHandlerType uint8

const (
	CommandHandlerApp CommandHandlerType = iota + 1
	CommandHandlerDiscordLaunchActivity
)

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-choice-structure
type CommandOptionChoice struct {
	NameLocalizations map[Locale]string `json:"name_localizations,omitzero"`
	Value             any               `json:"value"`
	Name              string            `json:"name"`
}
