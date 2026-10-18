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

func TestNonce_DecodesStringAndNumber(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Nonce
	}{
		{"json string", `"abc123"`, "abc123"},
		{"json number", `12345`, "12345"},
		{"null", `null`, ""},
		{"empty string", `""`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg struct {
				Nonce Nonce `json:"nonce"`
			}

			if err := json.Unmarshal([]byte(`{"nonce":`+tt.input+`}`), &msg); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if msg.Nonce != tt.want {
				t.Errorf("Nonce = %q, want %q", msg.Nonce, tt.want)
			}
		})
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
	data, _ := json.Marshal(&msg)

	// Marshal through a pointer and reuse the decode target to avoid copying.
	var parsed Message
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&msg)
	}
}
