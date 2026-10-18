package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

type testApp struct {
	client *Client
	sign   func(body []byte) (sig, ts string)
}

func newTestApp(t *testing.T, cfg Config) testApp {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	cfg.PublicKey = hex.EncodeToString(pub)

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return testApp{
		client: client,
		sign: func(body []byte) (string, string) {
			ts := "1700000000"
			msg := append([]byte(ts), body...)
			return hex.EncodeToString(ed25519.Sign(priv, msg)), ts
		},
	}
}

func (app testApp) send(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	app.client.ServeHTTP(rec, app.request(body))
	return rec
}

// request builds a correctly signed POST for an interaction body.
func (app testApp) request(body string) *http.Request {
	sig, ts := app.sign([]byte(body))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)
	return req
}

// signalWriter records when the first byte of a response is written.
type signalWriter struct {
	header http.Header
	body   *bytes.Buffer
	wrote  chan struct{}
	once   sync.Once
}

func newSignalWriter() *signalWriter {
	return &signalWriter{header: make(http.Header), body: new(bytes.Buffer), wrote: make(chan struct{})}
}

func (w *signalWriter) Header() http.Header { return w.header }

func (w *signalWriter) WriteHeader(int) {}

func (w *signalWriter) Write(b []byte) (int, error) {
	w.once.Do(func() { close(w.wrote) })
	return w.body.Write(b)
}

