package discord

import "time"

// https://docs.discord.com/developers/resources/message#embed-object-embed-footer-structure
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitzero"`
	ProxyIconURL string `json:"proxy_icon_url,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-image-structure
type EmbedImage struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitzero"`
	Height   uint32 `json:"height,omitzero"`
	Width    uint32 `json:"width,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-thumbnail-structure
type EmbedThumbnail struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitzero"`
	Height   uint32 `json:"height,omitzero"`
	Width    uint32 `json:"width,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-video-structure
type EmbedVideo struct {
	URL      string `json:"url,omitzero"`
	ProxyURL string `json:"proxy_url,omitzero"`
	Height   uint32 `json:"height,omitzero"`
	Width    uint32 `json:"width,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-provider-structure
type EmbedProvider struct {
	Name string `json:"name,omitzero"`
	URL  string `json:"url,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-author-structure
type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitzero"`
	IconURL      string `json:"icon_url,omitzero"`
	ProxyIconURL string `json:"proxy_icon_url,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object-embed-field-structure
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitzero"`
}

// https://docs.discord.com/developers/resources/message#embed-object
type Embed struct {
	Timestamp   *time.Time      `json:"timestamp,omitzero"`
	Footer      *EmbedFooter    `json:"footer,omitzero"`
	Image       *EmbedImage     `json:"image,omitzero"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitzero"`
	Video       *EmbedVideo     `json:"video,omitzero"`
	Provider    *EmbedProvider  `json:"provider,omitzero"`
	Author      *EmbedAuthor    `json:"author,omitzero"`
	Title       string          `json:"title,omitzero"`
	Type        string          `json:"type,omitzero"`
	Description string          `json:"description,omitzero"`
	URL         string          `json:"url,omitzero"`
	Fields      []EmbedField    `json:"fields,omitzero"`
	Color       uint32          `json:"color,omitzero"`
}
