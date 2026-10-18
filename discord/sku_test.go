package discord

import (
	"encoding/json/v2"
	"testing"
)

// TestSKUParsesReference decodes the SKU payload from Discord's reference and
// checks the derived helpers against its flags.
func TestSKUParsesReference(t *testing.T) {
	raw := `{
		"id": "1088510058284990888",
		"type": 5,
		"dependent_sku_id": null,
		"application_id": "788708323867885999",
		"manifest_labels": null,
		"access_type": 1,
		"name": "Test Premium",
		"features": [],
		"release_date": null,
		"premium": false,
		"slug": "test-premium",
		"flags": 128,
		"show_age_gate": false
	}`

	var sku SKU
	if err := json.Unmarshal([]byte(raw), &sku); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if sku.ID != (SkuID{Snowflake: 1088510058284990888}) {
		t.Errorf("ID = %s", sku.ID)
	}

	if sku.Type != SKUSubscription || !sku.IsSubscription() {
		t.Errorf("Type = %d, IsSubscription = %v; want a subscription", sku.Type, sku.IsSubscription())
	}

	if sku.Name != "Test Premium" || sku.Slug != "test-premium" {
		t.Errorf("Name/Slug = %q/%q", sku.Name, sku.Slug)
	}

	// Flags 128 is the guild subscription bit, with no AVAILABLE bit.
	if !sku.GuildSubscription() {
		t.Error("GuildSubscription = false, want true")
	}

	if sku.UserSubscription() {
		t.Error("UserSubscription = true, want false")
	}

	if sku.Available() {
		t.Error("Available = true, want false")
	}
}

// TestSKUSubscriptionGroupIsNotASubscription proves a group SKU is detected, since
// Discord returns one alongside each real subscription SKU.
func TestSKUSubscriptionGroupIsNotASubscription(t *testing.T) {
	group := SKU{Type: SKUSubscriptionGroup, Flags: SKUFlagAvailable}
	if group.IsSubscription() {
		t.Error("IsSubscription = true for a subscription group")
	}

	if !group.Available() {
		t.Error("Available = false with the available flag set")
	}

	user := SKU{Type: SKUSubscription, Flags: SKUFlagUserSubscription}
	if !user.UserSubscription() || user.GuildSubscription() {
		t.Errorf("user subscription flags = %d", user.Flags)
	}
}

func TestSKURoundTrip(t *testing.T) {
	sku := SKU{
		ID:            SkuID{Snowflake: 8001},
		ApplicationID: AppID{Snowflake: 9001},
		Name:          "Premium",
		Slug:          "premium",
		Flags:         SKUFlagAvailable.Add(SKUFlagGuildSubscription),
		AccessType:    1,
		Type:          SKUSubscription,
		Premium:       true,
	}

	data, err := json.Marshal(sku)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var parsed SKU
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if parsed.ID != sku.ID || parsed.Type != sku.Type || parsed.Flags != sku.Flags {
		t.Errorf("parsed = %+v, want %+v", parsed, sku)
	}

	if !parsed.Premium || parsed.AccessType != 1 {
		t.Errorf("Premium/AccessType = %v/%d", parsed.Premium, parsed.AccessType)
	}
}
