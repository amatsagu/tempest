package discord

// https://docs.discord.com/developers/resources/message#attachment-object-attachment-flags
type AttachmentFlags = BitSet

const (
	AttachmentFlagIsRemix AttachmentFlags = 1 << 2
)

// https://docs.discord.com/developers/resources/message#attachment-object
type Attachment struct {
	Filename     string          `json:"filename"`
	Title        string          `json:"title,omitzero"`
	Description  string          `json:"description,omitzero"`
	ContentType  string          `json:"content_type,omitzero"`
	URL          string          `json:"url,omitzero"`
	ProxyURL     string          `json:"proxy_url,omitzero"`
	Waveform     string          `json:"waveform,omitzero"`
	ID           AttachmentID    `json:"id"`
	Size         uint64          `json:"size,omitzero"`
	Height       uint32          `json:"height,omitzero"`
	Width        uint32          `json:"width,omitzero"`
	Flags        AttachmentFlags `json:"flags,omitzero"`
	DurationSecs float64         `json:"duration_secs,omitzero"`
	Ephemeral    bool            `json:"ephemeral,omitzero"`
}
