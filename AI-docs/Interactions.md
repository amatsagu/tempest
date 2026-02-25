# Interaction Handling

## Types
- `CommandInteraction`: Type 2.
- `ComponentInteraction`: Type 3.
- `ModalInteraction`: Type 5.

## Responders
### Standard Replies
```go
func (itx *CommandInteraction) SendReply(reply ResponseMessageData, ephemeral bool, files []File) error
func (itx *CommandInteraction) SendLinearReply(content string, ephemeral bool) error
func (itx *CommandInteraction) Defer(ephemeral bool) error
```

### Updates (Component/Modal)
```go
func (itx *ComponentInteraction) Acknowledge() error
func (itx *ComponentInteraction) AcknowledgeWithMessage(reply ResponseMessageData, ephemeral bool) error
```

### Follow-up (Post-Defer)
```go
func (itx *CommandInteraction) SendFollowUp(content ResponseMessageData, ephemeral bool) (Message, error)
func (itx *CommandInteraction) EditReply(content ResponseMessageData, ephemeral bool) error
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
- `itx.BaseUser()`: Helper to get `User` regardless of context.

### Resolved Data
```go
func (itx *CommandInteraction) ResolveUser(id Snowflake) User
func (itx *CommandInteraction) ResolveMember(id Snowflake) Member
func (itx *ModalInteraction) GetInputValue(customID string) string
```
