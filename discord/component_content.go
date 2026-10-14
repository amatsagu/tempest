package discord

// https://docs.discord.com/developers/components/reference#unfurled-media-item-unfurled-media-item-flags
type UnfurledMediaItemFlags = BitSet

const (
	UnfurledMediaItemFlagSpoiler UnfurledMediaItemFlags = 1 << 0
)

// https://docs.discord.com/developers/components/reference#unfurled-media-item-unfurled-media-item-structure
type UnfurledMediaItem struct {
	URL                string                 `json:"url"`
	ProxyURL           string                 `json:"proxy_url,omitzero"`
	Placeholder        string                 `json:"placeholder,omitzero"`
	Height             uint32                 `json:"height,omitzero"`
	Width              uint32                 `json:"width,omitzero"`
	PlaceholderVersion uint32                 `json:"placeholder_version,omitzero"`
	Flags              UnfurledMediaItemFlags `json:"flags,omitzero"`
}

// https://docs.discord.com/developers/components/reference#text-display-text-display-structure
type TextDisplay struct {
	Content string        `json:"content"`
	ID      uint32        `json:"id,omitzero"`
	Type    ComponentType `json:"type"`
}

func (t TextDisplay) ComponentType() ComponentType { return ComponentTextDisplay }

// https://docs.discord.com/developers/components/reference#thumbnail-thumbnail-structure
type Thumbnail struct {
	Description string            `json:"description,omitzero"`
	Media       UnfurledMediaItem `json:"media"`
	ID          uint32            `json:"id,omitzero"`
	Type        ComponentType     `json:"type"`
	Spoiler     bool              `json:"spoiler,omitzero"`
}

func (t Thumbnail) ComponentType() ComponentType { return ComponentThumbnail }

// https://docs.discord.com/developers/components/reference#media-gallery-media-gallery-item-structure
type MediaGalleryItem struct {
	Description string            `json:"description,omitzero"`
	Media       UnfurledMediaItem `json:"media"`
	Spoiler     bool              `json:"spoiler,omitzero"`
}

// https://docs.discord.com/developers/components/reference#media-gallery-media-gallery-structure
type MediaGallery struct {
	Items []MediaGalleryItem `json:"items"`
	ID    uint32             `json:"id,omitzero"`
	Type  ComponentType      `json:"type"`
}

func (m MediaGallery) ComponentType() ComponentType { return ComponentMediaGallery }

// https://docs.discord.com/developers/components/reference#file-file-structure
type FileComponent struct {
	File    UnfurledMediaItem `json:"file"`
	ID      uint32            `json:"id,omitzero"`
	Type    ComponentType     `json:"type"`
	Spoiler bool              `json:"spoiler,omitzero"`
}

func (f FileComponent) ComponentType() ComponentType { return ComponentFile }
