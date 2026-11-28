package rest_test

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/amatsagu/tempest/rest"
)

func BenchmarkRouteKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = rest.RouteKey(http.MethodPost, "/api/v10/channels/123456789012345678/messages")
	}
}

func BenchmarkRateLimiter_HotRoute(b *testing.B) {
	rl := rest.NewRateLimiter(rest.RateLimiterConfig{})
	hdr := http.Header{}
	hdr.Set("X-RateLimit-Bucket", "abc")
	hdr.Set("X-RateLimit-Limit", "10000000")
	hdr.Set("X-RateLimit-Remaining", "9999999")
	hdr.Set("X-RateLimit-Reset-After", "60")

	cb, err := rl.Wait(context.Background(), http.MethodPost, "/channels/123456/messages")
	if err != nil {
		b.Fatal(err)
	}
	cb(hdr)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			done, err := rl.Wait(ctx, http.MethodPost, "/channels/123456/messages")
			if err != nil {
				b.Error(err)
				return
			}
			done(hdr)
		}
	})
}

func BenchmarkRateLimiter_1500kGuilds(b *testing.B) {
	rl := rest.NewRateLimiter(rest.RateLimiterConfig{})
	hdr := http.Header{}
	hdr.Set("X-RateLimit-Bucket", "guild-bucket")
	hdr.Set("X-RateLimit-Limit", "50")
	hdr.Set("X-RateLimit-Remaining", "49")
	hdr.Set("X-RateLimit-Reset-After", "5")

	// Precompute a rotating working set of route strings representative of 1.5M guild IDs.
	const sampleSize = 16384
	paths := make([]string, sampleSize)
	for i := range sampleSize {
		guildID := 100000000000000000 + uint64(i)*91548874999
		paths[i] = "/guilds/" + strconv.FormatUint(guildID%1_500_000+1_000_000_000, 10) + "/members"
	}

	var idx atomic.Uint64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			n := idx.Add(1)
			path := paths[n&(sampleSize-1)]
			done, err := rl.Wait(ctx, http.MethodGet, path)
			if err != nil {
				b.Error(err)
				return
			}
			done(hdr)
		}
	})
}

type benchPayload struct {
	Content string `json:"content"`
	Flags   uint64 `json:"flags,omitzero"`
}

func BenchmarkClient_RequestJSON(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", rest.ContentTypeJSON)
		w.Header().Set("X-RateLimit-Bucket", "msg-bucket")
		w.Header().Set("X-RateLimit-Limit", "1000000")
		w.Header().Set("X-RateLimit-Remaining", "999999")
		w.Header().Set("X-RateLimit-Reset-After", "60")
		_, _ = w.Write([]byte(`{"id":"1234567890","content":"ok"}`))
	}))
	defer srv.Close()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:   testToken("123456"),
		BaseURL: srv.URL + "/api/v10",
	})
	payload := benchPayload{Content: "hello discord", Flags: 64}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		var out struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		}
		for pb.Next() {
			if err := client.Request(ctx, http.MethodPost, "/channels/999/messages", payload, &out); err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkClient_RequestWithFiles(b *testing.B) {
	fileBytes := bytes.Repeat([]byte("A"), 16*1024)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", rest.ContentTypeJSON)
		w.Header().Set("X-RateLimit-Bucket", "upload-bucket")
		w.Header().Set("X-RateLimit-Limit", "1000000")
		w.Header().Set("X-RateLimit-Remaining", "999999")
		w.Header().Set("X-RateLimit-Reset-After", "60")
		_, _ = w.Write([]byte(`{"id":"55"}`))
	}))
	defer srv.Close()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:   testToken("123456"),
		BaseURL: srv.URL + "/api/v10",
	})
	payload := benchPayload{Content: "attachment test"}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		r := bytes.NewReader(nil)
		files := []rest.File{{Name: "image.png", Reader: r, ContentType: "image/png"}}
		var out map[string]string

		for pb.Next() {
			r.Reset(fileBytes)
			if err := client.RequestWithFiles(ctx, http.MethodPost, "/channels/999/messages", payload, files, &out); err != nil {
				b.Error(err)
				return
			}
		}
	})
}

