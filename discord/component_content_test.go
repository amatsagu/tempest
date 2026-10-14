package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestComponentContent(t *testing.T) {
	txt := NewTextDisplay("**Bold Notice**")
	if txt.Type != ComponentTextDisplay {
		t.Errorf("got %d, want %d", txt.Type, ComponentTextDisplay)
	}

	data, err := json.Marshal(txt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed TextDisplay
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Content != txt.Content {
		t.Errorf("parsed.Content = %s, want %s", parsed.Content, txt.Content)
	}

	thumb := NewThumbnail("https://example.com/thumb.png", "thumbnail")
	if thumb.Type != ComponentThumbnail {
		t.Errorf("thumb.Type = %d, want %d", thumb.Type, ComponentThumbnail)
	}
}

func BenchmarkComponentContent_JSON(b *testing.B) {
	txt := NewTextDisplay("**Bold Notice**")
	data, _ := json.Marshal(&txt)

	var parsed TextDisplay
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&txt)
	}
}
