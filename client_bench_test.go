package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/rest"
)

type noopRoundTripper struct{}

func (noopRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     make(http.Header),
		Body:       http.NoBody,
		Request:    req,
	}, nil
}

type noopResponseWriter struct {
	hdr http.Header
}

func (w *noopResponseWriter) Header() http.Header         { return w.hdr }
func (w *noopResponseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *noopResponseWriter) WriteHeader(int)             {}

type resettableBody struct {
	r *bytes.Reader
}

func (rb *resettableBody) Read(p []byte) (int, error) { return rb.r.Read(p) }
func (rb *resettableBody) Close() error               { return nil }

func newSignedRequestFactory(priv ed25519.PrivateKey, body []byte) func() *http.Request {
	const ts = "1700000000"
	sigHex := hex.EncodeToString(ed25519.Sign(priv, append([]byte(ts), body...)))

	hdr := make(http.Header, 2)
	hdr.Set("X-Signature-Ed25519", sigHex)
	hdr.Set("X-Signature-Timestamp", ts)

	br := bytes.NewReader(nil)
	rb := &resettableBody{r: br}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", rb)
	req.Header = hdr

	return func() *http.Request {
		br.Reset(body)
		return req
	}
}

func newBenchPatternRouter() *patternRouter[int] {
	r := newPatternRouter[int]()

	for i := range 24 {
		_ = r.register("panel:exact:"+strconv.Itoa(i), i+1)
	}

	patterns := []string{
		"ticket:{guild_id}:close",
		"ticket:{guild_id}:claim",
		"poll/{owner_id}/close",
		"calc:{session}:{key}",
		"role:{guild_id}:{role_id}:{action}",
		"ban/{user_id}",
		"act/{user_id}/{verb}",
		"act/{user_id}/mute/{duration}",
		"act/{guild_id}/{user_id}/mute/{duration}",
		"shop/{category}/{item_id}/buy/{qty}",
		"cfg:{guild_id}:{module}:{sub}:{field}",
		"act:note:{trail...}",
		"fallback:{rest...}",
	}
	for i, pat := range patterns {
		_ = r.register(pat, 100+i)
	}

	return r
}

func BenchmarkPatternRouter_Exact(b *testing.B) {
	r := newBenchPatternRouter()

	b.ReportAllocs()
	for b.Loop() {
		id, _, _, _, ok := r.match("panel:exact:19")
		if !ok || id == 0 {
			b.Fatal("expected exact match")
		}
	}
}

func BenchmarkPatternRouter_SingleWildcard(b *testing.B) {
	r := newBenchPatternRouter()

	b.ReportAllocs()
	for b.Loop() {
		id, params, _, n, ok := r.match("ban/123456789012345678")
		if !ok || id == 0 || n != 1 || params[0].value == "" {
			b.Fatal("expected single wildcard match")
		}
	}
}

func BenchmarkPatternRouter_SpecificOverGeneric(b *testing.B) {
	r := newBenchPatternRouter()

	b.ReportAllocs()
	for b.Loop() {
		id, params, _, n, ok := r.match("act/123456789012345678/mute/30m")
		if !ok || id == 0 || n != 2 || params[1].value != "30m" {
			b.Fatal("expected specific 2-wildcard match")
		}
	}
}

func BenchmarkPatternRouter_FourWildcards(b *testing.B) {
	r := newBenchPatternRouter()

	b.ReportAllocs()
	for b.Loop() {
		id, params, _, n, ok := r.match("cfg:987654321012345678:automod:spam:threshold")
		if !ok || id == 0 || n != 4 || params[3].value != "threshold" {
			b.Fatal("expected 4-wildcard match")
		}
	}
}

func BenchmarkPatternRouter_CatchAll(b *testing.B) {
	r := newBenchPatternRouter()

	b.ReportAllocs()
	for b.Loop() {
		id, params, _, n, ok := r.match("act:note:guild:987654321012345678:case:42:appeal")
		if !ok || id == 0 || n != 1 || params[0].value == "" {
			b.Fatal("expected catch-all match")
		}
	}
}

func BenchmarkPatternRouter_Miss(b *testing.B) {
	r := newBenchPatternRouter()

	b.ReportAllocs()
	for b.Loop() {
		_, _, _, _, ok := r.match("unmatched:custom:id:12345")
		if ok {
			b.Fatal("unexpected match")
		}
	}
}