// Measures unthrottled mixed workload (95% JSON, 5% 16KB multipart files across 1.5M server IDs).
func BenchmarkClient_Mixed1500kServers(b *testing.B) {
	fileBytes := bytes.Repeat([]byte("X"), 16*1024)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", rest.ContentTypeJSON)
		w.Header().Set("X-RateLimit-Bucket", "mix-bucket")
		w.Header().Set("X-RateLimit-Limit", "1000000")
		w.Header().Set("X-RateLimit-Remaining", "999999")
		w.Header().Set("X-RateLimit-Reset-After", "60")
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:   testToken("123456"),
		BaseURL: srv.URL + "/api/v10",
	})

	const routeCount = 16384
	routes := make([]string, routeCount)
	for i := range routeCount {
		serverID := uint64(100_000_000_000_000_000) + uint64(i)*97
		routes[i] = "/channels/" + strconv.FormatUint(serverID+(uint64(i)%1_500_000), 10) + "/messages"
	}

	payloadJSON, _ := json.Marshal(benchPayload{Content: "status ok", Flags: 64})
	_ = payloadJSON
	payload := benchPayload{Content: "status ok", Flags: 64}

	var seq atomic.Uint64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		r := bytes.NewReader(nil)
		files := []rest.File{{Name: "att.png", Reader: r, ContentType: "image/png"}}
		var res struct {
			ID string `json:"id"`
		}

		for pb.Next() {
			n := seq.Add(1)
			route := routes[n&(routeCount-1)]
			if n%20 == 0 {
				r.Reset(fileBytes)
				if err := client.RequestWithFiles(ctx, http.MethodPost, route, payload, files, &res); err != nil {
					b.Error(err)
					return
				}
			} else {
				if err := client.Request(ctx, http.MethodPost, route, payload, &res); err != nil {
					b.Error(err)
					return
				}
			}
		}
	})
}

type benchMediaItem struct {
	Media struct {
		URL string `json:"url"`
	} `json:"media"`
	Description string `json:"description,omitzero"`
	Spoiler     bool   `json:"spoiler,omitzero"`
}

type benchSelectOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitzero"`
	Default     bool   `json:"default,omitzero"`
}

type benchComponent struct {
	Accessory *benchComponent `json:"accessory,omitzero"`
	Component *benchComponent `json:"component,omitzero"`
	Required  *bool           `json:"required,omitzero"`
	Divider   *bool           `json:"divider,omitzero"`
	Media     *struct {
		URL string `json:"url"`
	} `json:"media,omitzero"`
	CustomID    string              `json:"custom_id,omitzero"`
	Label       string              `json:"label,omitzero"`
	Content     string              `json:"content,omitzero"`
	Placeholder string              `json:"placeholder,omitzero"`
	URL         string              `json:"url,omitzero"`
	Components  []benchComponent    `json:"components,omitzero"`
	Items       []benchMediaItem    `json:"items,omitzero"`
	Options     []benchSelectOption `json:"options,omitzero"`
	MaxValues   int                 `json:"max_values,omitzero"`
	MinLength   int                 `json:"min_length,omitzero"`
	MaxLength   int                 `json:"max_length,omitzero"`
	AccentColor int                 `json:"accent_color,omitzero"`
	Type        uint8               `json:"type"`
	Style       uint8               `json:"style,omitzero"`
	Spacing     uint8               `json:"spacing,omitzero"`
	Disabled    bool                `json:"disabled,omitzero"`
}

type benchCV2Message struct {
	Components []benchComponent `json:"components"`
	Flags      uint64           `json:"flags"`
}

type benchModalResponse struct {
	Data struct {
		CustomID   string           `json:"custom_id"`
		Title      string           `json:"title"`
		Components []benchComponent `json:"components"`
	} `json:"data"`
	Type uint8 `json:"type"`
}

