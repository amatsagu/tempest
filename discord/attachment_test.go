package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestAttachment(t *testing.T) {
	att := Attachment{
		ID:          AttachmentID{Snowflake: 6001},
		Filename:    "screenshot.png",
		ContentType: "image/png",
		Size:        1024,
		URL:         "https://cdn.discordapp.com/attachments/1/2/screenshot.png",
		Width:       800,
		Height:      600,
	}

	data, err := json.Marshal(att)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Attachment
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != att.ID || parsed.Filename != att.Filename || parsed.Size != att.Size {
		t.Errorf("parsed = %+v, want %+v", parsed, att)
	}
}

func BenchmarkAttachment_JSON(b *testing.B) {
	att := Attachment{
		ID:          AttachmentID{Snowflake: 6001},
		Filename:    "screenshot.png",
		ContentType: "image/png",
		Size:        1024,
		URL:         "https://cdn.discordapp.com/attachments/1/2/screenshot.png",
		Width:       800,
		Height:      600,
	}
	data, _ := json.Marshal(&att)

	var parsed Attachment
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&att)
	}
}
