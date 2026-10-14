package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestMessageStickerItems(t *testing.T) {
	raw := []byte(`{"id":"1","channel_id":"2","timestamp":"2024-01-01T00:00:00+00:00","author":{"id":"3","username":"u"},"sticker_items":[{"id":"4","name":"wave","format_type":3}]}`)

	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}

	if len(msg.StickerItems) != 1 {
		t.Fatalf("sticker items = %d, want 1", len(msg.StickerItems))
	}

	si := msg.StickerItems[0]
	if si.ID != 4 || si.Name != "wave" || si.FormatType != StickerFormatLottie {
		t.Errorf("sticker item = %+v, want id 4 name wave format Lottie", si)
	}
}

func TestStickerGuild(t *testing.T) {
	raw := []byte(`{"id":"5","name":"pack sticker","tags":"cute","type":2,"format_type":1,"guild_id":"9","available":true}`)

	var s Sticker
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal sticker: %v", err)
	}

	if s.Type != StickerGuild || s.FormatType != StickerFormatPNG || s.GuildID.Snowflake != 9 || !s.Available {
		t.Errorf("sticker = %+v", s)
	}
}
