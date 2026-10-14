package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestLocale(t *testing.T) {
	loc := LocaleEnglishUS
	if loc != "en-US" {
		t.Errorf("got %s, want en-US", loc)
	}

	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(data) != `"en-US"` {
		t.Errorf("got %s, want \"en-US\"", string(data))
	}

	var parsed Locale
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed != LocaleEnglishUS {
		t.Errorf("got %s, want %s", parsed, LocaleEnglishUS)
	}
}

func BenchmarkLocale_JSON(b *testing.B) {
	loc := LocaleEnglishUS
	raw := []byte(`"en-US"`)

	b.ReportAllocs()
	for b.Loop() {
		var l Locale
		_ = json.Unmarshal(raw, &l)
		_, _ = json.Marshal(loc)
	}
}