func makeBigCV2Payload() benchCV2Message {
	options := make([]benchSelectOption, 12)
	for i := range options {
		n := strconv.Itoa(i + 1)
		options[i] = benchSelectOption{
			Label:       "Cluster Region " + n,
			Value:       "region_" + n,
			Description: "Route guild interactions through edge shard #" + n,
			Default:     i == 0,
		}
	}

	return benchCV2Message{
		Flags: 1 << 15,
		Components: []benchComponent{{
			Type:        17, // Container
			AccentColor: 0x5865F2,
			Components: []benchComponent{
				{
					Type: 9, // Section
					Components: []benchComponent{
						{Type: 10, Content: "# Cluster Control Panel\nRealtime telemetry across sharded REST pools and bucket state."},
						{Type: 10, Content: "Active Workers: **64 shards** | Max Buckets: **65,536** | Upload Cap: **10 MiB**"},
					},
					Accessory: &benchComponent{
						Type: 11, // Thumbnail
						Media: &struct {
							URL string `json:"url"`
						}{URL: "https://cdn.discordapp.com/embed/avatars/0.png"},
					},
				},
				{Type: 14, Divider: new(true), Spacing: 2},
				{
					Type: 12, // MediaGallery
					Items: []benchMediaItem{
						{Media: struct {
							URL string `json:"url"`
						}{URL: "https://example.com/a.png"}, Description: "Latency histogram"},
						{Media: struct {
							URL string `json:"url"`
						}{URL: "https://example.com/b.png"}, Description: "Heap profile"},
						{Media: struct {
							URL string `json:"url"`
						}{URL: "https://example.com/c.png"}, Description: "Shard distribution"},
						{Media: struct {
							URL string `json:"url"`
						}{URL: "https://example.com/d.png"}, Description: "Error budget", Spoiler: true},
					},
				},
				{Type: 14, Divider: new(false), Spacing: 1},
				{
					Type: 1, // ActionRow
					Components: []benchComponent{
						{Type: 2, Style: 1, Label: "Refresh", CustomID: "panel:refresh"},
						{Type: 2, Style: 3, Label: "Scale Up", CustomID: "panel:scale"},
						{Type: 2, Style: 4, Label: "Drain", CustomID: "panel:drain"},
						{Type: 2, Style: 5, Label: "Docs", URL: "https://discord.com/developers/docs"},
					},
				},
				{
					Type: 1, // ActionRow with large StringSelect
					Components: []benchComponent{{
						Type:        3,
						CustomID:    "panel:region",
						Placeholder: "Select preferred gateway + REST region",
						MaxValues:   3,
						Options:     options,
					}},
				},
			},
		}},
	}
}

func makeBigModalPayload() benchModalResponse {
	options := make([]benchSelectOption, 10)
	for i := range options {
		n := strconv.Itoa(i + 1)
		options[i] = benchSelectOption{
			Label:       "Permission Profile " + n,
			Value:       "profile_" + n,
			Description: "Apply curated moderation ruleset tier " + n,
		}
	}

	var modal benchModalResponse
	modal.Type = 9 // Modal response
	modal.Data.CustomID = "modal:guild_onboarding_v2"
	modal.Data.Title = "Server Configuration & Onboarding"
	modal.Data.Components = []benchComponent{
		{
			Type:  18, // Label
			Label: "Primary Audit Webhook Name",
			Component: &benchComponent{
				Type:        4,
				CustomID:    "field:webhook_name",
				Style:       1,
				Placeholder: "tempest-audit-relay",
				MinLength:   3,
				MaxLength:   64,
				Required:    new(true),
			},
		},
		{
			Type:  18,
			Label: "Escalation Playbook Notes",
			Component: &benchComponent{
				Type:        4,
				CustomID:    "field:playbook",
				Style:       2,
				Placeholder: "Describe moderator escalation steps and channel routing rules...",
				MinLength:   10,
				MaxLength:   1000,
				Required:    new(true),
			},
		},
		{
			Type:  18,
			Label: "Preset Security Policies",
			Component: &benchComponent{
				Type:        3,
				CustomID:    "field:policies",
				Placeholder: "Pick up to 4 policy presets",
				MaxValues:   4,
				Required:    new(false),
				Options:     options,
			},
		},
		{
			Type:  18,
			Label: "Reviewer Roles",
			Component: &benchComponent{
				Type:        6, // RoleSelect
				CustomID:    "field:roles",
				Placeholder: "Choose moderator roles",
				MaxValues:   5,
				Required:    new(true),
			},
		},
	}
	return modal
}

type benchRoundTripper struct {
	hdr  http.Header
	body []byte
}

func (rt *benchRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}

	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Header:        rt.hdr,
		ContentLength: int64(len(rt.body)),
		Body:          io.NopCloser(bytes.NewReader(rt.body)),
	}, nil
}

func mustRawJSON(b *testing.B, v any) jsontext.Value {
	b.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		b.Fatalf("marshal benchmark fixture: %v", err)
	}

	return jsontext.Value(raw)
}

