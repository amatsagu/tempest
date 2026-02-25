# REST API

## Messaging
```go
func (client *BaseClient) SendMessage(channelID Snowflake, message Message, files []File) (Message, error)
func (client *BaseClient) EditMessage(channelID Snowflake, messageID Snowflake, content Message) error
func (client *BaseClient) DeleteMessage(channelID Snowflake, messageID Snowflake) error
```

## Fetching
```go
func (client *BaseClient) FetchUser(id Snowflake) (User, error)
func (client *BaseClient) FetchMember(guildID Snowflake, memberID Snowflake) (Member, error)
```

## Entitlements
```go
func (client *BaseClient) FetchEntitlementsPage(queryFilter string) ([]Entitlement, error)
func (client *BaseClient) FetchEntitlement(entitlementID Snowflake) (Entitlement, error)
func (client *BaseClient) ConsumeEntitlement(entitlementID Snowflake) error
```

## Utility Types
### `Snowflake`
```go
type Snowflake uint64
func (s Snowflake) String() string
func (s Snowflake) CreationTimestamp() time.Time
func StringToSnowflake(s string) (Snowflake, error)
```

### `Message`
```go
type Message struct {
	ID              Snowflake
	ChannelID       Snowflake
	Author          *User
	Content         string
	Embeds          []Embed `json:"embeds,omitzero"`
	Components      []MessageComponent `json:"components,omitzero"`
	Attachments     []Attachment `json:"attachments,omitzero"`
}
```

## Raw Requests
```go
func (rest *Rest) Request(method, route string, jsonPayload any) ([]byte, error)
```
