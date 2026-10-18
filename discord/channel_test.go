package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestChannel(t *testing.T) {
	ch := Channel{
		ID:       ChannelID{Snowflake: 4001},
		Type:     ChannelGuildText,
		Name:     "general",
		Position: 1,
	}

	if ch.ID.Mention() != "<#4001>" {
		t.Errorf("ch.ID.Mention() = %s, want <#4001>", ch.ID.Mention())
	}

	data, err := json.Marshal(ch)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Channel
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != ch.ID || parsed.Name != ch.Name || parsed.Type != ch.Type {
		t.Errorf("parsed = %+v, want %+v", parsed, ch)
	}
}

func BenchmarkChannel_JSON(b *testing.B) {
	ch := Channel{
		ID:       ChannelID{Snowflake: 4001},
		Type:     ChannelGuildText,
		Name:     "general",
		Position: 1,
	}
	data, _ := json.Marshal(&ch)

	// Marshal through a pointer and reuse the decode target to avoid copying.
	var parsed Channel
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&ch)
	}
}
