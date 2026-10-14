package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestUser(t *testing.T) {
	u := User{
		ID:         UserID{Snowflake: 1159846879854792704},
		Username:   "amatsagu",
		GlobalName: "Amatsagu Dev",
		Avatar:     "abcdef123456",
		Bot:        true,
	}

	if u.ID.Mention() != "<@1159846879854792704>" {
		t.Errorf("u.ID.Mention() = %s, want <@1159846879854792704>", u.ID.Mention())
	}

	if u.EffectiveName() != "Amatsagu Dev" {
		t.Errorf("u.EffectiveName() = %s, want Amatsagu Dev", u.EffectiveName())
	}

	avatarURL := u.AvatarURL()
	expectedURL := CDNBaseURL + "/avatars/1159846879854792704/abcdef123456.png"
	if avatarURL != expectedURL {
		t.Errorf("u.AvatarURL() = %s, want %s", avatarURL, expectedURL)
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed User
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != u.ID || parsed.Username != u.Username || parsed.GlobalName != u.GlobalName {
		t.Errorf("parsed = %+v, want %+v", parsed, u)
	}
}

func BenchmarkUser_JSON(b *testing.B) {
	u := User{
		ID:         UserID{Snowflake: 1159846879854792704},
		Username:   "amatsagu",
		GlobalName: "Amatsagu Dev",
		Avatar:     "abcdef123456",
		Bot:        true,
	}
	data, _ := json.Marshal(&u)

	var parsed User
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&u)
	}
}
