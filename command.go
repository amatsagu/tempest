package tempest

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-types
type CommandType uint8

const (
	CHAT_INPUT_COMMAND_TYPE          CommandType = iota + 1 // Slash command (default option)
	USER_COMMAND_TYPE                                       // Mounted to user/member profile
	MESSAGE_COMMAND_TYPE                                    // Mounted to text message
	PRIMARY_ENTRY_POINT_COMMAND_TYPE                        // An UI-based command that represents the primary way to invoke an app's Activity
)

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-type
type OptionType uint8

const (
	SUB_OPTION_TYPE               OptionType = iota + 1
	SUB_COMMAND_GROUP_OPTION_TYPE            // NOT SUPPORTED BY LIBRARY
	STRING_OPTION_TYPE
	INTEGER_OPTION_TYPE
	BOOLEAN_OPTION_TYPE
	USER_OPTION_TYPE
	CHANNEL_OPTION_TYPE
	ROLE_OPTION_TYPE
	MENTIONABLE_OPTION_TYPE
	NUMBER_OPTION_TYPE
	ATTACHMENT_OPTION_TYPE
)

// https://discord.com/developers/docs/resources/application#application-object-application-integration-types
type ApplicationIntegrationType uint8

const (
	GUILD_INSTALL ApplicationIntegrationType = iota
	USER_INSTALL
)

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-entry-point-command-handler-types
type CommandHandlerType uint8

const (
	APP_COMMAND_HANDLER CommandHandlerType = iota + 1
	DISCORD_LAUNCH_ACTIVITY_COMMAND_HANDLER
)

// https://discord.com/developers/docs/reference#locales
type Language string

const (
	DANISH_LANGUAGE         Language = "da"
	GERMAN_LANGUAGE         Language = "de"
	ENGLISH_UK_LANGUAGE     Language = "en-GB"
	ENGLISH_US_LANGUAGE     Language = "en-US"
	SPANISH_LANGUAGE        Language = "es-ES"
	FRENCH_LANGUAGE         Language = "fr"
	CROATIAN_LANGUAGE       Language = "hr"
	ITALIAN_LANGUAGE        Language = "it"
	LITHUANIAN_LANGUAGE     Language = "lt"
	HUNGARIAN_LANGUAGE      Language = "hu"
	DUTCH_LANGUAGE          Language = "nl"
	NORWEGIAN_LANGUAGE      Language = "no"
	POLISH_LANGUAGE         Language = "pl"
	PORTUGUESE_BR_LANGUAGE  Language = "pt-BR"
	ROMANIAN_LANGUAGE       Language = "ro"
	FINNISH_LANGUAGE        Language = "fi"
	SWEDISH_LANGUAGE        Language = "sv-SE"
	VIETNAMESE_LANGUAGE     Language = "vi"
	TURKISH_LANGUAGE        Language = "tr"
	CHECH_LANGUAGE          Language = "cs"
	GREEK_LANGUAGE          Language = "el"
	BULGARIAN_LANGUAGE      Language = "bg"
	RUSSIAN_LANGUAGE        Language = "ru"
	UKRAINIAN_LANGUAGE      Language = "uk"
	HINDI_LANGUAGE          Language = "hi"
	THAI_LANGUAGE           Language = "th"
	CHINESE_CHINA_LANGUAGE  Language = "zh-CN"
	JAPANESE_LANGUAGE       Language = "ja"
	CHINESE_TAIWAN_LANGUAGE Language = "zh-TW"
	KOREAN_LANGUAGE         Language = "ko"
)

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure
type Command struct {
	ID                       Snowflake                    `json:"-"` // It's not needed on app side to work.
	Type                     CommandType                  `json:"type,omitempty"`
	ApplicationID            Snowflake                    `json:"application_id"`
	GuildID                  Snowflake                    `json:"guild_id,omitempty"`
	Name                     string                       `json:"name"`
	NameLocalizations        map[Language]string          `json:"name_localizations,omitzero"`
	Description              string                       `json:"description"`
	DescriptionLocalizations map[Language]string          `json:"description_localizations,omitzero"`
	Options                  []CommandOption              `json:"options,omitzero"`
	RequiredPermissions      PermissionFlags              `json:"default_member_permissions,string,omitempty"` // Set of permissions represented as a bit set that are required from user/member to use command. Set it to 0 to make command unavailable for regular members (guild administrators still can use it).
	IntegrationTypes         []ApplicationIntegrationType `json:"integration_types,omitzero"`
	Contexts                 []InteractionContextType     `json:"contexts,omitzero"` // Interaction context(s) where the command can be used, only for globally-scoped commands. By default, all interaction context types included for new commands.
	NSFW                     bool                         `json:"nsfw,omitempty"`    // https://discord.com/developers/docs/interactions/application-commands#agerestricted-commands
	Version                  Snowflake                    `json:"version,omitempty"` // Autoincrementing version identifier updated during substantial record changes.
	Handler                  CommandHandlerType           `json:"handler,omitempty"`

	AutoCompleteHandler func(itx CommandInteraction) []CommandOptionChoice `json:"-"` // Custom handler for auto complete interactions. It's a Tempest specific field.
	SlashCommandHandler func(itx *CommandInteraction)                      `json:"-"` // Custom handler for slash command interactions. It's a Tempest specific field. It receives pointer to CommandInteraction as it's being used with pre & post client hooks.
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-structure
type CommandOption struct {
	Type                     OptionType            `json:"type"`
	Name                     string                `json:"name"`
	NameLocalizations        map[Language]string   `json:"name_localizations,omitzero"`
	Description              string                `json:"description"`
	DescriptionLocalizations map[Language]string   `json:"description_localizations,omitzero"`
	Required                 bool                  `json:"required,omitempty"`
	Choices                  []CommandOptionChoice `json:"choices,omitzero"`
	Options                  []CommandOption       `json:"options,omitzero"`
	ChannelTypes             []ChannelType         `json:"channel_types,omitzero"`
	MinValue                 float64               `json:"min_value,omitempty"`
	MaxValue                 float64               `json:"max_value,omitempty"`
	MinLength                uint16                `json:"min_length,omitempty"`
	MaxLength                uint16                `json:"max_length,omitempty"`
	AutoComplete             bool                  `json:"autocomplete,omitempty"` // Required to be = true if you want to catch it later in auto complete handler.
}