func TestHTTPPing(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	rec := app.send(t, `{"id":"1","application_id":"2","type":1,"token":"t"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	if got := strings.TrimSpace(rec.Body.String()); got != `{"type":1}` {
		t.Errorf("body = %s, want {\"type\":1}", got)
	}
}

func TestHTTPRejectsBadSignature(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":1}`))
	req.Header.Set("X-Signature-Ed25519", strings.Repeat("00", 64))
	req.Header.Set("X-Signature-Timestamp", "1700000000")

	rec := httptest.NewRecorder()
	app.client.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHTTPRejectsWrongMethod(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.client.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

// TestHTTPProbeAndDebugGuidance covers the endpoint probe and debug error guidance.
func TestHTTPProbeAndDebugGuidance(t *testing.T) {
	// Default mode: a GET answers with no content, revealing nothing.
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})
	rec := httptest.NewRecorder()
	app.client.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("probe status = %d, want 405", rec.Code)
	}

	if rec.Body.Len() != 0 {
		t.Fatalf("probe body = %q, want empty", rec.Body.String())
	}

	debugApp := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0", Debug: true, Logf: func(string, ...any) {}})

	// Debug mode: the GET probe confirms the endpoint is mounted.
	rec = httptest.NewRecorder()
	debugApp.client.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "endpoint is running") {
		t.Fatalf("debug probe: status=%d body=%q", rec.Code, rec.Body.String())
	}

	// Debug mode: a failed verification explains what to check.
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":1}`))
	req.Header.Set("X-Signature-Ed25519", strings.Repeat("00", 64))
	req.Header.Set("X-Signature-Timestamp", "1700000000")
	rec = httptest.NewRecorder()
	debugApp.client.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "Config.PublicKey") {
		t.Fatalf("debug signature: status=%d body=%q", rec.Code, rec.Body.String())
	}

	// Default mode: the same failure carries only the basic error.
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"type":1}`))
	req.Header.Set("X-Signature-Ed25519", strings.Repeat("00", 64))
	req.Header.Set("X-Signature-Timestamp", "1700000000")
	rec = httptest.NewRecorder()
	app.client.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), "Config.PublicKey") {
		t.Fatalf("default signature: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

// TestHTTPRejectsOversizedBody checks that a body above Config.MaxBodyBytes is refused.
func TestHTTPRejectsOversizedBody(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0", MaxBodyBytes: 128})

	// Under the limit the request is processed normally.
	if rec := app.send(t, `{"type":1}`); rec.Code != http.StatusOK {
		t.Fatalf("small body status = %d, want 200", rec.Code)
	}

	// Over the limit it is rejected without being dispatched.
	big := `{"type":1,"pad":"` + strings.Repeat("a", 512) + `"}`
	if rec := app.send(t, big); rec.Code != http.StatusBadRequest {
		t.Fatalf("big body status = %d, want 400", rec.Code)
	}

	// The pooled body buffer is reused cleanly after the rejection.
	if rec := app.send(t, `{"type":1}`); rec.Code != http.StatusOK {
		t.Fatalf("body after rejection status = %d, want 200", rec.Code)
	}
}

func TestHTTPCommandDispatch(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	var gotName string
	if err := app.client.Command("hello", func(_ context.Context, itx *discord.CommandInteraction) error {
		gotName = itx.Data.Name
		return itx.Reply(discord.ResponseMessageData{Content: "hi"}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	rec := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"hello","type":1}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	if gotName != "hello" {
		t.Errorf("handler got %q, want hello", gotName)
	}

	if !strings.Contains(rec.Body.String(), `"content":"hi"`) {
		t.Errorf("body = %s, want content hi", rec.Body.String())
	}
}

func TestHTTPComponentPrefixRouting(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	var gotParams string
	err := app.client.Component("/panel.test/{id}", func(_ context.Context, itx *discord.ComponentInteraction, params *route.Params) error {
		gotParams, _ = params.Get("id")
		return itx.Defer(false)
	})
	if err != nil {
		t.Fatalf("Component: %v", err)
	}

	body := `{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel.test:42","component_type":2}}`
	rec := app.send(t, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	if gotParams != "42" {
		t.Errorf("params[id] = %q, want 42", gotParams)
	}
}

func TestHTTPServerRoundtrip(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})
	if err := app.client.Command("ping", func(_ context.Context, itx *discord.CommandInteraction) error {
		return itx.Reply(discord.ResponseMessageData{Content: "pong"}, false)
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	server := httptest.NewServer(app.client)
	defer server.Close()

	sig, ts := app.sign([]byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`))

	req, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestConfigResolveAppliesDefaults(t *testing.T) {
	cfg := Config{Token: "x", PublicKey: "y"}
	res := cfg.resolve()

	if res.MaxBodyBytes != defaultMaxBodyBytes {
		t.Errorf("MaxBodyBytes = %d, want %d", res.MaxBodyBytes, defaultMaxBodyBytes)
	}

	if res.IdleConnsPerHost != defaultMaxIdleConnsPerHost {
		t.Errorf("IdleConnsPerHost = %d, want %d", res.IdleConnsPerHost, defaultMaxIdleConnsPerHost)
	}
}

func TestConfigResolveHonorsOverrides(t *testing.T) {
	cfg := Config{
		MaxBodyBytes:        2048,
		MaxIdleConnsPerHost: 3,
	}
	res := cfg.resolve()

	if res.MaxBodyBytes != 2048 {
		t.Errorf("MaxBodyBytes = %d, want 2048", res.MaxBodyBytes)
	}

	if res.IdleConnsPerHost != 3 {
		t.Errorf("IdleConnsPerHost = %d, want 3", res.IdleConnsPerHost)
	}
}

func TestNewRejectsMultipleModes(t *testing.T) {
	_, err := New(Config{
		PublicKey: strings.Repeat("00", 32),
		HTTPAddr:  "127.0.0.1:0",
		Gateway:   true,
	})

	if !errors.Is(err, ErrMultipleModes) {
		t.Fatalf("err = %v, want ErrMultipleModes", err)
	}
}

func TestHTTPExactAndPatternRouting(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	var exactHit, patternAction string

	if err := app.client.Component("panel_ok", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		exactHit = "yes"
		return itx.Defer(false)
	}); err != nil {
		t.Fatalf("register exact: %v", err)
	}

	if err := app.client.Component("panel/{action}", func(_ context.Context, itx *discord.ComponentInteraction, params *route.Params) error {
		patternAction, _ = params.Get("action")
		return itx.Defer(false)
	}); err != nil {
		t.Fatalf("register pattern: %v", err)
	}

	rec := app.send(t, `{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel_ok","component_type":2}}`)
	if rec.Code != http.StatusOK || exactHit != "yes" {
		t.Fatalf("exact route: status=%d hit=%q", rec.Code, exactHit)
	}

	rec = app.send(t, `{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel:reset","component_type":2}}`)
	if rec.Code != http.StatusOK || patternAction != "reset" {
		t.Fatalf("pattern route: status=%d action=%q", rec.Code, patternAction)
	}
}

// TestUnmatchedInteractionsDropByDefault checks an unrouted interaction is dropped
// without acknowledgement outside debug mode.
func TestUnmatchedInteractionsDropByDefault(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	for _, body := range []string{
		`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"missing","type":1}}`,
		`{"id":"2","application_id":"2","type":3,"token":"t","data":{"custom_id":"nope","component_type":2}}`,
		`{"id":"3","application_id":"2","type":5,"token":"t","data":{"custom_id":"nope"}}`,
	} {
		if rec := app.send(t, body); rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d for %s, want 204", rec.Code, body)
		}
	}
}

// TestUnmatchedInteractionsExplainInDebug checks the debug ephemeral hint.
func TestUnmatchedInteractionsExplainInDebug(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0", Debug: true, Logf: func(string, ...any) {}})

	rec := app.send(t, `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"missing","type":1}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("command status = %d, want 200", rec.Code)
	}

	if body := rec.Body.String(); !strings.Contains(body, "no matching route") || !strings.Contains(body, "client.Command") {
		t.Fatalf("command body = %s", body)
	}

	rec = app.send(t, `{"id":"2","application_id":"2","type":3,"token":"t","data":{"custom_id":"panel_x","component_type":2}}`)
	if body := rec.Body.String(); rec.Code != http.StatusOK || !strings.Contains(body, "client.Component") {
		t.Fatalf("component: status=%d body=%s", rec.Code, body)
	}
}

// TestDeferAnswersBeforeHandlerReturns proves the acknowledgement is sent before
// the handler returns.
func TestDeferAnswersBeforeHandlerReturns(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	entered := make(chan struct{})
	release := make(chan struct{})

	if err := app.client.Command("slow", func(_ context.Context, itx *discord.CommandInteraction) error {
		if err := itx.Defer(true); err != nil {
			t.Errorf("Defer: %v", err)
		}
		close(entered)
		<-release

		// The defer was the first reply, so returning without another is fine.
		return nil
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	w := newSignalWriter()
	done := make(chan struct{})
	go func() {
		app.client.ServeHTTP(w, app.request(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"slow","type":1}}`))
		close(done)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never ran")
	}

	select {
	case <-w.wrote:
	case <-time.After(2 * time.Second):
		t.Fatal("the acknowledgement waited for the handler to return")
	}

	close(release)
	<-done

	if body := w.body.String(); !strings.Contains(body, `"type":5`) {
		t.Fatalf("body = %s, want a deferred acknowledgement", body)
	}
}

