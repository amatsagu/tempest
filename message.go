package tempest

import "time"

type PartialChannel struct {
	ID              Snowflake   `json:"id"`
	Name            string      `json:"name"`
	PermissionFlags uint64      `json:"permissions,string"`
	Type            ChannelType `json:"type"`
}

type PartialEmoji struct {
	ID       Snowflake `json:"id,omitempty"`
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
	URL         string          `json:"url,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Color       uint32          `json:"color,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Description string          `json:"description,omitempty"`
	Fields      []*EmbedField   `json:"fields,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Video       *EmbedVideo     `json:"video,omitempty"`
	Provider    *EmbedProvider  `json:"provider,omitempty"`
	Timestamp   *time.Time      `json:"timestamp,omitempty"`
}

type EmbedAuthor struct {
	IconURL string `json:"icon_url,omitempty"`
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
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
	IconURL string `json:"icon_url,omitempty"`
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
	ID                Snowflake         `json:"id"`
	ChannelID         Snowflake         `json:"channel_id"`
	GuildID           Snowflake         `json:"guild_id,omitempty"`
	TTS               bool              `json:"tts"`
	Pinned            bool              `json:"pinned"`
	MentionEveryone   bool              `json:"mention_everyone"`
	Mentions          []*User           `json:"mentions"`
	MentionRoleIDs    []Snowflake       `json:"mention_roles"`
	Author            *User             `json:"author"`
	Content           string            `json:"content"`
	Timestamp         *time.Time        `json:"timestamp,omitempty"`
	EditedTimestamp   *time.Time        `json:"edited_timestamp,omitempty"`
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

// Ephemeral attachments will automatically be removed after a set period of time. Ephemeral attachments on messages are guaranteed to be available as long as the message itself exists.
type Attachment struct {
	ID          Snowflake `json:"id"`
	FileName    string    `json:"filename,omitempty"`
	Description string    `json:"description,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	// Size of file in bytes.
	Size     uint32 `json:"size"`
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url"`
	// Height of file in pixels (if image).
	Height uint32 `json:"height,omitempty"`
	// Width of file in pixels (if image).
	Width     uint32 `json:"width,omitempty"`
	Ephemeral bool   `json:"ephemeral"`
}
