package discord

import (
	"encoding/json/v2"
	"testing"
	"time"
)

// TestSubscriptionParsesReference decodes the subscription payload from Discord's
// reference, whose id lists are arrays of strings and whose nullable members are
// null.
func TestSubscriptionParsesReference(t *testing.T) {
	raw := `{
		"id": "1278078770116427839",
		"user_id": "1088605110638227537",
		"sku_ids": ["1158857122189168803"],
		"entitlement_ids": ["1158857122189168804"],
		"renewal_sku_ids": null,
		"current_period_start": "2024-08-27T19:48:44.406602+00:00",
		"current_period_end": "2024-09-27T19:48:44.406602+00:00",
		"status": 0,
		"canceled_at": null
	}`

	var sub Subscription
	if err := json.Unmarshal([]byte(raw), &sub); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if sub.ID != (SubscriptionID{Snowflake: 1278078770116427839}) {
		t.Errorf("ID = %s, want 1278078770116427839", sub.ID)
	}

	if sub.UserID != (UserID{Snowflake: 1088605110638227537}) {
		t.Errorf("UserID = %s, want 1088605110638227537", sub.UserID)
	}

	if len(sub.SkuIDs) != 1 || sub.SkuIDs[0] != (SkuID{Snowflake: 1158857122189168803}) {
		t.Errorf("SkuIDs = %v", sub.SkuIDs)
	}

	if len(sub.EntitlementIDs) != 1 || sub.EntitlementIDs[0] != (EntitlementID{Snowflake: 1158857122189168804}) {
		t.Errorf("EntitlementIDs = %v", sub.EntitlementIDs)
	}

	if sub.Status != SubscriptionActive {
		t.Errorf("Status = %d, want active", sub.Status)
	}

	if sub.CanceledAt != nil {
		t.Errorf("CanceledAt = %v, want nil", sub.CanceledAt)
	}

	if sub.RenewalSkuIDs != nil {
		t.Errorf("RenewalSkuIDs = %v, want nil for a null member", sub.RenewalSkuIDs)
	}
}

func TestSubscriptionRoundTrip(t *testing.T) {
	canceled := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	sub := Subscription{
		ID:                 SubscriptionID{Snowflake: 7001},
		UserID:             UserID{Snowflake: 1001},
		SkuIDs:             []SkuID{{Snowflake: 8001}},
		EntitlementIDs:     []EntitlementID{{Snowflake: 7002}},
		RenewalSkuIDs:      []SkuID{{Snowflake: 8002}},
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
		CanceledAt:         &canceled,
		Status:             SubscriptionEnding,
		Country:            "PL",
	}

	data, err := json.Marshal(sub)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var parsed Subscription
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if parsed.ID != sub.ID || parsed.Status != sub.Status || parsed.Country != sub.Country {
		t.Errorf("parsed = %+v, want %+v", parsed, sub)
	}

	if !parsed.CurrentPeriodEnd.Equal(end) || parsed.CanceledAt == nil || !parsed.CanceledAt.Equal(canceled) {
		t.Errorf("period end/canceled = %v/%v", parsed.CurrentPeriodEnd, parsed.CanceledAt)
	}

	if len(parsed.RenewalSkuIDs) != 1 || parsed.RenewalSkuIDs[0] != sub.RenewalSkuIDs[0] {
		t.Errorf("RenewalSkuIDs = %v", parsed.RenewalSkuIDs)
	}
}
