package discord

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestResponseAutocomplete(t *testing.T) {
	resp := NewAutocompleteResponse(
		CommandOptionChoice{Name: "Option A", Value: "a"},
		CommandOptionChoice{Name: "Option B", Value: "b"},
	)

	if resp.Type != ResponseApplicationCommandAutocomplete {
		t.Errorf("got %d, want %d", resp.Type, ResponseApplicationCommandAutocomplete)
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
		CommandOptionChoice{Name: "Option A", Value: "a"},
		CommandOptionChoice{Name: "Option B", Value: "b"},
	)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(resp)
	}
}
