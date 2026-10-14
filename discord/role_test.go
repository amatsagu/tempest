package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestRole(t *testing.T) {
	r := Role{
		ID:          RoleID{Snowflake: 3001},
		Name:        "Moderator",
		Color:       0x3498db,
		Permissions: PermissionManageMessages | PermissionKickMembers,
		Position:    5,
		Mentionable: true,
	}

	if r.Mention() != "<@&3001>" {
		t.Errorf("r.Mention() = %s, want <@&3001>", r.Mention())
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Role
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != r.ID || parsed.Name != r.Name || parsed.Permissions != r.Permissions {
		t.Errorf("parsed = %+v, want %+v", parsed, r)
	}
}

func BenchmarkRole_JSON(b *testing.B) {
	r := Role{
		ID:          RoleID{Snowflake: 3001},
		Name:        "Moderator",
		Color:       0x3498db,
		Permissions: PermissionManageMessages | PermissionKickMembers,
		Position:    5,
		Mentionable: true,
	}
	data, _ := json.Marshal(r)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Role
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(r)
	}
}
