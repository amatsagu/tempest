package tempest

type PartialEmoji struct {
	Id       Snowflake `json:"id,omitempty"`
	Name     string    `json:"name"`
	Animated bool      `json:"animated,omitempty"`
}

type Emoji struct {
	ID            Snowflake   `json:"id,omitempty"`
	Name          string      `json:"name"`
	Roles         []Snowflake `json:"roles,omitempty"`
	User          *User       `json:"user,omitempty"`
	RequireColons bool        `json:"require_colons,omitempty"`
	Managed       bool        `json:"managed,omitempty"`
	Animated      bool        `json:"animated,omitempty"`
	Available     bool        `json:"available,omitempty"`
}

type Embed struct {
	Title       string          `json:"title,omitempty"`
	Url         string          `json:"url,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Color       uint32          `json:"color,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Description string          `json:"description,omitempty"`
	Fields      []*EmbedField   `json:"fields,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image.url,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
}

type EmbedAuthor struct {
	IconUrl string `json:"icon_url,omitempty"`
	Name    string `json:"name,omitempty"`
	Url     string `json:"url,omitempty"`
}

type EmbedThumbnail struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	IconUrl string `json:"icon_url,omitempty"`
	Text    string `json:"text,omitempty"`
}

type EmbedImage struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type EmbedVideo struct {
	URL    string `json:"url,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type EmbedProvider struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

type Message struct {
	Id                Snowflake         `json:"id"`
	ChannelId         Snowflake         `json:"channel_id"`
	GuildId           Snowflake         `json:"guild_id,omitempty"`
	TTS               bool              `json:"tts"`
	Pinned            bool              `json:"pinned"`
	MentionEveryone   bool              `json:"mention_everyone"`
	Mentions          []*User           `json:"mentions"`
	MentionRoleIDs    []Snowflake       `json:"mention_roles"`
	Author            *User             `json:"author"`
	Content           string            `json:"content"`
	Timestamp         string            `json:"timestamp"`
	EditedTimestamp   string            `json:"edited_timestamp,omitempty"`
	Embeds            []*Embed          `json:"embeds"`
	Components        []*Component      `json:"components,omitempty"`
	Reference         *MessageReference `json:"message_reference,omitempty"`  // Reference data sent with crossposted messages and inline replies.
	ReferencedMessage *Message          `json:"referenced_message,omitempty"` // ReferencedMessage is the message that was replied to.
}

type MessageReference struct {
	MessageID Snowflake `json:"message_id,omitempty"`
	ChannelID Snowflake `json:"channel_id,omitempty"`
	GuildID   Snowflake `json:"guild_id,omitempty"`
}
