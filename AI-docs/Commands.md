# Slash Commands

## Structures
### Command
```go
type Command struct {
	Type                     CommandType
	ApplicationID            Snowflake
	GuildID                  Snowflake
	Name                     string
	NameLocalizations        map[Language]string
	Description              string
	DescriptionLocalizations map[Language]string
	Options                  []CommandOption
	RequiredPermissions      PermissionFlags
	IntegrationTypes         []ApplicationIntegrationType
	Contexts                 []InteractionContextType
	NSFW                     bool
	AutoCompleteHandler      func(itx CommandInteraction) []CommandOptionChoice
	SlashCommandHandler      func(itx *CommandInteraction)
}
```

### CommandOption
```go
type CommandOption struct {
	Type                     OptionType
	Name                     string
	NameLocalizations        map[Language]string
	Description              string
	DescriptionLocalizations map[Language]string
	Required                 bool
	Choices                  []CommandOptionChoice
	Options                  []CommandOption
	ChannelTypes             []ChannelType
	MinValue                 float64
	MaxValue                 float64
	MinLength                uint16
	MaxLength                uint16
	AutoComplete             bool
}
```

### CommandOptionChoice
```go
type CommandOptionChoice struct {
	Name              string
	NameLocalizations map[Language]string
	Value             any // string, float64, or bool
}
```

## Enums & Constants
### CommandType
```go
const (
	CHAT_INPUT_COMMAND_TYPE CommandType = iota + 1
	USER_COMMAND_TYPE
	MESSAGE_COMMAND_TYPE
	PRIMARY_ENTRY_POINT_COMMAND_TYPE
)
```

### OptionType
```go
const (
	SUB_COMMAND_OPTION_TYPE OptionType = iota + 1
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
```

## Methods
```go
func (client *BaseClient) RegisterCommand(cmd Command) error
func (client *BaseClient) RegisterSubCommand(subCommand Command, parentCommandName string) error
func (client *BaseClient) SyncCommandsWithDiscord(guildIDs []Snowflake, whitelist []string, reverseMode bool) error
```

## Sub-commands
Internal tracking via `parent@name`. Register parent first.
```go
client.RegisterCommand(tempest.Command{Name: "user", ...})
client.RegisterSubCommand(tempest.Command{Name: "info", ...}, "user")
```

## Autocomplete
Return <= 25 choices. Use `itx.GetFocusedValue()` to identify targeted option.