var (
	bigCommandJSON = []byte(`{
		"id": "998877665544332211",
		"application_id": "123456789012345678",
		"guild_id": "223456789012345678",
		"channel_id": "323456789012345678",
		"type": 2,
		"token": "bench-cmd-token-payload",
		"app_permissions": "2147483648",
		"locale": "en-US",
		"guild_locale": "en-US",
		"context": 0,
		"authorizing_integration_owners": {"0": "223456789012345678"},
		"member": {
			"nick": "Moderator",
			"joined_at": "2024-01-01T00:00:00Z",
			"roles": ["101", "102", "103", "104"],
			"permissions": "8",
			"flags": 0,
			"user": {"id": "777", "username": "mod", "global_name": "Mod"}
		},
		"data": {
			"id": "888",
			"name": "admin",
			"type": 1,
			"options": [
				{
					"name": "moderation",
					"type": 2,
					"options": [
						{
							"name": "sanction",
							"type": 1,
							"options": [
								{"name": "target", "type": 6, "value": "999"},
								{"name": "channel", "type": 7, "value": "323456789012345678"},
								{"name": "role", "type": 8, "value": "101"},
								{"name": "duration_min", "type": 4, "value": 60},
								{"name": "reason", "type": 3, "value": "Automated heavy flood detection rule triggered"}
							]
						}
					]
				}
			],
			"resolved": {
				"users": {
					"999": {"id": "999", "username": "violator", "global_name": "Violator"}
				},
				"members": {
					"999": {"joined_at": "2024-06-01T12:00:00Z", "roles": ["105"], "permissions": "0", "flags": 0}
				},
				"roles": {
					"101": {"id": "101", "name": "Muted", "color": 16711680, "permissions": "0"}
				},
				"channels": {
					"323456789012345678": {"id": "323456789012345678", "name": "general", "type": 0}
				}
			}
		}
	}`)

	bigComponentJSON = []byte(`{
		"id": "998877665544332222",
		"application_id": "123456789012345678",
		"guild_id": "223456789012345678",
		"channel_id": "323456789012345678",
		"type": 3,
		"token": "bench-cmp-token-payload",
		"app_permissions": "2147483648",
		"locale": "en-US",
		"guild_locale": "en-US",
		"member": {
			"joined_at": "2024-01-01T00:00:00Z",
			"roles": ["101", "102"],
			"permissions": "8",
			"flags": 0,
			"user": {"id": "777", "username": "mod", "global_name": "Mod"}
		},
		"data": {
			"custom_id": "act/223456789012345678/999/mute/30m",
			"component_type": 2,
			"values": ["tier-1", "notify-user"],
			"resolved": {
				"users": {"999": {"id": "999", "username": "violator"}},
				"roles": {"101": {"id": "101", "name": "Muted", "color": 16711680, "permissions": "0"}}
			}
		}
	}`)

	bigModalJSON = []byte(`{
		"id": "998877665544332233",
		"application_id": "123456789012345678",
		"guild_id": "223456789012345678",
		"channel_id": "323456789012345678",
		"type": 5,
		"token": "bench-mdl-token-payload",
		"app_permissions": "2147483648",
		"member": {
			"joined_at": "2024-01-01T00:00:00Z",
			"roles": ["101"],
			"permissions": "8",
			"flags": 0,
			"user": {"id": "777", "username": "mod", "global_name": "Mod"}
		},
		"data": {
			"custom_id": "act:modal:223456789012345678:999:sanction",
			"components": [
				{
					"type": 18,
					"id": 1,
					"label": "Summary",
					"component": {"type": 4, "id": 2, "style": 1, "custom_id": "summary", "value": "Flooding links across three channels"}
				},
				{
					"type": 18,
					"id": 3,
					"label": "Reasons",
					"component": {"type": 3, "id": 4, "custom_id": "reasons", "values": ["spam", "phishing", "botnet"]}
				},
				{
					"type": 18,
					"id": 5,
					"label": "Severity",
					"component": {"type": 21, "id": 6, "custom_id": "severity", "value": "high"}
				},
				{
					"type": 18,
					"id": 7,
					"label": "Flags",
					"component": {"type": 22, "id": 8, "custom_id": "flags", "values": ["purge", "dm_notice"]}
				},
				{
					"type": 18,
					"id": 9,
					"label": "Notify Staff",
					"component": {"type": 23, "id": 10, "custom_id": "notify", "value": true}
				},
				{
					"type": 18,
					"id": 11,
					"label": "Evidence",
					"component": {"type": 19, "id": 12, "custom_id": "evidence", "values": ["555666777888999"]}
				}
			],
			"resolved": {
				"attachments": {
					"555666777888999": {
						"id": "555666777888999",
						"filename": "proof.png",
						"size": 245760,
						"url": "https://cdn.discordapp.com/attachments/1/2/proof.png",
						"proxy_url": "https://media.discordapp.net/attachments/1/2/proof.png",
						"content_type": "image/png",
						"width": 1920,
						"height": 1080
					}
				}
			}
		}
	}`)
)

