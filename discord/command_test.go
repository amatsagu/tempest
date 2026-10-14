package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestCommand(t *testing.T) {
	choice := CommandOptionChoice{
		Name:  "Option One",
		Value: "val1",
		NameLocalizations: map[Locale]string{
			LocaleGerman: "Option Eins",
		},
	}

	data, err := json.Marshal(choice)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed CommandOptionChoice
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Name != choice.Name || parsed.Value != choice.Value || parsed.NameLocalizations[LocaleGerman] != "Option Eins" {
		t.Errorf("parsed = %+v, want %+v", parsed, choice)
	}
}

func BenchmarkCommand_JSON(b *testing.B) {
	choice := CommandOptionChoice{
		Name:  "Option One",
		Value: "val1",
		NameLocalizations: map[Locale]string{
			LocaleGerman: "Option Eins",
		},
	}
	data, _ := json.Marshal(&choice)

	var parsed CommandOptionChoice
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&choice)
	}
}
