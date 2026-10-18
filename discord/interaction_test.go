package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestInteraction(t *testing.T) {
	raw := []byte(`{
		"id": "1159846879854792704",
		"application_id": "9999846879854792704",
		"type": 2,
		"token": "interaction_continuation_token_xyz",
		"version": 1,
		"context": 0,
		"locale": "en-US",
		"app_permissions": "1071698660929",
		"data": {
			"id": "1234",
			"name": "ping",
			"type": 1
		}
	}`)

	var itx Interaction
	if err := json.Unmarshal(raw, &itx); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if itx.ID.String() != "1159846879854792704" {
		t.Errorf("itx.ID = %s, want 1159846879854792704", itx.ID.String())
	}

	if itx.Type != InteractionApplicationCommand {
		t.Errorf("itx.Type = %d, want %d", itx.Type, InteractionApplicationCommand)
	}

	if itx.Token != "interaction_continuation_token_xyz" {
		t.Errorf("itx.Token mismatch")
	}

	if itx.Context != InteractionContextGuild {
		t.Errorf("itx.Context = %d, want %d", itx.Context, InteractionContextGuild)
	}

	if itx.Locale != LocaleEnglishUS {
		t.Errorf("itx.Locale = %s, want %s", itx.Locale, LocaleEnglishUS)
	}

	var cmdItx CommandInteraction
	if err := json.Unmarshal(raw, &cmdItx); err != nil {
		t.Fatalf("Unmarshal CommandInteraction failed: %v", err)
	}

	if cmdItx.Data.Name != "ping" {
		t.Errorf("cmdItx.Data.Name = %s, want ping", cmdItx.Data.Name)
	}
}

func BenchmarkInteraction_JSON(b *testing.B) {
	raw := []byte(`{
		"id": "1159846879854792704",
		"application_id": "9999846879854792704",
		"type": 2,
		"token": "interaction_continuation_token_xyz",
		"version": 1,
		"context": 0,
		"locale": "en-US",
		"app_permissions": "1071698660929"
	}`)

	var itx Interaction
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(raw, &itx)
	}
}
