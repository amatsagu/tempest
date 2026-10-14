package discord

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestResponseModal(t *testing.T) {
	input := NewShortTextInput("user_name", "Enter your name")
	label := NewLabel("Name", input)
	resp := NewModalResponse("profile_form", "Edit Profile", label)

	if resp.Type != ResponseModal {
		t.Errorf("got %d, want %d", resp.Type, ResponseModal)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"custom_id":"profile_form"`) || !strings.Contains(str, `"title":"Edit Profile"`) || !strings.Contains(str, `"type":9`) {
		t.Errorf("got %s, missing expected response modal fields", str)
	}
}

func BenchmarkResponseModal_JSON(b *testing.B) {
	input := NewShortTextInput("user_name", "Enter your name")
	label := NewLabel("Name", input)
	resp := NewModalResponse("profile_form", "Edit Profile", label)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&resp)
	}
}
