# Slash Commands

## Structures
### `Command`
```go
type Command struct {
	Name                     string
	Description              string
	Type                     CommandType
	Options                  []CommandOption
	DefaultMemberPermissions uint64
	Contexts                 []InteractionContextType
	IntegrationTypes         []IntegrationType
	NSFW                     bool
	SlashCommandHandler      func(itx *CommandInteraction)
	AutoCompleteHandler      func(itx CommandInteraction) []CommandOptionChoice
}
```

### `CommandOption`
```go
type CommandOption struct {
	Type         CommandOptionType
	Name         string
	Description  string
	Required     bool
	Choices      []CommandOptionChoice
	Options      []CommandOption
	ChannelTypes []ChannelType
	MinValue     float64
	MaxValue     float64
	MinLength    uint32
	MaxLength    uint32
	Autocomplete bool
}
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
Return <= 25 choices.
```go
AutoCompleteHandler: func(itx tempest.CommandInteraction) []tempest.CommandOptionChoice {
    name, value := itx.GetFocusedValue()
    // return slice of choices
}
```
