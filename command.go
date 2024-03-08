package tempest

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-types
type CommandType uint8

const (
	CHAT_INPUT_COMMAND_TYPE CommandType = iota + 1 // Default option, a slash command.
	USER_COMMAND_TYPE                              // Mounted to user/member profile.
	MESSAGE_COMMAND_TYPE                           // Mounted to text message.
)

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-type
type OptionType uint8

const (
	SUB_OPTION_TYPE OptionType = iota + 1
	_                          // OPTION_SUB_COMMAND_GROUP (not supported)
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
	ID                       Snowflake           `json:"-"` // Omit in json parsing for now because it was breaking Client#commandParse.
	Type                     CommandType         `json:"type,omitempty"`
	ApplicationID            Snowflake           `json:"application_id"`
	GuildID                  Snowflake           `json:"guild_id,omitempty"`
	Name                     string              `json:"name"`
	NameLocalizations        map[Language]string `json:"name_localizations,omitempty"` // https://discord.com/developers/docs/reference#locales
	Description              string              `json:"description"`
	DescriptionLocalizations map[Language]string `json:"description_localizations,omitempty"`
	Options                  []CommandOption     `json:"options,omitempty"`
	DefaultMemberPermissions uint64              `json:"default_member_permissions,string,omitempty"` // Set of permissions represented as a bit set. Set it to 0 to make command unavailable for regular members.
	AvailableInDM            bool                `json:"dm_permission,omitempty"`                     // Whether command should be visible (usable) from private, dm channels. Works only for global commands!
	NSFW                     bool                `json:"nsfw,omitempty"`                              // https://discord.com/developers/docs/interactions/application-commands#agerestricted-commands
	Version                  Snowflake           `json:"version,omitempty"`                           // Autoincrementing version identifier updated during substantial record changes

	AutoCompleteHandler func(itx CommandInteraction) []Choice `json:"-"` // Custom handler for auto complete interactions. It's a Tempest specific field.
	SlashCommandHandler func(itx *CommandInteraction)         `json:"-"` // Custom handler for slash command interactions. It's a Tempest specific field. It receives pointer to CommandInteraction as it's being used with pre & post client hooks.
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-structure
type CommandOption struct {
	Type                     OptionType          `json:"type"`
	Name                     string              `json:"name"`
	NameLocalizations        map[Language]string `json:"name_localizations,omitempty"` // https://discord.com/developers/docs/reference#locales
	Description              string              `json:"description"`
	DescriptionLocalizations map[Language]string `json:"description_localizations,omitempty"`
	Required                 bool                `json:"required,omitempty"`
	MinValue                 float64             `json:"min_value,omitempty"`
	MaxValue                 float64             `json:"max_value,omitempty"`
	MinLength                uint                `json:"min_length,omitempty"`
	MaxLength                uint                `json:"max_length,omitempty"`
	Options                  []CommandOption     `json:"options,omitempty"`
	ChannelTypes             []ChannelType       `json:"channel_types,omitempty"`
	Choices                  []Choice            `json:"choices,omitempty"`
	AutoComplete             bool                `json:"autocomplete,omitempty"` // Required to be = true if you want to catch it later in auto complete handler.
}
