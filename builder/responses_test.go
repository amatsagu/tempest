package builder

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func TestResponse(t *testing.T) {
	pong := NewPongResponse()
	if pong.Type != discord.ResponsePong {
		t.Errorf("got %d, want %d", pong.Type, discord.ResponsePong)
	}

	data, err := json.Marshal(pong)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"type":1}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}

	def := NewDeferredResponse(true)
	if def.Type != discord.ResponseDeferredChannelMessageWithSource {
		t.Errorf("def.Type = %d, want %d", def.Type, discord.ResponseDeferredChannelMessageWithSource)
	}

	defData, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("Marshal deferred failed: %v", err)
	}

	expectedDef := `{"data":{"flags":64},"type":5}`
	if string(defData) != expectedDef {
		t.Errorf("got %s, want %s", string(defData), expectedDef)
	}
}

func BenchmarkResponse_JSON(b *testing.B) {
	def := NewDeferredResponse(true)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&def)
	}
}

func TestResponseAutocomplete(t *testing.T) {
	resp := NewAutocompleteResponse(
		discord.CommandOptionChoice{Name: "Option A", Value: "a"},
		discord.CommandOptionChoice{Name: "Option B", Value: "b"},
	)

	if resp.Type != discord.ResponseApplicationCommandAutocomplete {
		t.Errorf("got %d, want %d", resp.Type, discord.ResponseApplicationCommandAutocomplete)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"type":8`) || !strings.Contains(str, `"name":"Option A"`) {
		t.Errorf("got %s, missing expected autocomplete fields", str)
	}
}

func BenchmarkResponseAutocomplete_JSON(b *testing.B) {
	resp := NewAutocompleteResponse(
		discord.CommandOptionChoice{Name: "Option A", Value: "a"},
		discord.CommandOptionChoice{Name: "Option B", Value: "b"},
	)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&resp)
	}
}

func TestResponseMessage(t *testing.T) {
	btn := NewButton("click", "Click Me", discord.ButtonPrimary)
	row := NewActionRow(btn)
	resp := NewMessageResponse("Welcome!", discord.MessageFlagEphemeral, row)

	if resp.Type != discord.ResponseChannelMessageWithSource {
		t.Errorf("got %d, want %d", resp.Type, discord.ResponseChannelMessageWithSource)
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
	if updateResp.Type != discord.ResponseUpdateMessage {
		t.Errorf("updateResp.Type = %d, want %d", updateResp.Type, discord.ResponseUpdateMessage)
	}
}

func BenchmarkResponseMessage_JSON(b *testing.B) {
	btn := NewButton("click", "Click Me", discord.ButtonPrimary)
	row := NewActionRow(btn)
	resp := NewMessageResponse("Welcome!", discord.MessageFlagEphemeral, row)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&resp)
	}
}

func TestResponseModal(t *testing.T) {
	input := NewShortTextInput("user_name", "Enter your name")
	label := NewLabel("Name", input)
	resp := NewModalResponse("profile_form", "Edit Profile", label)

	if resp.Type != discord.ResponseModal {
		t.Errorf("got %d, want %d", resp.Type, discord.ResponseModal)
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
