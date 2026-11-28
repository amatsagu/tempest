package tempest

import (
	"fmt"
	"slices"
	"testing"
)

func TestPatternRouterMatchingAndPrecedence(t *testing.T) {
	t.Parallel()

	r := newPatternRouter[string]()

	mustReg := func(pat, id string) {
		t.Helper()
		if err := r.register(pat, id); err != nil {
			t.Fatalf("register(%q): %v", pat, err)
		}
	}

	mustReg("ban/{user}/confirm", "ban-confirm")
	mustReg("ban/{user}", "ban-root")
	mustReg("ban/exact", "ban-exact")
	mustReg("ticket:{guild}:{action}", "ticket-action")
	mustReg("ticket:{rest...}", "ticket-rest")
	mustReg("{kind}-{a}-{b}", "dash")

	tests := []struct {
		wantParams map[string]string
		id         string
		wantID     string
		wantOK     bool
	}{
		{
			id:     "ban/exact",
			wantID: "ban-exact",
			wantOK: true,
		},
		{
			id:         "ban/12345/confirm",
			wantID:     "ban-confirm",
			wantParams: map[string]string{"user": "12345"},
			wantOK:     true,
		},
		{
			id:         "ban/12345",
			wantID:     "ban-root",
			wantParams: map[string]string{"user": "12345"},
			wantOK:     true,
		},
		{
			id:         "ticket:999:close",
			wantID:     "ticket-action",
			wantParams: map[string]string{"guild": "999", "action": "close"},
			wantOK:     true,
		},
		{
			id:         "ticket:onlyone",
			wantID:     "ticket-rest",
			wantParams: map[string]string{"rest": "onlyone"},
			wantOK:     true,
		},
		{
			id:         "role-10-toggle",
			wantID:     "dash",
			wantParams: map[string]string{"kind": "role", "a": "10", "b": "toggle"},
			wantOK:     true,
		},
		{
			id:     "ban/",
			wantOK: false,
		},
		{
			id:         "ticket::close",
			wantID:     "ticket-rest",
			wantParams: map[string]string{"rest": ":close"},
			wantOK:     true,
		},
		{
			id:     "unknown",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		got, params, extra, n, ok := r.match(tc.id)
		if ok != tc.wantOK {
			t.Errorf("match(%q) ok = %t, want %t (handler=%q)", tc.id, ok, tc.wantOK, got)
			continue
		}
		if !ok {
			continue
		}
		if got != tc.wantID {
			t.Errorf("match(%q) handler = %q, want %q", tc.id, got, tc.wantID)
		}

		ic := interactionContext{params: params, extraParams: extra, paramCount: n}
		for k, wantV := range tc.wantParams {
			if gotV, exists := ic.param(k); !exists || gotV != wantV {
				t.Errorf("match(%q) param %q = (%q, %t), want (%q, true)", tc.id, k, gotV, exists, wantV)
			}
		}
	}

	// Unregister catch-all so ticket::close verifies empty-segment rejection.
	if !r.unregister("ticket:{rest...}") {
		t.Fatal("expected unregister catch-all to succeed")
	}
	if _, _, _, _, ok := r.match("ticket::close"); ok {
		t.Fatal("expected ticket::close not to match ticket:{guild}:{action}")
	}

	// Overflow beyond maxInlinePatternParams (> 8 params).
	many := newPatternRouter[string]()
	if err := many.register("p/{p1}/{p2}/{p3}/{p4}/{p5}/{p6}/{p7}/{p8}/{p9}/{p10}", "ten"); err != nil {
		t.Fatalf("register 10-param pattern: %v", err)
	}
	got, params, extra, n, ok := many.match("p/1/2/3/4/5/6/7/8/9/10")
	if !ok || got != "ten" || n != 10 {
		t.Fatalf("10-param match = (%q, n=%d, ok=%t)", got, n, ok)
	}
	ic := interactionContext{params: params, extraParams: extra, paramCount: n}
	for i := 1; i <= 10; i++ {
		if v, has := ic.param(fmt.Sprintf("p%d", i)); !has || v != fmt.Sprintf("%d", i) {
			t.Fatalf("param p%d = (%q, %t), want %d", i, v, has, i)
		}
	}

	// Unregister & clear lifecycle.
	if !r.unregister("ban/{user}/confirm") {
		t.Fatal("expected unregister wildcard to succeed")
	}
	if r.unregister("ban/{user}/confirm") {
		t.Fatal("expected second unregister to return false")
	}
	if !r.unregister("ban/exact") {
		t.Fatal("expected unregister exact to succeed")
	}

	var keys []string
	for k := range r.all() {
		keys = append(keys, k)
	}
	if !slices.Contains(keys, "ban/{user}") || slices.Contains(keys, "ban/exact") {
		t.Fatalf("unexpected keys after unregister: %v", keys)
	}

	r.clear()
	for k := range r.all() {
		t.Fatalf("expected empty router after clear, got %q", k)
	}
}

func TestPatternValidationErrors(t *testing.T) {
	t.Parallel()

	r := newPatternRouter[string]()
	invalid := []string{
		"",
		"{}",
		"open/{id",
		"open/id}",
		"open/{{id}}",
		"{a}{b}",
		"open/{rest...}/more",
		"dup/{id}/{id}",
	}
	for _, pat := range invalid {
		if err := r.register(pat, "h"); err == nil {
			t.Errorf("expected register(%q) to fail", pat)
		}
	}

	if err := r.register("dup/{a}", "first"); err != nil {
		t.Fatal(err)
	}
	if err := r.register("dup/{b}", "conflict"); err == nil {
		t.Fatal("expected skeleton collision error for dup/{b}")
	}
	if err := r.register("exact", "first"); err != nil {
		t.Fatal(err)
	}
	if err := r.register("exact", "dup"); err == nil {
		t.Fatal("expected duplicate exact key error")
	}
}
