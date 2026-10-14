package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestEntitlement(t *testing.T) {
	ent := Entitlement{
		ID:            Snowflake(7001),
		SkuID:         SkuID{Snowflake: 8001},
		ApplicationID: AppID{Snowflake: 9001},
		UserID:        UserID{Snowflake: 1001},
		Type:          EntitlementApplicationSubscription,
	}

	data, err := json.Marshal(ent)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Entitlement
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != ent.ID || parsed.SkuID != ent.SkuID || parsed.Type != ent.Type {
		t.Errorf("parsed = %+v, want %+v", parsed, ent)
	}
}

func BenchmarkEntitlement_JSON(b *testing.B) {
	ent := Entitlement{
		ID:            Snowflake(7001),
		SkuID:         SkuID{Snowflake: 8001},
		ApplicationID: AppID{Snowflake: 9001},
		UserID:        UserID{Snowflake: 1001},
		Type:          EntitlementApplicationSubscription,
	}
	data, _ := json.Marshal(ent)

	b.ReportAllocs()
	for b.Loop() {
		var parsed Entitlement
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(ent)
	}
}
