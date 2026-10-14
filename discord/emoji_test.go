package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestEmoji(t *testing.T) {
	e1 := Emoji{Name: "thumbsup"}
	if e1.Mention() != "thumbsup" {
		t.Errorf("e1.Mention() = %s, want thumbsup", e1.Mention())
	}

	e2 := Emoji{
		ID:       EmojiID{Snowflake: 5001},
		Name:     "fire",
		Animated: true,
	}
	if e2.Mention() != "<a:fire:5001>" {
		t.Errorf("e2.Mention() = %s, want <a:fire:5001>", e2.Mention())
	}

	data, err := json.Marshal(e2)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Emoji
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != e2.ID || parsed.Name != e2.Name || parsed.Animated != e2.Animated {
		t.Errorf("parsed = %+v, want %+v", parsed, e2)
	}
}

func BenchmarkEmoji_JSON(b *testing.B) {
	e := Emoji{
		ID:       EmojiID{Snowflake: 5001},
		Name:     "fire",
		Animated: true,
	}
	data, _ := json.Marshal(e)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Emoji
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(e)
	}
}
