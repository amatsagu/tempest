package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestPermissions(t *testing.T) {
	p := PermissionSendMessages.Add(PermissionEmbedLinks, PermissionAttachFiles)
	if !p.Has(PermissionSendMessages, PermissionEmbedLinks, PermissionAttachFiles) {
		t.Errorf("expected permissions to have all added flags")
	}

	type Payload struct {
		Perms Permissions `json:"app_permissions,string"`
	}

	pl := Payload{Perms: p}
	data, err := json.Marshal(pl)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Payload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Perms != p {
		t.Errorf("got %d, want %d", parsed.Perms, p)
	}
}

func BenchmarkPermissions_JSON(b *testing.B) {
	type Payload struct {
		Perms Permissions `json:"app_permissions,string"`
	}

	pl := Payload{Perms: PermissionAdministrator | PermissionSendMessages}
	raw := []byte(`{"app_permissions":"2056"}`)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Payload
		_ = json.Unmarshal(raw, &parsed)
		_, _ = json.Marshal(pl)
	}
}
