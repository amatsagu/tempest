package discord

import (
	"encoding/json/v2"
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	msg := Message{
		ID:        MessageID{Snowflake: 8888},
		ChannelID: ChannelID{Snowflake: 9999},
		Author: User{
			ID:       UserID{Snowflake: 1001},
			Username: "bot",
		},
		Content:   "Hello world",
		Timestamp: now,
		Type:      MessageTypeDefault,
		Flags:     MessageFlagEphemeral,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Message
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != msg.ID || parsed.Content != msg.Content || parsed.Flags != msg.Flags {
		t.Errorf("parsed = %+v, want %+v", parsed, msg)
	}
}

func BenchmarkMessage_JSON(b *testing.B) {
	now := time.Now().UTC().Truncate(time.Second)
	msg := Message{
		ID:        MessageID{Snowflake: 8888},
		ChannelID: ChannelID{Snowflake: 9999},
		Author: User{
			ID:       UserID{Snowflake: 1001},
			Username: "bot",
		},
		Content:   "Hello world",
		Timestamp: now,
		Type:      MessageTypeDefault,
		Flags:     MessageFlagEphemeral,
	}
	data, _ := json.Marshal(msg)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Message
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(msg)
	}
}
