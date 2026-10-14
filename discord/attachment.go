package discord

import "time"

// https://docs.discord.com/developers/resources/message#attachment-object-attachment-flags
type AttachmentFlags = BitSet

const (
	AttachmentFlagIsRemix AttachmentFlags = 1 << 2
)

// https://docs.discord.com/developers/resources/message#attachment-object
type Attachment struct {
	ClipCreatedAt      *time.Time      `json:"clip_created_at,omitzero"`
	Title              string          `json:"title,omitzero"`
	Description        string          `json:"description,omitzero"`
	ContentType        string          `json:"content_type,omitzero"`
	URL                string          `json:"url,omitzero"`
	ProxyURL           string          `json:"proxy_url,omitzero"`
	Waveform           string          `json:"waveform,omitzero"`
	Placeholder        string          `json:"placeholder,omitzero"`
	Filename           string          `json:"filename"`
	ClipParticipants   []User          `json:"clip_participants,omitzero"`
	ID                 AttachmentID    `json:"id"`
	Size               uint64          `json:"size,omitzero"`
	Flags              AttachmentFlags `json:"flags,omitzero"`
	DurationSecs       float64         `json:"duration_secs,omitzero"`
	Height             uint32          `json:"height,omitzero"`
	Width              uint32          `json:"width,omitzero"`
	PlaceholderVersion uint32          `json:"placeholder_version,omitzero"`
	Ephemeral          bool            `json:"ephemeral,omitzero"`
}
