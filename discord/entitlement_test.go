package discord

import (
	"encoding/json/v2"
	"testing"
	"time"
)

// TestEntitlementParsesReference decodes the entitlement payload from Discord's
// reference, including its nullable members and an app subscription type.
func TestEntitlementParsesReference(t *testing.T) {
	raw := `{
		"id": "1019653849998299136",
		"sku_id": "1019475255913222144",
		"application_id": "1019370614521200640",
		"user_id": "771129655544643584",
		"promotion_id": null,
		"type": 8,
		"deleted": false,
		"gift_code_flags": 0,
		"consumed": false,
		"starts_at": "2022-09-14T17:00:18.704163+00:00",
		"ends_at": "2022-10-14T17:00:18.704163+00:00",
		"guild_id": "1015034326372454400",
		"subscription_id": "1019653835926409216"
	}`

	var ent Entitlement
	if err := json.Unmarshal([]byte(raw), &ent); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if ent.ID != (EntitlementID{Snowflake: 1019653849998299136}) {
		t.Errorf("ID = %s, want 1019653849998299136", ent.ID)
	}

	if ent.Type != EntitlementApplicationSubscription {
		t.Errorf("Type = %d, want %d", ent.Type, EntitlementApplicationSubscription)
	}

	if ent.SubscriptionID != (SubscriptionID{Snowflake: 1019653835926409216}) {
		t.Errorf("SubscriptionID = %s, want 1019653835926409216", ent.SubscriptionID)
	}

	if ent.UserID != (UserID{Snowflake: 771129655544643584}) {
		t.Errorf("UserID = %s, want 771129655544643584", ent.UserID)
	}

	if ent.GuildID != (GuildID{Snowflake: 1015034326372454400}) {
		t.Errorf("GuildID = %s, want 1015034326372454400", ent.GuildID)
	}

	if ent.PromotionID != 0 {
		t.Errorf("PromotionID = %d, want 0 for a null member", ent.PromotionID)
	}

	if ent.StartsAt == nil || ent.EndsAt == nil {
		t.Fatalf("window = %v..%v, want both ends", ent.StartsAt, ent.EndsAt)
	}
}

func TestEntitlementRoundTrip(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ent := Entitlement{
		ID:             EntitlementID{Snowflake: 7001},
		SkuID:          SkuID{Snowflake: 8001},
		ApplicationID:  AppID{Snowflake: 9001},
		SubscriptionID: SubscriptionID{Snowflake: 7002},
		UserID:         UserID{Snowflake: 1001},
		StartsAt:       &start,
		Type:           EntitlementApplicationSubscription,
	}

	data, err := json.Marshal(ent)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var parsed Entitlement
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if parsed.ID != ent.ID || parsed.SkuID != ent.SkuID || parsed.Type != ent.Type {
		t.Errorf("parsed = %+v, want %+v", parsed, ent)
	}

	if parsed.SubscriptionID != ent.SubscriptionID {
		t.Errorf("SubscriptionID = %s, want %s", parsed.SubscriptionID, ent.SubscriptionID)
	}
}

// TestEntitlementActive covers the check premium features gate on.
func TestEntitlementActive(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	cases := []struct {
		name string
		ent  Entitlement
		want bool
	}{
		{"open ended", Entitlement{StartsAt: &start}, true},
		{"inside window", Entitlement{StartsAt: &start, EndsAt: &end}, true},
		{"starts later", Entitlement{StartsAt: &end}, false},
		{"already ended", Entitlement{StartsAt: &start, EndsAt: &start}, false},
		{"deleted", Entitlement{Deleted: true}, false},
		{"consumed", Entitlement{Consumed: true}, false},
	}

	at := start.Add(time.Hour)
	for _, tc := range cases {
		if got := tc.ent.Active(at); got != tc.want {
			t.Errorf("%s: Active = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestMonetizationValues pins the documented integers: a wrong constant grants
// nothing, or the wrong thing.
func TestMonetizationValues(t *testing.T) {
	entitlements := map[EntitlementType]uint8{
		EntitlementPurchase:                1,
		EntitlementPremiumSubscription:     2,
		EntitlementDeveloperGift:           3,
		EntitlementTestModePurchase:        4,
		EntitlementFreePurchase:            5,
		EntitlementUserGift:                6,
		EntitlementPremiumPurchase:         7,
		EntitlementApplicationSubscription: 8,
	}

	for got, want := range entitlements {
		if uint8(got) != want {
			t.Errorf("entitlement type = %d, want %d", got, want)
		}
	}

	skus := map[SKUType]uint8{
		SKUDurable:           2,
		SKUConsumable:        3,
		SKUSubscription:      5,
		SKUSubscriptionGroup: 6,
	}

	for got, want := range skus {
		if uint8(got) != want {
			t.Errorf("SKU type = %d, want %d", got, want)
		}
	}

	owners := map[EntitlementOwnerType]uint8{
		EntitlementOwnerGuild: 1,
		EntitlementOwnerUser:  2,
	}

	for got, want := range owners {
		if uint8(got) != want {
			t.Errorf("owner type = %d, want %d", got, want)
		}
	}

	statuses := map[SubscriptionStatus]uint8{
		SubscriptionActive:   0,
		SubscriptionInactive: 1,
		SubscriptionEnding:   2,
	}

	for got, want := range statuses {
		if uint8(got) != want {
			t.Errorf("subscription status = %d, want %d", got, want)
		}
	}
}

func BenchmarkEntitlement_JSON(b *testing.B) {
	ent := Entitlement{
		ID:            EntitlementID{Snowflake: 7001},
		SkuID:         SkuID{Snowflake: 8001},
		ApplicationID: AppID{Snowflake: 9001},
		UserID:        UserID{Snowflake: 1001},
		Type:          EntitlementApplicationSubscription,
	}
	data, _ := json.Marshal(&ent)

	var parsed Entitlement
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&ent)
	}
}