func newBenchClient(b *testing.B, hooks ClientHooks) (*Client, ed25519.PrivateKey) {
	b.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		b.Fatal(err)
	}

	token := testBotToken("123456789012345678")
	client := NewClient(ClientConfig{
		Token:                 token,
		PublicKey:             hex.EncodeToString(pub),
		MaxConcurrentHandlers: -1,
		Rest: rest.NewRestClient(rest.RestClientConfig{
			Token:      token,
			BaseURL:    "https://discord.invalid/api/v10",
			HTTPClient: &http.Client{Transport: noopRoundTripper{}},
		}),
		Hooks: hooks,
	})
	b.Cleanup(func() {
		_ = client.Shutdown(context.Background())
	})

	_ = client.RegisterCommand(Command{Command: discord.Command{Name: "admin", Description: "Admin"}})
	_ = client.RegisterSubCommand(Command{
		Command: discord.Command{Name: "sanction", Description: "Sanction"},
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			return itx.SendLinearReply(ctx, "ok", true)
		},
	}, "admin", "moderation")

	// Seed router with 20 competing routes to benchmark realistic dispatch.
	for i := range 15 {
		_ = client.RegisterComponent("extra:exact:"+strconv.Itoa(i), func(ctx context.Context, itx *ComponentInteraction) error {
			return itx.Acknowledge(ctx)
		})
	}
	_ = client.RegisterComponent("act/{guild_id}/{user_id}/{verb}", func(ctx context.Context, itx *ComponentInteraction) error {
		return itx.Acknowledge(ctx)
	})
	_ = client.RegisterComponent("act/{guild_id}/{user_id}/mute/{duration}", func(ctx context.Context, itx *ComponentInteraction) error {
		return itx.AcknowledgeWithLinearUpdate(ctx, "muted:"+itx.PathValue("user_id")+":"+itx.PathValue("duration"))
	})
	_ = client.RegisterModal("act:modal:{guild_id}:{user_id}:{kind}", func(ctx context.Context, itx *ModalInteraction) error {
		return itx.AcknowledgeWithLinearMessage(ctx, itx.PathValue("kind")+":"+itx.GetInputValue("summary"), true)
	})

	return client, priv
}

func BenchmarkPipeline_BigCommand_HTTP(b *testing.B) {
	var hookHits atomic.Uint64
	client, priv := newBenchClient(b, ClientHooks{
		PreCommand: func(_ context.Context, _ Command, _ *CommandInteraction) bool {
			hookHits.Add(1)
			return true
		},
		PostCommand: func(_ context.Context, _ Command, _ *CommandInteraction) {
			hookHits.Add(1)
		},
	})

	newReq := newSignedRequestFactory(priv, bigCommandJSON)
	w := &noopResponseWriter{hdr: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		client.ServeHTTP(w, newReq())
	}
	if hookHits.Load() == 0 {
		b.Fatal("expected command hooks to execute")
	}
}

func BenchmarkPipeline_BigCommand_Gateway(b *testing.B) {
	var hookHits atomic.Uint64
	client, _ := newBenchClient(b, ClientHooks{
		PreCommand: func(_ context.Context, _ Command, _ *CommandInteraction) bool {
			hookHits.Add(1)
			return true
		},
		PostCommand: func(_ context.Context, _ Command, _ *CommandInteraction) {
			hookHits.Add(1)
		},
	})

	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if err := client.HandleInteraction(ctx, bigCommandJSON); err != nil {
			b.Fatal(err)
		}
	}
	if hookHits.Load() == 0 {
		b.Fatal("expected command hooks to execute")
	}
}

func BenchmarkPipeline_BigComponent_HTTP(b *testing.B) {
	var hookHits atomic.Uint64
	client, priv := newBenchClient(b, ClientHooks{
		PreComponent: func(_ context.Context, _ *ComponentInteraction) bool {
			hookHits.Add(1)
			return true
		},
		PostComponent: func(_ context.Context, _ *ComponentInteraction) {
			hookHits.Add(1)
		},
	})

	newReq := newSignedRequestFactory(priv, bigComponentJSON)
	w := &noopResponseWriter{hdr: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		client.ServeHTTP(w, newReq())
	}
	if hookHits.Load() == 0 {
		b.Fatal("expected component hooks to execute")
	}
}

func BenchmarkPipeline_BigModal_HTTP(b *testing.B) {
	var hookHits atomic.Uint64
	client, priv := newBenchClient(b, ClientHooks{
		PreModal: func(_ context.Context, _ *ModalInteraction) bool {
			hookHits.Add(1)
			return true
		},
		PostModal: func(_ context.Context, _ *ModalInteraction) {
			hookHits.Add(1)
		},
	})

	newReq := newSignedRequestFactory(priv, bigModalJSON)
	w := &noopResponseWriter{hdr: make(http.Header)}

	b.ReportAllocs()
	for b.Loop() {
		client.ServeHTTP(w, newReq())
	}
	if hookHits.Load() == 0 {
		b.Fatal("expected modal hooks to execute")
	}
}

func BenchmarkPipeline_BigModal_Gateway(b *testing.B) {
	var hookHits atomic.Uint64
	client, _ := newBenchClient(b, ClientHooks{
		PreModal: func(_ context.Context, _ *ModalInteraction) bool {
			hookHits.Add(1)
			return true
		},
		PostModal: func(_ context.Context, _ *ModalInteraction) {
			hookHits.Add(1)
		},
	})

	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if err := client.HandleInteraction(ctx, bigModalJSON); err != nil {
			b.Fatal(err)
		}
	}
	if hookHits.Load() == 0 {
		b.Fatal("expected modal hooks to execute")
	}
}