func TestRespondSendsImmediatelyAndRefusesASecondReply(t *testing.T) {
	app := newTestApp(t, Config{HTTPAddr: "127.0.0.1:0"})

	second := make(chan error, 1)
	if err := app.client.Component("plant", func(_ context.Context, itx *discord.ComponentInteraction, _ *route.Params) error {
		if err := itx.Respond(discord.Response{
			Type: discord.ResponseUpdateMessage,
			Data: discord.ResponseMessageData{Content: "planted"},
		}); err != nil {
			t.Errorf("Respond: %v", err)
		}
		second <- itx.Defer(true)
		return nil
	}); err != nil {
		t.Fatalf("Component: %v", err)
	}

	rec := app.send(t, `{"id":"1","application_id":"2","type":3,"token":"t","data":{"custom_id":"plant","component_type":2}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "planted") {
		t.Fatalf("body = %s, want the early response", rec.Body.String())
	}

	if err := <-second; !errors.Is(err, ErrResponded) {
		t.Fatalf("second reply = %v, want ErrResponded", err)
	}
}

func TestParseBytes(t *testing.T) {
	cases := map[string]int64{
		"512MiB": 512 << 20,
		"1GiB":   1 << 30,
		"256MB":  256_000_000,
		"off":    0,
		"":       0,
	}

	for in, want := range cases {
		got, ok := parseBytes(in)
		if in == "off" || in == "" {
			if ok {
				t.Errorf("parseBytes(%q) unexpectedly ok", in)
			}
			continue
		}

		if !ok || got != want {
			t.Errorf("parseBytes(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
}
