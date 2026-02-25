# Interaction Handling

## Structures
### Interaction
```go
type Interaction struct {
	ID              Snowflake
	ApplicationID   Snowflake
	Type            InteractionType
	Data            json.RawMessage
	GuildID         Snowflake
	ChannelID       Snowflake
	Member          *Member
	User            *User
	Token           string
	PermissionFlags PermissionFlags
	Locale          Language
	GuildLocale     string
	Entitlements    []Entitlement
}
```

### CommandInteraction
```go
type CommandInteraction struct {
	*Interaction
	Data CommandInteractionData
}

type CommandInteractionData struct {
	ID       Snowflake
	Name     string
	Type     CommandType
	Resolved *InteractionDataResolved
	Options  []CommandInteractionOption
	GuildID  Snowflake
	TargetID Snowflake
}
```

### Resolved Data
```go
type InteractionDataResolved struct {
	Users       map[Snowflake]User
	Members     map[Snowflake]Member
	Roles       map[Snowflake]Role
	Channels    map[Snowflake]PartialChannel
	Messages    map[Snowflake]Message
	Attachments map[Snowflake]Attachment
}
```

## Responders
### Response Structs
```go
type ResponseMessageData struct {
	TTS             bool
	Content         string
	Embeds          []Embed
	AllowedMentions *AllowedMentions
	Flags           MessageFlags
	Components      []MessageComponent
	Attachments     []Attachment
}

type ResponseModalData struct {
	CustomID   string
	Title      string
	Components []ModalComponent
}
```

### Methods
```go
func (itx *CommandInteraction) SendReply(reply ResponseMessageData, ephemeral bool, files []File) error
func (itx *CommandInteraction) SendLinearReply(content string, ephemeral bool) error
func (itx *CommandInteraction) Defer(ephemeral bool) error
func (itx *CommandInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error)
func (itx *CommandInteraction) EditReply(content ResponseMessageData, ephemeral bool) error

func (itx *ComponentInteraction) Acknowledge() error
func (itx *ComponentInteraction) AcknowledgeWithMessage(reply ResponseMessageData, ephemeral bool) error
```

## Extraction Helpers
### Options
`GetOptionValue` returns `any` (usually `float64` for numbers) and `bool` (presence).
```go
val, _ := itx.GetOptionValue("name")
str := val.(string)
```

### Identity
- `itx.Member`: Non-nil in Guilds.
- `itx.User`: Non-nil in DMs.
- `itx.BaseUser()`: Returns `User` regardless of context.

### Resolution
```go
func (itx *CommandInteraction) ResolveUser(id Snowflake) User
func (itx *CommandInteraction) ResolveMember(id Snowflake) Member
func (itx *ModalInteraction) GetInputValue(customID string) string
```
