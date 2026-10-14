package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestAllowedMentions(t *testing.T) {
	am := AllowedMentions{
		Parse:       []AllowedMentionType{AllowedMentionUsers},
		Roles:       []RoleID{{Snowflake: 101}},
		Users:       []UserID{{Snowflake: 202}},
		RepliedUser: true,
	}

	data, err := json.Marshal(am)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed AllowedMentions
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Parse) != 1 || len(parsed.Roles) != 1 || len(parsed.Users) != 1 || !parsed.RepliedUser {
		t.Errorf("parsed = %+v, want %+v", parsed, am)
	}
}

func BenchmarkAllowedMentions_JSON(b *testing.B) {
	am := AllowedMentions{
		Parse:       []AllowedMentionType{AllowedMentionUsers},
		Roles:       []RoleID{{Snowflake: 101}},
		Users:       []UserID{{Snowflake: 202}},
		RepliedUser: true,
	}
	data, _ := json.Marshal(&am)

	var parsed AllowedMentions
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&am)
	}
}
