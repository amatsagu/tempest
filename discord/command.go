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

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-structure
type Command struct {
	DescriptionLocalizations map[Locale]string            `json:"description_localizations,omitzero"`
	DefaultMemberPermissions *Permissions                 `json:"default_member_permissions,string,omitzero"`
	DefaultPermission        *bool                        `json:"default_permission,omitzero"`
	DMPermission             *bool                        `json:"dm_permission,omitzero"`
	NameLocalizations        map[Locale]string            `json:"name_localizations,omitzero"`
	Description              string                       `json:"description,omitzero"`
	Name                     string                       `json:"name"`
	Options                  []CommandOption              `json:"options,omitzero"`
	IntegrationTypes         []ApplicationIntegrationType `json:"integration_types,omitzero"`
	Contexts                 []InteractionContextType     `json:"contexts,omitzero"`
	GuildID                  GuildID                      `json:"guild_id,omitzero"`
	Version                  Snowflake                    `json:"version,omitzero"`
	ApplicationID            AppID                        `json:"application_id,omitzero"`
	ID                       CommandID                    `json:"id,omitzero"`
	// Type defaults to CommandChatInput when left at its zero value.
	Type    CommandType        `json:"type,omitzero"`
	Handler CommandHandlerType `json:"handler,omitzero"`
	NSFW    bool               `json:"nsfw,omitzero"`
}

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-structure
type CommandOption struct {
	MaxLength                *int                  `json:"max_length,omitzero"`
	DescriptionLocalizations map[Locale]string     `json:"description_localizations,omitzero"`
	MaxValue                 *float64              `json:"max_value,omitzero"`
	NameLocalizations        map[Locale]string     `json:"name_localizations,omitzero"`
	MinValue                 *float64              `json:"min_value,omitzero"`
	MinLength                *int                  `json:"min_length,omitzero"`
	Name                     string                `json:"name"`
	Description              string                `json:"description,omitzero"`
	FileTypes                []string              `json:"file_types,omitzero"`
	ChannelTypes             []ChannelType         `json:"channel_types,omitzero"`
	Options                  []CommandOption       `json:"options,omitzero"`
	Choices                  []CommandOptionChoice `json:"choices,omitzero"`
	Type                     OptionType            `json:"type"`
	Required                 bool                  `json:"required,omitzero"`
	Autocomplete             bool                  `json:"autocomplete,omitzero"`
}
