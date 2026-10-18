package tempest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/ratelimit"
	"github.com/amatsagu/tempest/rest"
	"github.com/amatsagu/tempest/route"
)

// TestDeferOnGatewayPostsCallbackImmediately proves the gateway callback fires
// before the handler returns, without a second callback.
func TestDeferOnGatewayPostsCallbackImmediately(t *testing.T) {
	type callback struct {
		path string
		body string
	}
	callbacks := make(chan callback, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		callbacks <- callback{path: r.URL.Path, body: string(body)}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newGatewayClient(t, Config{})
	c.rest = rest.New("t", rest.Options{
		BaseURL: server.URL,
		Limiter: ratelimit.New(ratelimit.Options{GlobalRate: 1e9}),
	})

	entered := make(chan struct{})
	release := make(chan struct{})

	if err := c.Command("slow", func(_ context.Context, itx *discord.CommandInteraction) error {
		if err := itx.Defer(false); err != nil {
			t.Errorf("Defer: %v", err)
		}
		close(entered)
		<-release

		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	done := make(chan struct{})
	go func() {
		c.handleGatewayInteraction(c.runCtx, 0, []byte(`{"id":"1","application_id":"2","type":2,"token":"tok","data":{"id":"3","name":"slow","type":1}}`))
		close(done)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never ran")
	}

	select {
	case got := <-callbacks:
		if got.path != "/interactions/1/tok/callback" {
			t.Fatalf("callback path = %s", got.path)
		}

		if !strings.Contains(got.body, `"type":5`) {
			t.Fatalf("callback body = %s, want a deferred acknowledgement", got.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the callback waited for the handler to return")
	}

	close(release)
	<-done

	select {
	case got := <-callbacks:
		t.Fatalf("the return value produced a second callback: %+v", got)
	case <-time.After(100 * time.Millisecond):
	}
}

// newGatewayClient builds a gateway-mode client without a live connection.
func newGatewayClient(t *testing.T, cfg Config) *Client {
	t.Helper()

	cfg.Gateway = true
	if cfg.Token == "" {
		cfg.Token = "x"
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.runCtx = context.Background()
	return c
}

// reqRecord is one request a recording REST server received.
type reqRecord struct {
	method string
	path   string
	body   string
}

// recorder is an httptest handler that remembers every request it serves.
type recorder struct {
	reqs []reqRecord
	mu   sync.Mutex
}

func (rec *recorder) serveHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	rec.mu.Lock()
	rec.reqs = append(rec.reqs, reqRecord{method: r.Method, path: r.URL.Path, body: string(body)})
	rec.mu.Unlock()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{}"))
}

func (rec *recorder) find(method, path string) (reqRecord, bool) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, req := range rec.reqs {
		if req.method == method && req.path == path {
			return req, true
		}
	}
	return reqRecord{}, false
}

func (rec *recorder) paths(method string) []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var out []string
	for _, req := range rec.reqs {
		if req.method == method {
			out = append(out, req.path)
		}
	}
	return out
}

// newRecorder starts a recording REST server and a client pointed at it.
func newRecorder(t *testing.T) (*recorder, *rest.Client) {
	t.Helper()
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(rec.serveHTTP))
	t.Cleanup(server.Close)

	return rec, rest.New("t", rest.Options{
		BaseURL: server.URL,
		Limiter: ratelimit.New(ratelimit.Options{GlobalRate: 1e9}),
	})
}

// TestReplyBeforeFirstReplySendsInitialResponse proves Reply is a type 4 first response.
func TestReplyBeforeFirstReplySendsInitialResponse(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	if err := app.client.Command("hello", func(_ context.Context, itx *discord.CommandInteraction) error {
		return itx.Reply(discord.ResponseMessageData{Content: "first"}, true)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	rec := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"hello","type":1}}`)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"type":4`) || !strings.Contains(body, `"content":"first"`) {
		t.Fatalf("status=%d body=%s, want the initial channel message", rec.Code, body)
	}
}

// TestReplyAfterDeferEditsOriginal proves Reply PATCHes the original after a Defer.
func TestReplyAfterDeferEditsOriginal(t *testing.T) {
	rec, restClient := newRecorder(t)
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})
	app.client.rest = restClient

	// The handler keeps running after ServeHTTP returns; signal when Reply is done.
	done := make(chan error, 1)
	if err := app.client.Command("slow", func(_ context.Context, itx *discord.CommandInteraction) error {
		if err := itx.Defer(false); err != nil {
			done <- err
			return err
		}
		err := itx.Reply(discord.ResponseMessageData{Content: "finished"}, false)
		done <- err
		return err
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	res := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"slow","type":1}}`)
	if !strings.Contains(res.Body.String(), `"type":5`) {
		t.Fatalf("body = %s, want the deferred acknowledgement", res.Body.String())
	}

	if err := <-done; err != nil {
		t.Fatalf("Reply after Defer: %v", err)
	}

	edit, ok := rec.find(http.MethodPatch, "/webhooks/2/t/messages/@original")
	if !ok {
		t.Fatalf("no edit request, got %v", rec.paths(http.MethodPatch))
	}

	if !strings.Contains(edit.body, `"content":"finished"`) {
		t.Fatalf("edit body = %s, want the finished message", edit.body)
	}
}

// TestReplyAfterDeferEditsOriginalGateway covers the same swap over the gateway.
func TestReplyAfterDeferEditsOriginalGateway(t *testing.T) {
	rec, restClient := newRecorder(t)
	c := newGatewayClient(t, Config{})
	c.rest = restClient

	if err := c.Command("slow", func(_ context.Context, itx *discord.CommandInteraction) error {
		if err := itx.Defer(false); err != nil {
			t.Fatalf("Defer: %v", err)
		}
		return itx.Reply(discord.ResponseMessageData{Content: "finished"}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	c.handleGatewayInteraction(c.runCtx, 0, []byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"slow","type":1}}`))

	if _, ok := rec.find(http.MethodPost, "/interactions/1/t/callback"); !ok {
		t.Fatalf("no callback posted, got %v", rec.paths(http.MethodPost))
	}
	edit, ok := rec.find(http.MethodPatch, "/webhooks/2/t/messages/@original")
	if !ok {
		t.Fatalf("no edit request, got %v", rec.paths(http.MethodPatch))
	}

	if !strings.Contains(edit.body, `"content":"finished"`) {
		t.Fatalf("edit body = %s, want the finished message", edit.body)
	}
}

