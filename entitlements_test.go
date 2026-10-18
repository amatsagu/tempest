package tempest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/ratelimit"
	"github.com/amatsagu/tempest/rest"
)

// callRecorder captures the last request a monetization helper made.
type callRecorder struct {
	query    url.Values
	method   string
	path     string
	body     string
	response string
	mu       sync.Mutex
}

func (r *callRecorder) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)

		r.mu.Lock()
		r.method = req.Method
		r.path = req.URL.Path
		r.query = req.URL.Query()
		r.body = string(body)
		response := r.response
		r.mu.Unlock()

		if response == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}
}

func (r *callRecorder) last() (method, path string, query url.Values, body string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.method, r.path, r.query, r.body
}

func newMonetizationClient(t *testing.T, recorder *callRecorder) *Client {
	t.Helper()

	server := httptest.NewServer(recorder.handler())
	t.Cleanup(server.Close)

	c, err := New(Config{Token: "x", ApplicationID: discord.AppID{Snowflake: 123}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	c.rest = rest.New("t", rest.Options{
		BaseURL: server.URL,
		Limiter: ratelimit.New(ratelimit.Options{GlobalRate: 1e9}),
	})
	return c
}

func TestSKUs(t *testing.T) {
	recorder := &callRecorder{response: `[{"id":"8001","type":5,"application_id":"123","name":"Premium","slug":"premium","flags":128}]`}
	c := newMonetizationClient(t, recorder)

	skus, err := c.SKUs(context.Background())
	if err != nil {
		t.Fatalf("SKUs: %v", err)
	}

	method, path, _, _ := recorder.last()
	if method != http.MethodGet || path != "/applications/123/skus" {
		t.Fatalf("call = %s %s, want GET /applications/123/skus", method, path)
	}

	if len(skus) != 1 || skus[0].ID != (discord.SkuID{Snowflake: 8001}) {
		t.Fatalf("skus = %+v", skus)
	}

	if !skus[0].IsSubscription() || !skus[0].GuildSubscription() {
		t.Errorf("sku helpers = %v/%v", skus[0].IsSubscription(), skus[0].GuildSubscription())
	}
}

// TestEntitlementsQuery proves every filter member reaches the query string with
// Discord's names, and that a nil filter sends none.
func TestEntitlementsQuery(t *testing.T) {
	recorder := &callRecorder{response: `[]`}
	c := newMonetizationClient(t, recorder)

	_, err := c.Entitlements(context.Background(), nil)
	if err != nil {
		t.Fatalf("Entitlements: %v", err)
	}

	method, path, query, _ := recorder.last()
	if method != http.MethodGet || path != "/applications/123/entitlements" {
		t.Fatalf("call = %s %s", method, path)
	}

	if len(query) != 0 {
		t.Fatalf("nil filter query = %v, want none", query)
	}

	filter := &discord.EntitlementFilter{
		UserID:         discord.UserID{Snowflake: 1},
		GuildID:        discord.GuildID{Snowflake: 2},
		Before:         discord.EntitlementID{Snowflake: 3},
		After:          discord.EntitlementID{Snowflake: 4},
		SkuIDs:         []discord.SkuID{{Snowflake: 5}, {Snowflake: 6}},
		Limit:          7,
		ExcludeEnded:   true,
		IncludeDeleted: true,
	}

	if _, err := c.Entitlements(context.Background(), filter); err != nil {
		t.Fatalf("Entitlements: %v", err)
	}

	want := map[string]string{
		"user_id":         "1",
		"guild_id":        "2",
		"before":          "3",
		"after":           "4",
		"sku_ids":         "5,6",
		"limit":           "7",
		"exclude_ended":   "true",
		"exclude_deleted": "false",
	}

	_, _, query, _ = recorder.last()
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("query %s = %q, want %q", key, got, value)
		}
	}
}

