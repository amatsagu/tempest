package discord

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-types
type CommandType uint8

const (
	ChatInputCommandType         CommandType = iota + 1 // Slash command (default option)
	UserCommandType                                     // Mounted to user/member profile
	MessageCommandType                                  // Mounted to text message
	PrimaryEntryPointCommandType                        // An UI-based command that represents the primary way to invoke an app's Activity
)

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-type
type OptionType uint8

const (
	SubCommandOptionType      OptionType = iota + 1
	SubCommandGroupOptionType            // NOT OFFICIALLY SUPPORTED BY LIBRARY
	StringOptionType
	IntegerOptionType
	BooleanOptionType
	UserOptionType
	ChannelOptionType
	RoleOptionType
	MentionableOptionType
	NumberOptionType
	AttachmentOptionType
)

// https://docs.discord.com/developers/resources/application#application-object-application-integration-types
type ApplicationIntegrationType uint8

const (
	GuildInstall ApplicationIntegrationType = iota
	UserInstall
)

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-entry-point-command-handler-types
type CommandHandlerType uint8

const (
	AppCommandHandler CommandHandlerType = iota + 1
	DiscordLaunchActivityCommandHandler
)

// https://docs.discord.com/developers/reference#locales
type Language string

const (
	DanishLanguage        Language = "da"
	GermanLanguage        Language = "de"
	EnglishUKLanguage     Language = "en-GB"
	EnglishUSLanguage     Language = "en-US"
	SpanishLanguage       Language = "es-ES"
	FrenchLanguage        Language = "fr"
	CroatianLanguage      Language = "hr"
	ItalianLanguage       Language = "it"
	LithuanianLanguage    Language = "lt"
	HungarianLanguage     Language = "hu"
	DutchLanguage         Language = "nl"
	NorwegianLanguage     Language = "no"
	PolishLanguage        Language = "pl"
	PortugueseBRLanguage  Language = "pt-BR"
	RomanianLanguage      Language = "ro"
	FinnishLanguage       Language = "fi"
	SwedishLanguage       Language = "sv-SE"
	VietnameseLanguage    Language = "vi"
	TurkishLanguage       Language = "tr"
	CzechLanguage         Language = "cs"
	GreekLanguage         Language = "el"
	BulgarianLanguage     Language = "bg"
	RussianLanguage       Language = "ru"
	UkrainianLanguage     Language = "uk"
	HindiLanguage         Language = "hi"
	ThaiLanguage          Language = "th"
	ChineseChinaLanguage  Language = "zh-CN"
	JapaneseLanguage      Language = "ja"
	ChineseTaiwanLanguage Language = "zh-TW"
	KoreanLanguage        Language = "ko"
)

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-structure
type Command struct {
	DescriptionLocalizations map[Language]string          `json:"description_localizations,omitzero"`
	NameLocalizations        map[Language]string          `json:"name_localizations,omitzero"`
	Description              string                       `json:"description,omitzero"`
	Name                     string                       `json:"name"`
	Options                  []CommandOption              `json:"options,omitzero"`
	IntegrationTypes         []ApplicationIntegrationType `json:"integration_types,omitzero"`
	Contexts                 []InteractionContextType     `json:"contexts,omitzero"` // Interaction context(s) where the command can be used, only for globally-scoped commands. By default, all interaction context types included for new commands.
	GuildID                  GuildID                      `json:"guild_id,omitzero"`
	ID                       CommandID                    `json:"-"`                                          // It's not needed on app side to work.
	RequiredPermissions      PermissionFlags              `json:"default_member_permissions,string,omitzero"` // Bitwise permissions required to use this command (omitted when 0; use AdministratorPermissionFlag to restrict to administrators).
	Version                  Snowflake                    `json:"version,omitzero"`                           // Autoincrementing version identifier updated during substantial record changes.
	ApplicationID            AppID                        `json:"application_id,omitzero"`
	NSFW                     bool                         `json:"nsfw,omitzero"` // https://docs.discord.com/developers/interactions/application-commands#age-restricted-commands
	Handler                  CommandHandlerType           `json:"handler,omitzero"`
	Type                     CommandType                  `json:"type,omitzero"`
}

// https://docs.discord.com/developers/interactions/application-commands#application-command-object-application-command-option-structure
type CommandOption struct {
	MinValue                 *float64            `json:"min_value,omitzero"`
	NameLocalizations        map[Language]string `json:"name_localizations,omitzero"`
	MaxValue                 *float64            `json:"max_value,omitzero"`
	DescriptionLocalizations map[Language]string `json:"description_localizations,omitzero"`
	Description              string              `json:"description"`
	Name                     string              `json:"name"`
	Options                  []CommandOption     `json:"options,omitzero"`
	ChannelTypes             []ChannelType       `json:"channel_types,omitzero"`
	// A list of max 10 (discord supported) file type extensions that you want this component to accept.
	//
	// For example: .png, .jpg, .qt, .mp3, .wav
	// It also supports 3 groups (by typing "image", "video" or "audio" as type, please do not hardcode these lists as they are subject to change):
	//
	// IMAGE_EXTENSIONS = ('.png', '.gif', '.jpg', '.jpeg', '.jfif', '.webp', '.avif')
	//
	// VIDEO_EXTENSIONS = ('.mp4', '.mov', '.qt', '.webm')
	//
	// AUDIO_EXTENSIONS = ('.mp3', '.m4a', '.wav', '.ogg', '.opus', '.flac')
	//
	// We recommend using the provided file groups. if you are specifying only extensions,
	// you must include .jpg for image uploads, and both .mp4 and .mov for video uploads,
	// due to mobile schenanigans
	//
	// This feature only checks the extension on the filename - it does not actually inspect
	// the contents of the file. You still need to make sure that the file is valid.
	FileTypes    []string              `json:"file_types,omitzero"`
	Choices      []CommandOptionChoice `json:"choices,omitzero"`
	MinLength    uint16                `json:"min_length,omitzero"`
	MaxLength    uint16                `json:"max_length,omitzero"`
	Required     bool                  `json:"required,omitzero"`
	Type         OptionType            `json:"type"`
	AutoComplete bool                  `json:"autocomplete,omitzero"` // Required to be = true if you want to catch it later in auto complete handler.
}
