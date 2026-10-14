package discord

import (
	"encoding/json/v2"
	"testing"
	"time"
)

func TestMember(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	m := Member{
		User: &User{
			ID:         UserID{Snowflake: 1159846879854792704},
			Username:   "testuser",
			GlobalName: "Tester",
		},
		Nick:        "Captain",
		Roles:       []RoleID{{Snowflake: 2001}, {Snowflake: 2002}},
		JoinedAt:    now,
		Permissions: PermissionSendMessages,
	}

	if m.DisplayName() != "Captain" {
		t.Errorf("m.DisplayName() = %s, want Captain", m.DisplayName())
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Member
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Nick != m.Nick || len(parsed.Roles) != 2 || parsed.Permissions != m.Permissions {
		t.Errorf("parsed = %+v, want %+v", parsed, m)
	}
}

func BenchmarkMember_JSON(b *testing.B) {
	now := time.Now().UTC().Truncate(time.Second)
	m := Member{
		User: &User{
			ID:         UserID{Snowflake: 1159846879854792704},
			Username:   "testuser",
			GlobalName: "Tester",
		},
		Nick:        "Captain",
		Roles:       []RoleID{{Snowflake: 2001}, {Snowflake: 2002}},
		JoinedAt:    now,
		Permissions: PermissionSendMessages,
	}
	data, _ := json.Marshal(&m)

	var parsed Member
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&m)
	}
}