// TestGatewayFirstUpdatePostsCallbackNotPatch proves a first itx.Update uses the
// callback POST, never a PATCH before acknowledgement (which Discord rejects).
func TestGatewayFirstUpdatePostsCallbackNotPatch(t *testing.T) {
	rec, restClient := newRecorder(t)
	c := newGatewayClient(t, Config{})
	c.rest = restClient

	if err := c.Component("calc/{session}/{key}", func(_ context.Context, itx *discord.ComponentInteraction, params *route.Params) error {
		// The multi-byte × must survive the router's '/' and ':' split intact.
		if key, _ := params.Get("key"); key != "×" {
			t.Errorf("key = %q, want the multi-byte × intact", key)
		}
		return itx.Update(discord.ResponseMessageData{Content: "7"})
	}); err != nil {
		t.Fatalf("Component: %v", err)
	}

	c.handleGatewayInteraction(c.runCtx, 0, []byte(`{"id":"1","application_id":"2","type":3,"token":"tok","data":{"custom_id":"calc:s:×","component_type":2}}`))

	callback, ok := rec.find(http.MethodPost, "/interactions/1/tok/callback")
	if !ok {
		t.Fatalf("no callback posted, got %v", rec.paths(http.MethodPost))
	}

	if !strings.Contains(callback.body, `"type":7`) {
		t.Fatalf("callback body = %s, want an update-message response", callback.body)
	}

	if edits := rec.paths(http.MethodPatch); len(edits) != 0 {
		t.Fatalf("first reply PATCHed the original: %v", edits)
	}
}

