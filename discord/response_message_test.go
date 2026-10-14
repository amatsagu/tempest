package discord

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestResponseMessage(t *testing.T) {
	btn := NewButton("click", "Click Me", ButtonPrimary)
	row := NewActionRow(btn)
	resp := NewMessageResponse("Welcome!", MessageFlagEphemeral, row)

	if resp.Type != ResponseChannelMessageWithSource {
		t.Errorf("got %d, want %d", resp.Type, ResponseChannelMessageWithSource)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"content":"Welcome!"`) || !strings.Contains(str, `"flags":64`) || !strings.Contains(str, `"type":4`) {
		t.Errorf("got %s, missing expected response message fields", str)
	}

	updateResp := NewUpdateMessageResponse("Updated content", 0)
	if updateResp.Type != ResponseUpdateMessage {
		t.Errorf("updateResp.Type = %d, want %d", updateResp.Type, ResponseUpdateMessage)
	}
}

func BenchmarkResponseMessage_JSON(b *testing.B) {
	btn := NewButton("click", "Click Me", ButtonPrimary)
	row := NewActionRow(btn)
	resp := NewMessageResponse("Welcome!", MessageFlagEphemeral, row)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&resp)
	}
}
