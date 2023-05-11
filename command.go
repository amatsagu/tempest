package tempest

import "strconv"

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

// https://discord.com/developers/docs/resources/channel#channel-object-channel-types
type ChannelType uint8

const (
	GUILD_TEXT_CHANNEL_TYPE ChannelType = iota
	DM_CHANNEL_TYPE
	GUILD_VOICE_CHANNEL_TYPE
	GROUP_DM_CHANNEL_TYPE
	GUILD_CATEGORY_CHANNEL_TYPE
	GUILD_ANNOUNCEMENT_CHANNEL_TYPE // Formerly news channel.
	_
	_
	_
	_
	GUILD_ANNOUNCEMENT_THREAD_CHANNEL_TYPE
	GUILD_PUBLIC_THREAD_CHANNEL_TYPE
	GUILD_PRIVATE_THREAD_CHANNEL_TYPE
	GUILD_STAGE_VOICE_CHANNEL_TYPE
	GUILD_DIRECTORY_CHANNEL_TYPE
	GUILD_FORUM_CHANNEL_TYPE
)

func (ct ChannelType) MarshalJSON() (p []byte, err error) {
	buf := strconv.FormatUint(uint64(ct), 10)
	return []byte(buf), nil
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure
type Command struct {
	ID                       Snowflake         `json:"id"`
	Type                     CommandType       `json:"type,omitempty"`
	ApplicationID            Snowflake         `json:"application_id"`
	GuildID                  Snowflake         `json:"guild_id,omitempty"` // "Guild ID of the command, if not global"
	Name                     string            `json:"name"`
	NameLocalizations        map[string]string `json:"name_localizations,omitempty"` // https://discord.com/developers/docs/reference#locales
	Description              string            `json:"description"`
	DescriptionLocalizations map[string]string `json:"description_localizations,omitempty"`
	Options                  []CommandOption   `json:"options,omitempty"`
	DefaultMemberPermissions uint64            `json:"default_member_permissions,string,omitempty"` // Set of permissions represented as a bit set. Set it to 0 to make command unavailable for regular members.
	AvailableInDM            bool              `json:"dm_permission,omitempty"`                     // Whether command should be visible (usable) from private, dm channels. Works only for global commands!
	NSFW                     bool              `json:"nsfw,omitempty"`                              // https://discord.com/developers/docs/interactions/application-commands#agerestricted-commands
	Version                  Snowflake         `json:"version,omitempty"`                           // Autoincrementing version identifier updated during substantial record changes

	AutoCompleteHandler func(itx AutoCompleteInteraction) []CommandChoice `json:"-"` // Custom handler for auto complete interactions. It's a Tempest specific field.
	SlashCommandHandler func(itx CommandInteraction)                      `json:"-"` // Custom handler for slash command interactions. It's a Tempest specific field. Warning! Library will panic if command can be triggered but doesn't have this handler.
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-structure
type CommandOption struct {
	Type                     OptionType        `json:"type"`
	Name                     string            `json:"name"`
	NameLocalizations        map[string]string `json:"name_localizations,omitempty"` // https://discord.com/developers/docs/reference#locales
	Description              string            `json:"description"`
	DescriptionLocalizations map[string]string `json:"description_localizations,omitempty"`
	Required                 bool              `json:"required,omitempty"`
	MinValue                 float64           `json:"min_value,omitempty"`
	MaxValue                 float64           `json:"max_value,omitempty"`
	MinLength                uint              `json:"min_length,omitempty"`
	MaxLength                uint              `json:"max_length,omitempty"`
	Options                  []CommandOption   `json:"options,omitempty"`
	ChannelTypes             []ChannelType     `json:"channel_types,omitempty"`
	Choices                  []CommandChoice   `json:"choices,omitempty"`
	AutoComplete             bool              `json:"autocomplete,omitempty"` // Required to be = true if you want to catch it later in auto complete handler.
}

// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-option-choice-structure
type CommandChoice struct {
	Name              string            `json:"name"`
	NameLocalizations map[string]string `json:"name_localizations,omitempty"` // https://discord.com/developers/docs/reference#locales
	Value             any               `json:"value"`                        // string or float64 (integer or number type), needs to be handled
}
