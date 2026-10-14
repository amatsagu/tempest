package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestEmbed(t *testing.T) {
	emb := Embed{
		Title:       "Test Embed",
		Description: "Embed description",
		Color:       0x00ff00,
		Fields: []EmbedField{
			{Name: "Field1", Value: "Val1", Inline: true},
			{Name: "Field2", Value: "Val2", Inline: true},
		},
		Footer: &EmbedFooter{Text: "Footer text"},
	}

	data, err := json.Marshal(emb)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Embed
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Title != emb.Title || len(parsed.Fields) != 2 || parsed.Footer == nil || parsed.Footer.Text != "Footer text" {
		t.Errorf("parsed = %+v, want %+v", parsed, emb)
	}
}

func BenchmarkEmbed_JSON(b *testing.B) {
	emb := Embed{
		Title:       "Test Embed",
		Description: "Embed description",
		Color:       0x00ff00,
		Fields: []EmbedField{
			{Name: "Field1", Value: "Val1", Inline: true},
			{Name: "Field2", Value: "Val2", Inline: true},
		},
		Footer: &EmbedFooter{Text: "Footer text"},
	}
	data, _ := json.Marshal(&emb)

	var parsed Embed
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&emb)
	}
}
