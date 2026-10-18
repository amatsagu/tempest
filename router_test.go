package tempest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

func TestDynamicComponentRouting(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	var hit string
	handler := func(tag string) ComponentHandler {
		return func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
			hit = tag
			return itx.Defer(false)
		}
	}

	body := `{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"secret_abc","component_type":2}}`

	if err := app.client.Component("secret_abc", handler("first")); err != nil {
		t.Fatalf("Component: %v", err)
	}

	if rec := app.send(t, body); rec.Code != http.StatusOK || hit != "first" {
		t.Fatalf("after add: status=%d hit=%q", rec.Code, hit)
	}

	// Re-registering the same ID must swap the handler, not conflict.
	if err := app.client.Component("secret_abc", handler("second")); err != nil {
		t.Fatalf("re-register: %v", err)
	}
	hit = ""
	app.send(t, body)
	if hit != "second" {
		t.Fatalf("after replace: hit=%q, want second", hit)
	}

	// Removing the route drops the interaction without acknowledgement.
	app.client.RemoveComponent("secret_abc")
	hit = ""
	rec := app.send(t, body)
	if hit != "" {
		t.Fatalf("after remove: handler still fired (%q)", hit)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("after remove: status = %d, want 204", rec.Code)
	}
}

func TestDynamicModalRouting(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	var gotToken string
	if err := app.client.Modal("modal/{token}", func(_ context.Context, _ *discord.ModalInteraction, params *route.Params) error {
		gotToken, _ = params.Get("token")
		return nil
	}); err != nil {
		t.Fatalf("Modal: %v", err)
	}

	body := `{"id":"1","application_id":"2","type":5,"token":"t","data":{"custom_id":"modal:xyz"}}`
	if rec := app.send(t, body); rec.Code != http.StatusOK || gotToken != "xyz" {
		t.Fatalf("after add: status=%d token=%q", rec.Code, gotToken)
	}

	app.client.RemoveModal("modal/{token}")
	gotToken = ""
	rec := app.send(t, body)
	if gotToken != "" {
		t.Fatalf("after remove: handler still fired (%q)", gotToken)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("after remove: status = %d, want 204", rec.Code)
	}
}

func TestRouterPatternReplaceAndConflict(t *testing.T) {
	var r idRouter[int]

	if err := r.register("a/{x}", 1); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := r.register("a/{x}", 2); err != nil {
		t.Fatalf("replace: %v", err)
	}

	if h, _, ok := r.match("a:go", nil); !ok || h != 2 {
		t.Fatalf("after replace: h=%d ok=%v, want 2 true", h, ok)
	}

	// A conflicting wildcard name is rejected and leaves the original intact.
	if err := r.register("a/{y}", 3); err == nil {
		t.Fatal("expected conflict for a different wildcard name")
	}

	if h, _, ok := r.match("a:go", nil); !ok || h != 2 {
		t.Fatalf("after conflict: h=%d ok=%v, want 2 true", h, ok)
	}

	r.remove("a/{x}")
	if _, _, ok := r.match("a:go", nil); ok {
		t.Fatal("expected no match after remove")
	}
}

// TestRouterConcurrentMutation exercises add/remove/match concurrently; run with -race.
func TestRouterConcurrentMutation(t *testing.T) {
	var r idRouter[int]
	if err := r.register("panel/{action}", 1); err != nil {
		t.Fatal(err)
	}

	if err := r.register("fixed", 1); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var params route.Params
			for {
				select {
				case <-stop:
					return
				default:
				}

				if h, _, ok := r.match("fixed", nil); ok && h != 1 {
					t.Errorf("fixed = %d, want 1", h)
					return
				}

				if h, _, ok := r.match("panel:go", &params); ok && h != 1 {
					t.Errorf("panel = %d, want 1", h)
					return
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stop)
		for i := 0; i < 5000; i++ {
			id := fmt.Sprintf("dyn_%d", i)
			if err := r.register(id, i+2); err != nil {
				t.Errorf("register %s: %v", id, err)
				return
			}

			if h, _, ok := r.match(id, nil); !ok || h != i+2 {
				t.Errorf("dyn %s = %d (%v), want %d", id, h, ok, i+2)
				return
			}
			r.remove(id)
		}
	}()

	wg.Wait()
}

func TestCommandRegistrationRejectedAfterRun(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	noop := func(context.Context, *discord.CommandInteraction) error {
		return nil
	}

	if err := app.client.Command("ping", noop); err != nil {
		t.Fatalf("before run: %v", err)
	}

	app.client.running.Store(true)
	defer app.client.running.Store(false)

	if err := app.client.Command("late", noop); !errors.Is(err, ErrRunning) {
		t.Fatalf("Command after run: err = %v, want ErrRunning", err)
	}

	if err := app.client.CommandAutocomplete("late", func(context.Context, *discord.CommandInteraction) ([]discord.CommandOptionChoice, error) {
		return nil, nil
	}); !errors.Is(err, ErrRunning) {
		t.Fatalf("CommandAutocomplete after run: err = %v, want ErrRunning", err)
	}
}

func BenchmarkRouterExactMatch(b *testing.B) {
	var r idRouter[int]
	if err := r.register("panel_ok", 1); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, _, ok := r.match("panel_ok", nil); !ok {
			b.Fatal("miss")
		}
	}
}

func BenchmarkRouterPatternMatch(b *testing.B) {
	var r idRouter[int]
	if err := r.register("panel/{action}/{id}", 1); err != nil {
		b.Fatal(err)
	}

	var params route.Params
	b.ReportAllocs()
	for b.Loop() {
		if _, _, ok := r.match("panel:create:12345", &params); !ok {
			b.Fatal("miss")
		}
	}
}
