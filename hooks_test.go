package tempest

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

func TestLifecycleHooksCommand(t *testing.T) {
	var raw []InteractionInfo
	var completed []InteractionInfo
	completedCh := make(chan struct{}, 4)

	app := newTestApp(t, Config{
		HTTPAddr:               "127.0.0.1:0",
		OnRawInteraction:       func(info InteractionInfo) bool { raw = append(raw, info); return true },
		OnCompletedInteraction: func(info InteractionInfo) { completed = append(completed, info); completedCh <- struct{}{} },
	})

	if err := app.client.Command("hello", func(_ context.Context, itx *discord.CommandInteraction) error {
		return itx.Reply(discord.ResponseMessageData{}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	// A handled command fires both hooks; the request returns before completion.
	app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","guild_id":"5","channel_id":"6","data":{"id":"3","name":"hello","type":1}}`)
	waitForCompletion(t, completedCh)
	if len(raw) != 1 {
		t.Fatalf("raw calls = %d, want 1", len(raw))
	}

	if !raw[0].Handled {
		t.Fatalf("raw[0].Handled = false, want true")
	}

	if raw[0].Route.Kind != RouteCommand || raw[0].Route.Pattern != "hello" || !raw[0].Route.Exact {
		t.Fatalf("route = %+v", raw[0].Route)
	}

	if raw[0].CommandName != "hello" || raw[0].Kind != discord.InteractionApplicationCommand {
		t.Fatalf("info = %+v", raw[0])
	}

	if raw[0].GuildID.String() != "5" || raw[0].ChannelID.String() != "6" {
		t.Fatalf("guild/channel = %s/%s", raw[0].GuildID, raw[0].ChannelID)
	}

	if raw[0].Duration != 0 {
		t.Fatalf("raw duration = %v, want 0", raw[0].Duration)
	}

	if len(completed) != 1 {
		t.Fatalf("completed calls = %d, want 1", len(completed))
	}

	if completed[0].Duration <= 0 {
		t.Fatalf("completed duration = %v, want > 0", completed[0].Duration)
	}

	if completed[0].Response != discord.ResponseChannelMessageWithSource {
		t.Fatalf("completed response = %v, want channel message with source", completed[0].Response)
	}

	// A dropped command fires OnRawInteraction with Handled false, and never
	// reaches OnCompletedInteraction.
	raw = nil
	completed = nil
	app.send(t, `{"id":"2","application_id":"2","type":2,"token":"t","data":{"id":"4","name":"missing","type":1}}`)
	if len(raw) != 1 || raw[0].Handled {
		t.Fatalf("raw = %+v, want one dropped", raw)
	}

	if raw[0].Route.Kind != RouteNone {
		t.Fatalf("route = %+v, want RouteNone", raw[0].Route)
	}

	if len(completed) != 0 {
		t.Fatalf("completed calls = %d, want 0", len(completed))
	}

	// PING is excluded from both hooks.
	raw = nil
	completed = nil
	if rec := app.send(t, `{"id":"3","application_id":"2","type":1,"token":"t"}`); rec.Code != http.StatusOK {
		t.Fatalf("ping status = %d", rec.Code)
	}

	if len(raw) != 0 || len(completed) != 0 {
		t.Fatalf("ping fired hooks: raw=%d completed=%d", len(raw), len(completed))
	}
}

// waitForCompletion blocks until OnCompletedInteraction fires.
func waitForCompletion(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("OnCompletedInteraction never fired")
	}
}

// TestSlowHandlerWarnsInDebug checks the debug warning when a first reply risks
// Discord's 3 second deadline.
func TestSlowHandlerWarnsInDebug(t *testing.T) {
	var logged string

	debugClient, err := New(Config{
		Debug: true,
		Logf:  func(format string, args ...any) { logged = fmt.Sprintf(format, args...) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	info := InteractionInfo{Route: Route{Kind: RouteCommand, Pattern: "slow", Exact: true}}
	debugClient.finish(info, time.Now().Add(-3*time.Second), nil, discord.Response{Type: discord.ResponseDeferredChannelMessageWithSource})

	if !strings.Contains(logged, "within 3s") {
		t.Fatalf("log = %q, want a slow-handler warning", logged)
	}
}

// TestDeferThenSlowWorkDoesNotWarn proves the warning tracks the first reply, not
// the handler total: an immediate defer must not warn.
func TestDeferThenSlowWorkDoesNotWarn(t *testing.T) {
	var logged string
	var completed InteractionInfo
	var completedOK bool

	client, err := New(Config{
		Debug: true,
		Logf:  func(format string, args ...any) { logged = fmt.Sprintf(format, args...) },
		OnCompletedInteraction: func(info InteractionInfo) {
			completed = info
			completedOK = true
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	start := time.Now().Add(-10 * time.Second)
	// The first reply was delivered immediately; the handler then kept working.
	r := &responder{firstReplyAt: start}
	client.finish(InteractionInfo{Route: Route{Kind: RouteCommand, Pattern: "slow", Exact: true}}, start, r, discord.Response{Type: discord.ResponseDeferredChannelMessageWithSource})

	if strings.Contains(logged, "within 3s") {
		t.Fatalf("log = %q, want no warning for an immediate first reply", logged)
	}

	if !completedOK {
		t.Fatal("OnCompletedInteraction was never called")
	}

	if completed.Duration < 9*time.Second {
		t.Fatalf("Duration = %v, want the full handler time", completed.Duration)
	}
}

// TestLateFirstReplyWarns proves the warning fires when the first reply is late.
func TestLateFirstReplyWarns(t *testing.T) {
	var logged string

	client, err := New(Config{
		Debug: true,
		Logf:  func(format string, args ...any) { logged = fmt.Sprintf(format, args...) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	start := time.Now().Add(-3 * time.Second)
	r := &responder{firstReplyAt: time.Now()}
	client.finish(InteractionInfo{Route: Route{Kind: RouteCommand, Pattern: "late", Exact: true}}, start, r, discord.Response{Type: discord.ResponseChannelMessageWithSource})

	if !strings.Contains(logged, "within 3s") {
		t.Fatalf("log = %q, want a warning for a late first reply", logged)
	}
}

// TestDeferThenSlowWorkEndToEnd drives the real path: an early defer then slow work
// must not warn.
func TestDeferThenSlowWorkEndToEnd(t *testing.T) {
	var mu sync.Mutex
	var logs []string
	completed := make(chan struct{}, 1)

	app := newTestApp(t, Config{
		HTTPAddr: "127.0.0.1:0",
		Debug:    true,
		Logf: func(format string, args ...any) {
			mu.Lock()
			logs = append(logs, fmt.Sprintf(format, args...))
			mu.Unlock()
		},
		OnCompletedInteraction: func(InteractionInfo) { completed <- struct{}{} },
	})

	if err := app.client.Command("slow", func(_ context.Context, itx *discord.CommandInteraction) error {
		if err := itx.Defer(true); err != nil {
			return err
		}
		time.Sleep(2600 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	rec := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"slow","type":1}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	waitForCompletion(t, completed)

	mu.Lock()
	defer mu.Unlock()
	if strings.Contains(strings.Join(logs, "\n"), "within 3s") {
		t.Fatalf("logs = %v, want no slow-handler warning after an early defer", logs)
	}
}

// TestReplyAfterThresholdWarnsEndToEnd drives the real path with a late reply.
func TestReplyAfterThresholdWarnsEndToEnd(t *testing.T) {
	var mu sync.Mutex
	var logs []string
	completed := make(chan struct{}, 1)

	app := newTestApp(t, Config{
		HTTPAddr: "127.0.0.1:0",
		Debug:    true,
		Logf: func(format string, args ...any) {
			mu.Lock()
			logs = append(logs, fmt.Sprintf(format, args...))
			mu.Unlock()
		},
		OnCompletedInteraction: func(InteractionInfo) { completed <- struct{}{} },
	})

	if err := app.client.Command("late", func(_ context.Context, itx *discord.CommandInteraction) error {
		time.Sleep(2600 * time.Millisecond)
		return itx.Reply(discord.ResponseMessageData{Content: "late"}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"late","type":1}}`)
	waitForCompletion(t, completed)

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(strings.Join(logs, "\n"), "within 3s") {
		t.Fatalf("logs = %v, want a slow-handler warning", logs)
	}
}

func TestLifecycleHookVeto(t *testing.T) {
	var ran bool
	var completed int

	app := newTestApp(t, Config{
		HTTPAddr:               "127.0.0.1:0",
		OnRawInteraction:       func(InteractionInfo) bool { return false },
		OnCompletedInteraction: func(InteractionInfo) { completed++ },
	})

	if err := app.client.Command("hello", func(_ context.Context, itx *discord.CommandInteraction) error {
		ran = true
		return itx.Reply(discord.ResponseMessageData{}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	if err := app.client.Component("panel", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		ran = true
		return itx.Defer(false)
	}); err != nil {
		t.Fatalf("Component: %v", err)
	}

	// A vetoed command is dropped without acknowledgement, and its handler never
	// runs.
	rec := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"hello","type":1}}`)
	if ran {
		t.Fatal("vetoed command handler ran")
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("command status = %d, want 204", rec.Code)
	}

	// A vetoed component is dropped the same way.
	ran = false
	rec = app.send(t, `{"id":"2","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel","component_type":2}}`)
	if ran {
		t.Fatal("vetoed component handler ran")
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("component status = %d, want 204", rec.Code)
	}

	// A vetoed interaction is dropped, so it never reports completion.
	if completed != 0 {
		t.Fatalf("completed calls = %d, want 0", completed)
	}
}

func BenchmarkDispatchCommand(b *testing.B) {
	c, err := New(Config{})
	if err != nil {
		b.Fatal(err)
	}

	if err := c.Command("ping", func(context.Context, *discord.CommandInteraction) error {
		return nil
	}); err != nil {
		b.Fatal(err)
	}

	body := []byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := c.dispatch(ctx, body, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatchCommandWithHooks(b *testing.B) {
	c, err := New(Config{
		OnRawInteraction:       func(InteractionInfo) bool { return true },
		OnCompletedInteraction: func(InteractionInfo) {},
	})
	if err != nil {
		b.Fatal(err)
	}

	if err := c.Command("ping", func(context.Context, *discord.CommandInteraction) error {
		return nil
	}); err != nil {
		b.Fatal(err)
	}

	body := []byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := c.dispatch(ctx, body, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatchComponent(b *testing.B) {
	c, err := New(Config{})
	if err != nil {
		b.Fatal(err)
	}

	if err := c.Component("panel/{action}", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		return itx.Defer(false)
	}); err != nil {
		b.Fatal(err)
	}

	body := []byte(`{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel:reset","component_type":2}}`)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := c.dispatch(ctx, body, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatchComponentDropped(b *testing.B) {
	c, err := New(Config{})
	if err != nil {
		b.Fatal(err)
	}

	body := []byte(`{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"nope","component_type":2}}`)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := c.dispatch(ctx, body, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLifecycleHooksPatternRoute(t *testing.T) {
	var raw InteractionInfo

	app := newTestApp(t, Config{
		HTTPAddr:         "127.0.0.1:0",
		OnRawInteraction: func(info InteractionInfo) bool { raw = info; return true },
	})

	if err := app.client.Component("panel/{action}", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		return itx.Defer(false)
	}); err != nil {
		t.Fatalf("Component: %v", err)
	}

	app.send(t, `{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel:reset","component_type":2}}`)

	if !raw.Handled || raw.Route.Kind != RouteComponent {
		t.Fatalf("info = %+v", raw)
	}

	if raw.Route.Pattern != "panel/{action}" || raw.Route.Exact {
		t.Fatalf("route = %+v, want pattern match", raw.Route)
	}

	if raw.Params == nil {
		t.Fatal("params = nil, want captures")
	}

	if action, _ := raw.Params.Get("action"); action != "reset" {
		t.Fatalf("action = %q, want reset", action)
	}
}
