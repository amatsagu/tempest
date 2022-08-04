package tempest

type CommandType uint8

const (
	COMMAND_CHAT_INPUT CommandType = iota + 1
	COMMAND_USER
	COMMAND_MESSAGE
)

type Command struct {
	Id                 Snowflake   `json:"id,omitempty"`
	ApplicationID      Snowflake   `json:"application_id,omitempty"`
	GuildID            Snowflake   `json:"guild_id,omitempty"`
	Name               string      `json:"name,omitempty"`
	Description        string      `json:"description,omitempty"`
	Type               CommandType `json:"type,omitempty"`
	Options            []Option    `json:"options,omitempty"`
	DefaultPermissions uint64      `json:"default_member_permissions,string,omitempty"` // Set of permissions represented as a bit set. Set it to 0 to make command unavailable for regular members.
	AvailableInDM      bool        `json:"dm_permission,omitempty"`                     // Whether command should be visible (usable) from private, dm channels. Works only for global commands!
	Version            Snowflake   `json:"version,omitempty"`                           // Autoincrementing version identifier updated during substantial record changes

	AutoCompleteHandler func()                                      `json:"-"` // Custom handler for auto complete interactions. It's a Tempest specific field.
	SlashCommandHandler func(commandInteraction CommandInteraction) `json:"-"` // Custom handler for slash command interactions. It's a Tempest specific field.
}

// Option is an option for an application Command
type Option struct {
	Name        string        `json:"name"`
	Type        OptionType    `json:"type"`
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required,omitempty"`
	MinValue    int           `json:"min_value,omitempty"`  // Declares min value for integer/number option.
	MaxValue    int           `json:"max_value,omitempty"`  // Declares max value for integer/number option.
	MinLength   uint          `json:"min_length,omitempty"` // Declares min length for string option.
	MaxLength   uint          `json:"max_length,omitempty"` // Declares max length for string option.
	Options     []Option      `json:"options,omitempty"`
	Choices     []Choice[any] `json:"choices,omitempty"`
	Focused     bool          `json:"focused,omitempty"`
}

// Choice is an application Command choice
type Choice[T any] struct {
	Name  string `json:"name"`
	Value T      `json:"value"`
}
