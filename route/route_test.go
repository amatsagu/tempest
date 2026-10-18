package route_test

import (
	"testing"

	"github.com/amatsagu/tempest/route"
)

func TestStaticMatch(t *testing.T) {
	var r route.Router[string]
	if err := r.Handle("/panel/settings", "settings"); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var params route.Params
	h, ok := r.Match("panel:settings", &params)
	if !ok || h != "settings" {
		t.Fatalf("Match = %q, %v; want settings, true", h, ok)
	}

	if params.Len() != 0 {
		t.Errorf("params = %d, want 0", params.Len())
	}
}

func TestWildcardMatch(t *testing.T) {
	var r route.Router[string]
	if err := r.Handle("/panel/{action}", "panel"); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var params route.Params
	h, ok := r.Match("panel:create", &params)
	if !ok || h != "panel" {
		t.Fatalf("Match = %q, %v", h, ok)
	}

	if got, _ := params.Get("action"); got != "create" {
		t.Errorf("action = %q, want create", got)
	}
}

func TestMultiWildcard(t *testing.T) {
	var r route.Router[string]
	if err := r.Handle("/user/{id}/page/{page}", "page"); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var params route.Params
	h, ok := r.Match("/user/42/page/3", &params)
	if !ok || h != "page" {
		t.Fatalf("Match = %q, %v", h, ok)
	}

	if id, _ := params.Get("id"); id != "42" {
		t.Errorf("id = %q, want 42", id)
	}

	if pg, _ := params.Get("page"); pg != "3" {
		t.Errorf("page = %q, want 3", pg)
	}
}

func TestStaticPreferredOverWildcard(t *testing.T) {
	var r route.Router[string]
	if err := r.Handle("/panel/{action}", "wild"); err != nil {
		t.Fatal(err)
	}

	if err := r.Handle("/panel/reset", "static"); err != nil {
		t.Fatal(err)
	}

	var params route.Params
	h, _ := r.Match("panel:reset", &params)
	if h != "static" {
		t.Errorf("= %q, want static", h)
	}
}

func TestNoMatch(t *testing.T) {
	var r route.Router[string]
	_ = r.Handle("/panel/{action}", "panel")

	var params route.Params
	if _, ok := r.Match("/other/thing", &params); ok {
		t.Error("expected no match")
	}
}

func TestConflicts(t *testing.T) {
	var r route.Router[string]
	if err := r.Handle("/a/{x}", "one"); err != nil {
		t.Fatal(err)
	}

	if err := r.Handle("/a/{y}", "two"); err == nil {
		t.Error("expected conflict for two different wildcard names")
	}

	if err := r.Handle("/a/{x}", "again"); err == nil {
		t.Error("expected conflict for duplicate pattern")
	}
}

func TestInvalidPattern(t *testing.T) {
	var r route.Router[string]
	if err := r.Handle("", "x"); err == nil {
		t.Error("expected invalid pattern error for empty pattern")
	}

	if err := r.Handle("/{}", "x"); err == nil {
		t.Error("expected invalid pattern error for empty wildcard name")
	}
}

func TestRemove(t *testing.T) {
	var r route.Router[string]
	_ = r.Handle("/panel/{action}", "wild")
	_ = r.Handle("/panel/reset", "static")
	_ = r.Handle("/user/{id}/page/{page}", "page")

	var params route.Params

	if !r.Remove("/panel/reset") {
		t.Fatal("Remove(/panel/reset) = false, want true")
	}
	// The wildcard sibling must still match the freed path.
	if h, ok := r.Match("panel:reset", &params); !ok || h != "wild" {
		t.Errorf("Match = %q, %v; want wild, true", h, ok)
	}

	if r.Remove("/panel/reset") {
		t.Error("removing again should report false")
	}

	// A wildcard with a different name must not remove the existing node.
	if r.Remove("/panel/{other}") {
		t.Error("Remove with a different wildcard name should report false")
	}

	if _, ok := r.Match("panel:view", &params); !ok {
		t.Error("existing wildcard was removed by a mismatched name")
	}

	if !r.Remove("/user/{id}/page/{page}") {
		t.Fatal("Remove(deep pattern) = false, want true")
	}

	if _, ok := r.Match("/user/42/page/3", &params); ok {
		t.Error("deep pattern should no longer match")
	}
	// Pruned nodes must be reusable.
	if err := r.Handle("/user/{id}/other", "other"); err != nil {
		t.Fatalf("re-register after prune: %v", err)
	}

	if h, ok := r.Match("/user/7/other", &params); !ok || h != "other" {
		t.Errorf("Match = %q, %v; want other, true", h, ok)
	}

	if r.Remove("/nope") {
		t.Error("Remove(unknown) should report false")
	}

	if r.Remove("") {
		t.Error("Remove(empty) should report false")
	}
}

func BenchmarkMatch(b *testing.B) {
	var r route.Router[string]
	_ = r.Handle("/panel/{action}/{id}", "panel")

	var params route.Params
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Match("panel:create:12345", &params)
	}
}
