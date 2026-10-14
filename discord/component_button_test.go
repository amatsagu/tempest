package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestButton(t *testing.T) {
	btn := NewButton("btn_click", "Click Me", ButtonPrimary)
	if btn.Type != ComponentButton {
		t.Errorf("got %d, want %d", btn.Type, ComponentButton)
	}

	data, err := json.Marshal(btn)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Button
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.CustomID != btn.CustomID || parsed.Label != btn.Label || parsed.Style != btn.Style {
		t.Errorf("parsed = %+v, want %+v", parsed, btn)
	}
}

func BenchmarkButton_JSON(b *testing.B) {
	btn := NewButton("btn_click", "Click Me", ButtonPrimary)
	data, _ := json.Marshal(btn)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Button
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(btn)
	}
}
