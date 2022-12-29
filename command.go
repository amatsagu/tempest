package tempest

import "strconv"

type CommandType uint8

const (
	COMMAND_CHAT_INPUT CommandType = iota + 1
	COMMAND_USER
	COMMAND_MESSAGE
)

type ChannelType uint8

const (
	CHANNEL_GUILD_TEXT ChannelType = iota
	CHANNEL_DM
	CHANNEL_GUILD_VOICE
	CHANNEL_GROUP_DM
	CHANNEL_GUILD_CATEGORY
	CHANNEL_GUILD_NEWS
	_
	_
	_
	_
	CHANNEL_GUILD_NEWS_THREAD
	CHANNEL_GUILD_PUBLIC_THREAD
	CHANNEL_GUILD_PRIVATE_THREAD
	CHANNEL_GUILD_STAGE_VOICE
	CHANNEL_GUILD_DIRECTORY
	CHANNEL_GUILD_FORUM // (still in development) Channel that can only contain threads.
)

func (ct ChannelType) MarshalJSON() (p []byte, err error) {
	buf := strconv.FormatUint(uint64(ct), 10)
	return []byte(buf), nil
}

type Command struct {
	ID                 Snowflake   `json:"id,omitempty"`
	ApplicationID      Snowflake   `json:"application_id,omitempty"`
	GuildID            Snowflake   `json:"guild_id,omitempty"`
	Name               string      `json:"name,omitempty"`
	Description        string      `json:"description,omitempty"`
	Type               CommandType `json:"type,omitempty"`
	Options            []Option    `json:"options,omitempty"`
	DefaultPermissions uint64      `json:"default_member_permissions,string,omitempty"` // Set of permissions represented as a bit set. Set it to 0 to make command unavailable for regular members.
	AvailableInDM      bool        `json:"dm_permission,omitempty"`                     // Whether command should be visible (usable) from private, dm channels. Works only for global commands!
	Version            Snowflake   `json:"version,omitempty"`                           // Autoincrementing version identifier updated during substantial record changes

	AutoCompleteHandler func(itx AutoCompleteInteraction) []Choice `json:"-"` // Custom handler for auto complete interactions. It's a Tempest specific field.
	SlashCommandHandler func(itx CommandInteraction)               `json:"-"` // Custom handler for slash command interactions. It's a Tempest specific field. Warning! Library will panic if command can be triggered but doesn't have this handler.

	// There's missing localization support and "default_member_permissions" field which contains flag required for users/members to use this command.
	// If you really need this then feel free to make a pull request.
	// https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure
}

// Option is an option for an application Command
type Option struct {
	Name         string        `json:"name"`
	Type         OptionType    `json:"type"`
	Description  string        `json:"description,omitempty"`
	Required     bool          `json:"required,omitempty"`
	MinValue     int           `json:"min_value,omitempty"`  // Declares min value for integer/number option.
	MaxValue     int           `json:"max_value,omitempty"`  // Declares max value for integer/number option.
	MinLength    uint          `json:"min_length,omitempty"` // Declares min length for string option.
	MaxLength    uint          `json:"max_length,omitempty"` // Declares max length for string option.
	ChannelTypes []ChannelType `json:"channel_types,omitempty"`
	Options      []Option      `json:"options,omitempty"`
	Choices      []Choice      `json:"choices,omitempty"`
	AutoComplete bool          `json:"autocomplete,omitempty"` // Required to be = true if you want to catch it later in auto complete handler.
	Focused      bool          `json:"focused,omitempty"`
}

// Choice is an application Command choice
type Choice struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}