func runJSONBench(b *testing.B, inMemory, useDo bool, payload any) {
	hdr := http.Header{
		"Content-Type":            {rest.ContentTypeJSON},
		"X-RateLimit-Bucket":      {"bench-bucket"},
		"X-RateLimit-Limit":       {"10000000"},
		"X-RateLimit-Remaining":   {"9999999"},
		"X-RateLimit-Reset-After": {"60"},
	}
	respBytes := []byte(`{"id":"1234567890","content":"ok"}`)

	var client *rest.RestClient
	if inMemory {
		client = rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("123456"),
			HTTPClient: &http.Client{Transport: &benchRoundTripper{hdr: hdr, body: respBytes}},
		})
	} else {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			resHdr := w.Header()
			maps.Copy(resHdr, hdr)
			_, _ = w.Write(respBytes)
		}))
		defer srv.Close()

		client = rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("123456"),
			BaseURL: srv.URL + "/api/v10",
		})
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		var out struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		}

		if useDo {
			for pb.Next() {
				if err := client.Do(ctx, rest.CustomRequest{
					Method: http.MethodPost,
					Route:  "/channels/999/messages",
					Body:   payload,
					Into:   &out,
				}); err != nil {
					b.Error(err)
					return
				}
			}
			return
		}

		for pb.Next() {
			if err := client.Request(ctx, http.MethodPost, "/channels/999/messages", payload, &out); err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkPipeline_SmallJSON_Request(b *testing.B) {
	runJSONBench(b, true, false, benchPayload{Content: "hello discord", Flags: 64})
}

func BenchmarkPipeline_SmallJSON_Do(b *testing.B) {
	runJSONBench(b, true, true, benchPayload{Content: "hello discord", Flags: 64})
}

func BenchmarkPipeline_SmallJSON_PreEncoded_Request(b *testing.B) {
	runJSONBench(b, true, false, mustRawJSON(b, benchPayload{Content: "hello discord", Flags: 64}))
}

func BenchmarkPipeline_SmallJSON_PreEncoded_Do(b *testing.B) {
	runJSONBench(b, true, true, mustRawJSON(b, benchPayload{Content: "hello discord", Flags: 64}))
}

func BenchmarkPipeline_BigCV2_Request(b *testing.B) {
	runJSONBench(b, true, false, makeBigCV2Payload())
}

func BenchmarkPipeline_BigCV2_Do(b *testing.B) {
	runJSONBench(b, true, true, makeBigCV2Payload())
}

func BenchmarkPipeline_BigCV2_PreEncoded_Request(b *testing.B) {
	runJSONBench(b, true, false, mustRawJSON(b, makeBigCV2Payload()))
}

func BenchmarkPipeline_BigCV2_PreEncoded_Do(b *testing.B) {
	runJSONBench(b, true, true, mustRawJSON(b, makeBigCV2Payload()))
}

func BenchmarkPipeline_BigModal_Request(b *testing.B) {
	runJSONBench(b, true, false, makeBigModalPayload())
}

func BenchmarkPipeline_BigModal_Do(b *testing.B) {
	runJSONBench(b, true, true, makeBigModalPayload())
}

func BenchmarkPipeline_BigModal_PreEncoded_Request(b *testing.B) {
	runJSONBench(b, true, false, mustRawJSON(b, makeBigModalPayload()))
}

func BenchmarkPipeline_BigModal_PreEncoded_Do(b *testing.B) {
	runJSONBench(b, true, true, mustRawJSON(b, makeBigModalPayload()))
}

func BenchmarkClient_SmallJSON_Request(b *testing.B) {
	runJSONBench(b, false, false, benchPayload{Content: "hello discord", Flags: 64})
}

func BenchmarkClient_SmallJSON_Do(b *testing.B) {
	runJSONBench(b, false, true, benchPayload{Content: "hello discord", Flags: 64})
}

func BenchmarkClient_SmallJSON_PreEncoded_Request(b *testing.B) {
	runJSONBench(b, false, false, mustRawJSON(b, benchPayload{Content: "hello discord", Flags: 64}))
}

func BenchmarkClient_SmallJSON_PreEncoded_Do(b *testing.B) {
	runJSONBench(b, false, true, mustRawJSON(b, benchPayload{Content: "hello discord", Flags: 64}))
}

func BenchmarkClient_BigCV2_Request(b *testing.B) {
	runJSONBench(b, false, false, makeBigCV2Payload())
}

func BenchmarkClient_BigCV2_Do(b *testing.B) {
	runJSONBench(b, false, true, makeBigCV2Payload())
}

func BenchmarkClient_BigCV2_PreEncoded_Request(b *testing.B) {
	runJSONBench(b, false, false, mustRawJSON(b, makeBigCV2Payload()))
}

func BenchmarkClient_BigCV2_PreEncoded_Do(b *testing.B) {
	runJSONBench(b, false, true, mustRawJSON(b, makeBigCV2Payload()))
}

func BenchmarkClient_BigModal_Request(b *testing.B) {
	runJSONBench(b, false, false, makeBigModalPayload())
}

func BenchmarkClient_BigModal_Do(b *testing.B) {
	runJSONBench(b, false, true, makeBigModalPayload())
}

func BenchmarkClient_BigModal_PreEncoded_Request(b *testing.B) {
	runJSONBench(b, false, false, mustRawJSON(b, makeBigModalPayload()))
}

func BenchmarkClient_BigModal_PreEncoded_Do(b *testing.B) {
	runJSONBench(b, false, true, mustRawJSON(b, makeBigModalPayload()))
}