func TestEntitlementHelpers(t *testing.T) {
	recorder := &callRecorder{response: `{"id":"7001","sku_id":"8001","application_id":"123","user_id":"1","type":1}`}
	c := newMonetizationClient(t, recorder)

	entitlement, err := c.Entitlement(context.Background(), discord.EntitlementID{Snowflake: 7001})
	if err != nil {
		t.Fatalf("Entitlement: %v", err)
	}

	method, path, _, _ := recorder.last()
	if method != http.MethodGet || path != "/applications/123/entitlements/7001" {
		t.Fatalf("call = %s %s", method, path)
	}

	if entitlement.ID != (discord.EntitlementID{Snowflake: 7001}) {
		t.Errorf("ID = %s", entitlement.ID)
	}

	if err := c.ConsumeEntitlement(context.Background(), discord.EntitlementID{Snowflake: 7001}); err != nil {
		t.Fatalf("ConsumeEntitlement: %v", err)
	}

	method, path, _, _ = recorder.last()
	if method != http.MethodPost || path != "/applications/123/entitlements/7001/consume" {
		t.Fatalf("call = %s %s", method, path)
	}

	if err := c.DeleteTestEntitlement(context.Background(), discord.EntitlementID{Snowflake: 7001}); err != nil {
		t.Fatalf("DeleteTestEntitlement: %v", err)
	}

	method, path, _, _ = recorder.last()
	if method != http.MethodDelete || path != "/applications/123/entitlements/7001" {
		t.Fatalf("call = %s %s", method, path)
	}
}

// TestCreateTestEntitlement proves the payload uses Discord's owner_type, where 1
// is a guild and 2 is a user.
func TestCreateTestEntitlement(t *testing.T) {
	recorder := &callRecorder{response: `{"id":"7002","sku_id":"8001","application_id":"123","guild_id":"9","type":8}`}
	c := newMonetizationClient(t, recorder)

	entitlement, err := c.CreateTestEntitlement(context.Background(), discord.TestEntitlementPayload{
		SKUID:     discord.SkuID{Snowflake: 8001},
		OwnerID:   9,
		OwnerType: discord.EntitlementOwnerGuild,
	})
	if err != nil {
		t.Fatalf("CreateTestEntitlement: %v", err)
	}

	method, path, _, body := recorder.last()
	if method != http.MethodPost || path != "/applications/123/entitlements" {
		t.Fatalf("call = %s %s", method, path)
	}

	for _, want := range []string{`"sku_id":"8001"`, `"owner_id":"9"`, `"owner_type":1`} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want it to contain %s", body, want)
		}
	}

	if entitlement.GuildID != (discord.GuildID{Snowflake: 9}) {
		t.Errorf("GuildID = %s, want 9", entitlement.GuildID)
	}
}

func TestSubscriptions(t *testing.T) {
	recorder := &callRecorder{response: `[{"id":"7001","user_id":"1","sku_ids":["8001"],"entitlement_ids":[],"current_period_start":"2026-01-01T00:00:00Z","current_period_end":"2026-02-01T00:00:00Z","status":0}]`}
	c := newMonetizationClient(t, recorder)

	subscriptions, err := c.Subscriptions(context.Background(), discord.SkuID{Snowflake: 8001}, &discord.SubscriptionFilter{
		UserID: discord.UserID{Snowflake: 1},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Subscriptions: %v", err)
	}

	method, path, query, _ := recorder.last()
	if method != http.MethodGet || path != "/skus/8001/subscriptions" {
		t.Fatalf("call = %s %s, want GET /skus/8001/subscriptions", method, path)
	}

	if query.Get("user_id") != "1" || query.Get("limit") != "10" {
		t.Fatalf("query = %v", query)
	}

	if len(subscriptions) != 1 || subscriptions[0].Status != discord.SubscriptionActive {
		t.Fatalf("subscriptions = %+v", subscriptions)
	}

	if len(subscriptions[0].SkuIDs) != 1 || subscriptions[0].SkuIDs[0] != (discord.SkuID{Snowflake: 8001}) {
		t.Errorf("SkuIDs = %v", subscriptions[0].SkuIDs)
	}

	recorder.mu.Lock()
	recorder.response = `{"id":"7001","user_id":"1","sku_ids":["8001"],"current_period_end":"2026-02-01T00:00:00Z","status":2}`
	recorder.mu.Unlock()

	subscription, err := c.Subscription(context.Background(), discord.SkuID{Snowflake: 8001}, discord.SubscriptionID{Snowflake: 7001})
	if err != nil {
		t.Fatalf("Subscription: %v", err)
	}

	_, path, _, _ = recorder.last()
	if path != "/skus/8001/subscriptions/7001" {
		t.Fatalf("path = %s, want /skus/8001/subscriptions/7001", path)
	}

	if subscription.Status != discord.SubscriptionEnding {
		t.Errorf("Status = %d, want ending", subscription.Status)
	}
}

// TestMonetizationHelpersNeedToken proves the helpers report a missing token
// instead of panicking.
func TestMonetizationHelpersNeedToken(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.SKUs(context.Background()); err == nil {
		t.Error("SKUs without a token = nil error, want an error")
	}

	if _, err := c.Subscriptions(context.Background(), discord.SkuID{}, nil); err == nil {
		t.Error("Subscriptions without a token = nil error, want an error")
	}
}
