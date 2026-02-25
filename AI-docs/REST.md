# REST API & Objects

## Direct Methods
```go
func (client *BaseClient) SendMessage(channelID Snowflake, message Message, files []File) (Message, error)
func (client *BaseClient) FetchUser(id Snowflake) (User, error)
func (client *BaseClient) FetchMember(guildID Snowflake, memberID Snowflake) (Member, error)
func (client *BaseClient) SyncCommandsWithDiscord(guildIDs []Snowflake, whitelist []string, reverseMode bool) error
```

## Core Objects
### Message
```go
type Message struct {
	ID                  Snowflake
	ChannelID           Snowflake
	Author              *User
	Content             string
	Timestamp           *time.Time
	EditedTimestamp     *time.Time
	Embeds              []Embed
	Components          []MessageComponent
	Attachments         []Attachment
	Flags               MessageFlags
}
```

### Embed
```go
type Embed struct {
	Title       string
	URL         string
	Author      *EmbedAuthor
	Color       uint32
	Thumbnail   *EmbedThumbnail
	Description string
	Fields      []EmbedField
	Footer      *EmbedFooter
	Image       *EmbedImage
	Timestamp   *time.Time
}
```

### User & Member
```go
type User struct {
	ID           Snowflake
	Username     string
	GlobalName   string
	AvatarHash   string
	Bot          bool
	BannerHash   string
	AccentColor  uint32
}

type Member struct {
	User            *User
	Nickname        string
	RoleIDs         []Snowflake
	JoinedAt        *time.Time
	PermissionFlags PermissionFlags
}
```

### Role
```go
type Role struct {
	ID              Snowflake
	Name            string
	Color           uint32
	Hoist           bool
	PermissionFlags PermissionFlags
	Position        uint8
}
```

## Utils
### Snowflake
```go
type Snowflake uint64
func (s Snowflake) String() string
func (s Snowflake) CreationTimestamp() time.Time
func StringToSnowflake(s string) (Snowflake, error)
```

### BitSet / Permissions
```go
type BitSet uint64
func (f BitSet) Has(bits ...BitSet) bool
func (f BitSet) Add(bits ...BitSet) BitSet
```