// TestGatewaySecondReplyPatchesOriginal proves only a second reply PATCHes the original.
func TestGatewaySecondReplyPatchesOriginal(t *testing.T) {
	rec, restClient := newRecorder(t)
	c := newGatewayClient(t, Config{})
	c.rest = restClient

	if err := c.Component("calc/{session}/{key}", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		if err := itx.Update(discord.ResponseMessageData{Content: "first"}); err != nil {
			return err
		}
		return itx.Reply(discord.ResponseMessageData{Content: "second"}, false)
	}); err != nil {
		t.Fatalf("Component: %v", err)
	}

	c.handleGatewayInteraction(c.runCtx, 0, []byte(`{"id":"1","application_id":"2","type":3,"token":"tok","data":{"custom_id":"calc:s:×","component_type":2}}`))

	callback, ok := rec.find(http.MethodPost, "/interactions/1/tok/callback")
	if !ok {
		t.Fatalf("no callback posted, got %v", rec.paths(http.MethodPost))
	}

	if !strings.Contains(callback.body, `"content":"first"`) {
		t.Fatalf("callback body = %s, want the first reply", callback.body)
	}

	edit, ok := rec.find(http.MethodPatch, "/webhooks/2/tok/messages/@original")
	if !ok {
		t.Fatalf("no edit request, got %v", rec.paths(http.MethodPatch))
	}

	if !strings.Contains(edit.body, `"content":"second"`) {
		t.Fatalf("edit body = %s, want the second reply", edit.body)
	}
}

// TestDeferTwiceReturnsErrResponded proves the first reply is single-shot.
func TestDeferTwiceReturnsErrResponded(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	second := make(chan error, 1)
	if err := app.client.Command("twice", func(_ context.Context, itx *discord.CommandInteraction) error {
		if err := itx.Defer(false); err != nil {
			t.Errorf("first Defer: %v", err)
		}
		second <- itx.Defer(true)
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"twice","type":1}}`)
	if err := <-second; !errors.Is(err, ErrResponded) {
		t.Fatalf("second Defer = %v, want ErrResponded", err)
	}
}

// TestNilReturnWithoutReplySendsImplicitDefer proves a silent nil return still defers.
func TestNilReturnWithoutReplySendsImplicitDefer(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	if err := app.client.Command("quiet", func(context.Context, *discord.CommandInteraction) error {
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	rec := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"quiet","type":1}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"type":5`) {
		t.Fatalf("status=%d body=%s, want an implicit deferred acknowledgement", rec.Code, rec.Body.String())
	}
}

// TestHandlerErrorCallsOnInteractionError proves the hook gets the error, route, and
// typed pointer.
func TestHandlerErrorCallsOnInteractionError(t *testing.T) {
	errCh := make(chan error, 1)
	infoCh := make(chan InteractionInfo, 1)
	itxCh := make(chan any, 1)

	app := newTestApp(t, Config{
		HTTPAddr: "127.0.0.1:0",
		OnInteractionError: func(err error, info InteractionInfo, itx any) {
			errCh <- err
			infoCh <- info
			itxCh <- itx
		},
	})

	want := errors.New("boom")
	if err := app.client.Command("explode", func(context.Context, *discord.CommandInteraction) error {
		return want
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	rec := app.send(t, `{"id":"42","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"explode","type":1}}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (the hook did not reply)", rec.Code)
	}

	select {
	case gotErr := <-errCh:
		gotInfo := <-infoCh
		gotItx := <-itxCh

		if !errors.Is(gotErr, want) {
			t.Fatalf("err = %v, want boom", gotErr)
		}

		if gotInfo.Route.Pattern != "explode" || gotInfo.Kind != discord.InteractionApplicationCommand {
			t.Fatalf("info = %+v", gotInfo)
		}

		if gotInfo.ID.String() != "42" {
			t.Fatalf("info ID = %s, want 42", gotInfo.ID)
		}
		typed, ok := gotItx.(*discord.CommandInteraction)
		if !ok {
			t.Fatalf("itx type = %T, want *discord.CommandInteraction", gotItx)
		}

		if typed.Data.Name != "explode" {
			t.Fatalf("typed name = %q", typed.Data.Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnInteractionError was never called")
	}
}

// TestDebugDefaultErrorHookLogsAndReplies proves the default debug hook logs and replies.
func TestDebugDefaultErrorHookLogsAndReplies(t *testing.T) {
	var logs []string

	app := newTestApp(t, Config{
		HTTPAddr: "127.0.0.1:0",
		Debug:    true,
		Logf:     func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) },
	})

	if err := app.client.Command("explode", func(context.Context, *discord.CommandInteraction) error {
		return errors.New("kaboom")
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	rec := app.send(t, `{"id":"7","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"explode","type":1}}`)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Something went wrong.") {
		t.Fatalf("status=%d body=%s, want the generic error reply", rec.Code, body)
	}

	if !strings.Contains(strings.Join(logs, "\n"), "explode") {
		t.Fatalf("logs = %v, want the failed route", logs)
	}
}
