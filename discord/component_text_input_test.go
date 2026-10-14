package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestTextInput(t *testing.T) {
	input := NewShortTextInput("feedback", "Type feedback here")
	if input.Type != ComponentTextInput {
		t.Errorf("got %d, want %d", input.Type, ComponentTextInput)
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed TextInput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.CustomID != input.CustomID || parsed.Style != input.Style {
		t.Errorf("parsed = %+v, want %+v", parsed, input)
	}
}

func BenchmarkTextInput_JSON(b *testing.B) {
	input := NewShortTextInput("feedback", "Type feedback here")
	data, _ := json.Marshal(&input)

	var parsed TextInput
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&input)
	}
}
