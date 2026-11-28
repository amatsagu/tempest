package rest_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amatsagu/tempest/rest"
)

func testToken(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id)) + ".test.sig"
}

func TestExtractUserIDFromToken(t *testing.T) {
	t.Parallel()

	id, err := rest.ExtractUserIDFromToken("Bot " + testToken("123456789012345678"))
	if err != nil || id != 123456789012345678 {
		t.Fatalf("ExtractUserIDFromToken valid = (%d, %v), want 123456789012345678", id, err)
	}

	stdToken := base64.StdEncoding.EncodeToString([]byte("987654321")) + ".x.y"
	id, err = rest.ExtractUserIDFromToken(stdToken)
	if err != nil || id != 987654321 {
		t.Fatalf("ExtractUserIDFromToken std b64 = (%d, %v), want 987654321", id, err)
	}

	for _, bad := range []string{
		"",
		"nodot",
		".empty",
		"!!!invalid!!!.a.b",
		base64.RawURLEncoding.EncodeToString([]byte("notanumber")) + ".a.b",
		base64.RawURLEncoding.EncodeToString([]byte("0")) + ".a.b",
	} {
		if _, err := rest.ExtractUserIDFromToken(bad); err == nil {
			t.Fatalf("expected error for invalid token %q", bad)
		}
	}
}

func TestRouteKeyNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodPost, "/api/v10/channels/111222/messages", "POST /channels/111222/messages"},
		{http.MethodPatch, "/channels/111222/messages/999888?wait=true", "PATCH /channels/111222/messages/{id}"},
		{http.MethodDelete, "/guilds/333/members/444/roles/555", "DELETE /guilds/333/members/{id}/roles/{id}"},
		{http.MethodPatch, "/webhooks/777/secret_tok_abc/messages/@original", "PATCH /webhooks/777/{token}/messages/@original"},
		{http.MethodPost, "/interactions/888/inter_tok_xyz/callback", "POST /interactions/{id}/{token}/callback"},
		{http.MethodPut, "/channels/111/messages/222/reactions/custom:99/@me", "PUT /channels/111/messages/{id}/reactions/{emoji}/@me"},
		{http.MethodGet, "/users/@me/guilds", "GET /users/@me/guilds"},
	}

	for _, tc := range tests {
		if got := rest.RouteKey(tc.method, tc.path); got != tc.want {
			t.Errorf("RouteKey(%s, %s) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestRateLimiterLifecycleAndMemoryCap(t *testing.T) {
	t.Parallel()

	var nowMu sync.Mutex
	curTime := time.Unix(1_700_000_000, 0)
	nowFn := func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return curTime
	}
	advance := func(d time.Duration) {
		nowMu.Lock()
		curTime = curTime.Add(d)
		nowMu.Unlock()
	}

	rl := rest.NewRateLimiter(rest.RateLimiterConfig{
		Now:           nowFn,
		Shards:        8,
		IdleTTL:       time.Minute,
		MaxBuckets:    64,
		MaxRoutes:     64,
		SweepInterval: 32,
	})

	// Seed a bucket with Limit=2, Remaining=0, Reset-After=0.5s via header callback.
	release, err := rl.Wait(t.Context(), http.MethodPost, "/channels/100/messages")
	if err != nil {
		t.Fatalf("first Wait: %v", err)
	}

	hdr := http.Header{}
	hdr.Set("X-RateLimit-Bucket", "hash-msg")
	hdr.Set("X-RateLimit-Limit", "2")
	hdr.Set("X-RateLimit-Remaining", "1")
	hdr.Set("X-RateLimit-Reset-After", "0.5")
	release(hdr)

	// Next request consumes Remaining=1 without blocking.
	r2, err := rl.Wait(t.Context(), http.MethodPost, "/channels/100/messages")
	if err != nil {
		t.Fatalf("second Wait: %v", err)
	}
	r2(nil)

	// Bucket now at Remaining=0; Waiting with canceled context fails before sleep completes.
	cancelCtx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	if _, err := rl.Wait(cancelCtx, http.MethodPost, "/channels/100/messages"); err == nil {
		t.Fatal("expected deadline error waiting on depleted bucket")
	}

	// Advancing fake clock past window reopens limit.
	advance(time.Second)

	r3, err := rl.Wait(t.Context(), http.MethodPost, "/channels/100/messages")
	if err != nil {
		t.Fatalf("Wait after window reset: %v", err)
	}
	r3(hdr)

	// Stress bucket creation across 500 channels exceeding MaxBuckets=64 and verify strict cap.
	for i := range 500 {
		path := "/channels/" + strconv.Itoa(1000+i) + "/messages"
		cb, err := rl.Wait(t.Context(), http.MethodPost, path)
		if err != nil {
			t.Fatalf("Wait(%s): %v", path, err)
		}

		h := http.Header{}
		h.Set("X-RateLimit-Bucket", "hash-msg")
		h.Set("X-RateLimit-Limit", "5")
		h.Set("X-RateLimit-Remaining", "4")
		h.Set("X-RateLimit-Reset-After", "1.0")
		cb(h)
	}

	buckets, routes := rl.Size()
	if buckets > 64 || routes > 64 {
		t.Fatalf("rate limiter exceeded cap: buckets=%d routes=%d (max 64)", buckets, routes)
	}

	// Sweep idle entries after IdleTTL.
	advance(2 * time.Minute)
	removed := rl.Sweep(0)
	if removed == 0 {
		t.Fatal("expected idle sweep to remove expired buckets")
	}

	bucketsAfter, _ := rl.Size()
	if bucketsAfter != 0 {
		t.Fatalf("expected 0 remaining buckets after idle sweep, got %d", bucketsAfter)
	}
}

func TestRateLimiterGlobalPauseAndProactivePacer(t *testing.T) {
	t.Parallel()

	rl := rest.NewRateLimiter(rest.RateLimiterConfig{
		GlobalRate: 1000,
	})

	cb, err := rl.Wait(t.Context(), http.MethodGet, "/gateway/bot")
	if err != nil {
		t.Fatalf("initial Wait: %v", err)
	}

	hdr := http.Header{}
	hdr.Set("X-RateLimit-Global", "true")
	hdr.Set("Retry-After", "0.05")
	cb(hdr)

	shortCtx, cancel := context.WithTimeout(t.Context(), 15*time.Millisecond)
	defer cancel()

	if _, err := rl.Wait(shortCtx, http.MethodGet, "/channels/1/messages"); err == nil {
		t.Fatal("expected Wait to block on active global pause")
	}

	time.Sleep(350 * time.Millisecond)

	cb2, err := rl.Wait(t.Context(), http.MethodGet, "/channels/1/messages")
	if err != nil {
		t.Fatalf("Wait after global pause expiry: %v", err)
	}
	cb2(nil)
}

func TestClientRequestModesAndRetries(t *testing.T) {
	t.Parallel()

	type messageResp struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}

	defaultUA := "DiscordApp https://github.com/amatsagu/tempest"

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" || r.Header.Get("User-Agent") != defaultUA {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/api/v10/channels/10/messages":
			if r.Header.Get("X-Audit-Log-Reason") == "cleanup%20run" {
				w.Header().Set("Content-Type", rest.ContentTypeJSON)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"1","content":"audit"}`))
				return
			}

			var body map[string]any
			if err := json.UnmarshalRead(r.Body, &body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", rest.ContentTypeJSON)
			w.Header().Set("X-RateLimit-Bucket", "send-msg")
			w.Header().Set("X-RateLimit-Limit", "50")
			w.Header().Set("X-RateLimit-Remaining", "49")
			w.Header().Set("X-RateLimit-Reset-After", "0.5")
			_, _ = w.Write([]byte(`{"id":"42","content":"ok"}`))

		case "/api/v10/retry-get":
			n := attempts.Add(1)
			if n == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"99","content":"recovered"}`))

		case "/api/v10/retry-post":
			attempts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)

		case "/api/v10/rate-limited":
			n := attempts.Add(1)
			if n == 1 {
				w.Header().Set("X-RateLimit-Bucket", "rl-b")
				w.Header().Set("X-RateLimit-Limit", "1")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset-After", "0.02")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"message":"rate limited","retry_after":0.02,"global":false}`))
				return
			}
			w.WriteHeader(http.StatusNoContent)

		case "/api/v10/not-found":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Unknown Channel"}`))
		}
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	client := rest.NewRestClient(rest.RestClientConfig{
		Token:       testToken("123456"),
		BaseURL:     srv.URL + "/api/v10",
		TraceLogger: log.New(&logBuf, "", 0),
		Trace:       true,
		MaxRetries:  3,
	})

	t.Run("json struct and raw bytes and jsontext value", func(t *testing.T) {
		var out messageResp
		if err := client.Request(t.Context(), http.MethodPost, "/channels/10/messages", map[string]string{"content": "hello"}, &out); err != nil {
			t.Fatalf("Request: %v", err)
		}

		if out.ID != "42" || out.Content != "ok" {
			t.Fatalf("unexpected decoded response: %+v", out)
		}

		var raw []byte
		if err := client.Request(t.Context(), http.MethodPost, "/channels/10/messages", jsontext.Value(`{"content":"raw"}`), &raw); err != nil {
			t.Fatalf("Request raw bytes: %v", err)
		}

		if !strings.Contains(string(raw), `"id":"42"`) {
			t.Fatalf("unexpected raw bytes: %s", raw)
		}

		var outAudit messageResp
		if err := client.Do(t.Context(), rest.CustomRequest{
			Method:         http.MethodPost,
			Route:          "/channels/10/messages",
			Body:           map[string]string{"content": "buffered"},
			AuditLogReason: "cleanup run",
			Into:           &outAudit,
		}); err != nil {
			t.Fatalf("Do: %v", err)
		}

		if outAudit.Content != "audit" {
			t.Fatalf("expected audit response, got %+v", outAudit)
		}
	})

	t.Run("idempotent 5xx retries vs non-idempotent no retry", func(t *testing.T) {
		attempts.Store(0)
		var out messageResp
		if err := client.Request(t.Context(), http.MethodGet, "/retry-get", nil, &out); err != nil {
			t.Fatalf("expected GET 503 recovery, got %v", err)
		}

		if attempts.Load() != 2 || out.ID != "99" {
			t.Fatalf("unexpected retry outcome: attempts=%d out=%+v", attempts.Load(), out)
		}

		attempts.Store(0)
		if err := client.Request(t.Context(), http.MethodPost, "/retry-post", map[string]int{"x": 1}, nil); err == nil {
			t.Fatal("expected POST 500 failure without retry")
		}

		if attempts.Load() != 1 {
			t.Fatalf("expected non-idempotent POST to execute once, got %d", attempts.Load())
		}
	})

	t.Run("429 retries cleanly for POST and 404 returns HTTPError", func(t *testing.T) {
		attempts.Store(0)
		if err := client.Request(t.Context(), http.MethodPost, "/rate-limited", map[string]int{"ok": 1}, nil); err != nil {
			t.Fatalf("expected POST 429 retry to succeed, got %v", err)
		}

		err := client.Request(t.Context(), http.MethodGet, "/not-found", nil, nil)
		var httpErr *rest.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 *HTTPError, got %v", err)
		}
	})
}

func TestClientRequestWithFilesAndValidation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		parts := map[string]string{}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			data, _ := io.ReadAll(part)
			parts[part.FormName()+"|"+part.FileName()] = string(data)
		}

		if !strings.Contains(parts["payload_json|"], `"hello"`) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if parts[`files[0]|shot"1.png`] != "PNGDATA" || parts["files[1]|note.txt"] != "hello file" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:   testToken("555"),
		BaseURL: srv.URL + "/api/v10",
	})

	files := []rest.File{
		{Name: `shot"1.png`, Reader: strings.NewReader("PNGDATA"), ContentType: "image/png"},
		{Name: "note.txt", Reader: strings.NewReader("hello file")},
	}

	var resp struct {
		OK bool `json:"ok"`
	}
	if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/channels/9/messages", map[string]string{"content": "hello"}, files, &resp); err != nil {
		t.Fatalf("RequestWithFiles: %v", err)
	}

	if !resp.OK {
		t.Fatal("expected ok=true response")
	}

	// Reject files above MaxFileUploadSize before dialing.
	oversized := []rest.File{{
		Name:   "huge.bin",
		Reader: io.LimitReader(zeroReader{}, 10*1024*1024+1),
	}}
	if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/channels/9/messages", nil, oversized, nil); err == nil {
		t.Fatal("expected error when file exceeds MaxFileUploadSize")
	}
}

func TestCircuitBreakerTripsOnConcurrentRetryThreshold(t *testing.T) {
	t.Parallel()

	blockCh := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v10/slow-err" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		<-blockCh
	}))
	defer func() {
		close(blockCh)
		srv.Close()
	}()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:          testToken("123"),
		BaseURL:        srv.URL + "/api/v10",
		MaxRetries:     3,
		RetryThreshold: 2,
		MaxWaitTime:    2 * time.Second,
	})

	var wg sync.WaitGroup
	var tripped atomic.Int32
	for range 12 {
		wg.Go(func() {
			err := client.Request(t.Context(), http.MethodGet, "/slow-err", nil, nil)
			if errors.Is(err, rest.ErrTooManyRetries) {
				tripped.Add(1)
			}
		})
	}
	wg.Wait()

	if tripped.Load() == 0 {
		t.Fatal("expected ErrTooManyRetries circuit breaker to trip when concurrent retries exceeded threshold")
	}
}

func TestClientAndRateLimiterAdditionalEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("HTTPError Error string and pre-cancelled context", func(t *testing.T) {
		httpErr := &rest.HTTPError{StatusCode: 400, Status: "400 Bad Request", Body: []byte(`{"code":50035}`)}
		if !strings.Contains(httpErr.Error(), "400 Bad Request") || !strings.Contains(httpErr.Error(), "50035") {
			t.Fatalf("unexpected HTTPError.Error(): %q", httpErr.Error())
		}

		client := rest.NewRestClient(rest.RestClientConfig{Token: testToken("123")})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if err := client.Request(ctx, http.MethodGet, "/users/@me", nil, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}

		var raw []byte
		if err := client.Request(ctx, http.MethodGet, "/users/@me", nil, &raw); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled reading raw bytes, got %v", err)
		}

		if err := client.RequestWithFiles(ctx, http.MethodPost, "/channels/1/messages", nil, []rest.File{{Name: "a.txt", Reader: strings.NewReader("hi")}}, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled from RequestWithFiles, got %v", err)
		}

		if err := client.Do(ctx, rest.CustomRequest{Method: http.MethodGet, Route: "/users/@me"}); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled from Do, got %v", err)
		}
	})

	t.Run("Do supports raw bodies custom headers streams and files", func(t *testing.T) {
		var attempt atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)

			switch r.URL.Path {
			case "/api/v10/raw-bytes":
				if r.Header.Get("Content-Type") != rest.ContentTypeOctetStream || r.Header.Get("X-Custom") != "yes" || string(body) != "BIN" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_, _ = w.Write([]byte("PONG"))

			case "/api/v10/string-body":
				if r.Header.Get("Content-Type") != "text/plain" || string(body) != "hello text" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/jsontext-do":
				if r.Header.Get("Content-Type") != rest.ContentTypeJSON || string(body) != `{"raw":true}` {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/reader-retry":
				if string(body) != "stream-payload" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if attempt.Add(1) == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				_, _ = w.Write([]byte("OK-STREAM"))

			case "/api/v10/seeker-body":
				if string(body) != "seekable" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/custom-files":
				if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") || r.Header.Get("X-Audit-Log-Reason") != "audit%20file" || r.Header.Get("X-Trace") != "1" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("456"),
			BaseURL: srv.URL + "/api/v10",
		})

		var outBuf bytes.Buffer
		if err := client.Do(t.Context(), rest.CustomRequest{
			Method:  http.MethodPost,
			Route:   "/raw-bytes",
			Body:    []byte("BIN"),
			Headers: http.Header{"x-custom": {"yes"}},
			Into:    &outBuf,
		}); err != nil {
			t.Fatalf("Do raw []byte: %v", err)
		}

		if outBuf.String() != "PONG" {
			t.Fatalf("expected PONG in io.Writer target, got %q", outBuf.String())
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method:      http.MethodPost,
			Route:       "/string-body",
			Body:        "hello text",
			ContentType: "text/plain",
		}); err != nil {
			t.Fatalf("Do string body: %v", err)
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/jsontext-do",
			Body:   jsontext.Value(`{"raw":true}`),
		}); err != nil {
			t.Fatalf("Do jsontext.Value body: %v", err)
		}

		var streamOut []byte
		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPut,
			Route:  "/reader-retry",
			Body:   io.NopCloser(strings.NewReader("stream-payload")),
			Into:   &streamOut,
		}); err != nil {
			t.Fatalf("Do plain io.Reader with retry: %v", err)
		}

		if string(streamOut) != "OK-STREAM" {
			t.Fatalf("expected OK-STREAM, got %q", streamOut)
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/seeker-body",
			Body:   strings.NewReader("seekable"),
		}); err != nil {
			t.Fatalf("Do io.ReadSeeker: %v", err)
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method:         http.MethodPost,
			Route:          "/custom-files",
			Body:           map[string]string{"content": "hi"},
			AuditLogReason: "audit file",
			Headers:        http.Header{"X-Trace": {"1"}},
			Files:          []rest.File{{Name: "a.txt", Reader: strings.NewReader("data")}},
		}); err != nil {
			t.Fatalf("Do multipart files with AuditLogReason: %v", err)
		}
	})

	t.Run("Custom HTTPClient Shared Limiter Global JSON 429 and Headerless 429", func(t *testing.T) {
		var count atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v10/json-global":
				n := count.Add(1)
				if n == 1 {
					w.Header().Set("Content-Type", rest.ContentTypeJSON)
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"global":true,"retry_after":0.01,"message":"global"}`))
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/headerless-429":
				n := count.Add(1)
				if n == 1 {
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"global":false,"retry_after":0.01}`))
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/interactions/1/tok/callback":
				w.WriteHeader(http.StatusNoContent)
			}
		}))
		defer srv.Close()

		lim := rest.NewRateLimiter(rest.RateLimiterConfig{})
		customHTTP := &http.Client{Timeout: 5 * time.Second}
		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("999"),
			BaseURL:    srv.URL + "/api/v10",
			HTTPClient: customHTTP,
			Limiter:    lim,
		})

		if client.Limiter() != lim {
			t.Fatal("Limiter() did not return injected limiter")
		}

		count.Store(0)
		if err := client.Request(t.Context(), http.MethodPost, "/json-global", nil, nil); err != nil {
			t.Fatalf("expected recovery after JSON global 429, got %v", err)
		}

		// Interaction callback bypasses global pause cleanly.
		if err := client.Request(t.Context(), http.MethodPost, "/interactions/1/tok/callback", map[string]int{"type": 1}, nil); err != nil {
			t.Fatalf("interaction callback failed: %v", err)
		}

		// Wait out brief global pause tail from JSON global 429.
		time.Sleep(300 * time.Millisecond)

		count.Store(0)
		if err := client.Request(t.Context(), http.MethodPost, "/headerless-429", nil, nil); err != nil {
			t.Fatalf("expected recovery after headerless 429, got %v", err)
		}
	})

	t.Run("Bucket hash migration and >=10 multipart files", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		rl := rest.NewRateLimiter(rest.RateLimiterConfig{Shards: 1, SweepInterval: 2, MaxBuckets: 2, MaxRoutes: 2})
		cb1, err := rl.Wait(t.Context(), http.MethodPost, "/channels/10/messages")
		if err != nil {
			t.Fatal(err)
		}
		h1 := http.Header{}
		h1.Set("X-RateLimit-Bucket", "hash-a")
		h1.Set("X-RateLimit-Limit", "5")
		h1.Set("X-RateLimit-Remaining", "4")
		h1.Set("X-RateLimit-Reset-After", "1.0")
		cb1(h1)

		// Migrate route to hash-b and ensure cleanup works cleanly.
		h2 := http.Header{}
		h2.Set("X-RateLimit-Bucket", "hash-b")
		h2.Set("X-RateLimit-Limit", "5")
		h2.Set("X-RateLimit-Remaining", "4")
		h2.Set("X-RateLimit-Reset-After", "1.0")
		cb1(h2)

		if _, err := rl.Wait(t.Context(), http.MethodGet, "/users/@me"); err != nil {
			t.Fatal(err)
		}

		if removed := rl.Sweep(-1); removed != 0 {
			// Fresh buckets shouldn't be evicted yet.
			t.Fatalf("unexpected early sweep removal: %d", removed)
		}

		client := rest.NewRestClient(rest.RestClientConfig{Token: testToken("123"), BaseURL: srv.URL + "/api/v10"})
		var eleven []rest.File
		for i := range 11 {
			eleven = append(eleven, rest.File{
				Name:   "file_" + strconv.Itoa(i) + ".txt",
				Reader: strings.NewReader("x"),
			})
		}
		if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/channels/1/messages", nil, eleven, nil); err != nil {
			t.Fatalf("RequestWithFiles 11 files: %v", err)
		}
	})
}

func TestRestClientConfigAndDefaults(t *testing.T) {
	t.Parallel()

	t.Run("minimal config applies sane defaults", func(t *testing.T) {
		client := rest.NewRestClient(rest.RestClientConfig{
			Token: testToken("100"),
		})

		if client.Limiter() == nil {
			t.Fatal("expected default Limiter to be initialized")
		}

		if buckets, routes := client.Limiter().Size(); buckets != 0 || routes != 0 {
			t.Fatalf("expected empty initial limiter, got buckets=%d routes=%d", buckets, routes)
		}

		didPanic := false
		func() {
			defer func() {
				if recover() != nil {
					didPanic = true
				}
			}()
			rest.NewRestClient(rest.RestClientConfig{})
		}()

		if !didPanic {
			t.Fatal("expected NewRestClient without Token to panic")
		}
	})

	t.Run("custom UserAgent MaxFileUploadSize and disabled limits", func(t *testing.T) {
		var sawUA string
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sawUA = r.Header.Get("User-Agent")
			hits.Add(1)
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		limitedClient := rest.NewRestClient(rest.RestClientConfig{
			Token:                 testToken("101"),
			BaseURL:               srv.URL + "/api/v10",
			UserAgent:             "CustomBot/2.0",
			MaxFileUploadSize:     8,
			ForceAttemptHTTP2:     new(false),
			DialTimeout:           2 * time.Second,
			KeepAlive:             15 * time.Second,
			IdleConnTimeout:       45 * time.Second,
			TLSHandshakeTimeout:   4 * time.Second,
			ExpectContinueTimeout: 500 * time.Millisecond,
			MaxIdleConns:          16,
			MaxIdleConnsPerHost:   8,
			MaxWaitTime:           -1,
		})

		tooLarge := []rest.File{{Name: "big.bin", Reader: strings.NewReader("123456789")}}
		if err := limitedClient.RequestWithFiles(t.Context(), http.MethodPost, "/channels/1/messages", nil, tooLarge, nil); err == nil {
			t.Fatal("expected error when file exceeds custom MaxFileUploadSize=8")
		}

		if hits.Load() != 0 {
			t.Fatalf("expected oversized check before dialing, got %d requests", hits.Load())
		}

		exact := []rest.File{{Name: "ok.bin", Reader: strings.NewReader("12345678")}}
		if err := limitedClient.RequestWithFiles(t.Context(), http.MethodPost, "/channels/1/messages", nil, exact, nil); err != nil {
			t.Fatalf("expected 8-byte upload to succeed: %v", err)
		}

		if sawUA != "CustomBot/2.0" {
			t.Fatalf("expected custom User-Agent, got %q", sawUA)
		}

		// MaxFileUploadSize < 0 disables per-file size capping.
		unlimitedClient := rest.NewRestClient(rest.RestClientConfig{
			Token:             testToken("102"),
			BaseURL:           srv.URL + "/api/v10",
			MaxFileUploadSize: -1,
		})

		overDefaultLimit := int64(10*1024*1024 + 64)
		largeFiles := []rest.File{{
			Name:   "large.bin",
			Reader: io.LimitReader(zeroReader{}, overDefaultLimit),
		}}
		if err := unlimitedClient.RequestWithFiles(t.Context(), http.MethodPost, "/channels/2/messages", nil, largeFiles, nil); err != nil {
			t.Fatalf("expected upload >10MiB to succeed when MaxFileUploadSize=-1: %v", err)
		}
	})

	t.Run("custom ResetFudge and disabled IdleTTL", func(t *testing.T) {
		var nowMu sync.Mutex
		cur := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		nowFn := func() time.Time {
			nowMu.Lock()
			defer nowMu.Unlock()
			return cur
		}
		advance := func(d time.Duration) {
			nowMu.Lock()
			cur = cur.Add(d)
			nowMu.Unlock()
		}

		// ResetFudge=-1 disables padding, IdleTTL=-1 disables automatic/idle eviction.
		rl := rest.NewRateLimiter(rest.RateLimiterConfig{
			Now:        nowFn,
			Shards:     1,
			IdleTTL:    -1,
			ResetFudge: -1,
		})

		done, err := rl.Wait(t.Context(), http.MethodPost, "/channels/77/messages")
		if err != nil {
			t.Fatalf("initial Wait: %v", err)
		}

		hdr := http.Header{}
		hdr.Set("X-RateLimit-Bucket", "fudge-bucket")
		hdr.Set("X-RateLimit-Limit", "1")
		hdr.Set("X-RateLimit-Remaining", "0")
		hdr.Set("X-RateLimit-Reset-After", "0.05")
		done(hdr)

		// Without the default 100ms ResetFudge, advancing 60ms reopens a 50ms window immediately.
		advance(60 * time.Millisecond)

		nextDone, err := rl.Wait(t.Context(), http.MethodPost, "/channels/77/messages")
		if err != nil {
			t.Fatalf("expected immediate acquire after 60ms without ResetFudge, got %v", err)
		}
		nextDone(nil)

		// With IdleTTL < 0, Sweep(0) must not evict idle buckets even far in the future.
		advance(24 * time.Hour)

		if removed := rl.Sweep(0); removed != 0 {
			t.Fatalf("expected 0 swept entries with IdleTTL=-1, got %d", removed)
		}
	})
}

func TestRestClientNestedRateLimiterConfig(t *testing.T) {
	t.Parallel()

	var nowMu sync.Mutex
	cur := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	nowFn := func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return cur
	}
	advance := func(d time.Duration) {
		nowMu.Lock()
		cur = cur.Add(d)
		nowMu.Unlock()
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", rest.ContentTypeJSON)
		w.Header().Set("X-RateLimit-Bucket", "nested-bucket")
		w.Header().Set("X-RateLimit-Limit", "1")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset-After", "0.05")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	var limLogBuf bytes.Buffer
	client := rest.NewRestClient(rest.RestClientConfig{
		Token:   testToken("777"),
		BaseURL: srv.URL + "/api/v10",
		Trace:   false,
		RateLimiterConfig: rest.RateLimiterConfig{
			Now:           nowFn,
			Shards:        3, // Rounds up to 4
			MaxBuckets:    4,
			MaxRoutes:     4,
			SweepInterval: 2,
			IdleTTL:       time.Minute,
			ResetFudge:    -1,
			GlobalRate:    10_000,
			Trace:         true,
			TraceLogger:   log.New(&limLogBuf, "", 0),
		},
	})

	for i := range 12 {
		route := "/channels/" + strconv.Itoa(100+i) + "/messages"
		if err := client.Request(t.Context(), http.MethodPost, route, map[string]string{"content": "hi"}, nil); err != nil {
			t.Fatalf("Request(%s): %v", route, err)
		}
	}

	buckets, routes := client.Limiter().Size()
	if buckets > 4 || routes > 4 {
		t.Fatalf("nested MaxBuckets/MaxRoutes violated: buckets=%d routes=%d", buckets, routes)
	}

	// Deplete channel 200 and verify ResetFudge=-1 allows retry after advancing 60ms past 50ms window.
	if err := client.Request(t.Context(), http.MethodPost, "/channels/200/messages", nil, nil); err != nil {
		t.Fatalf("seed depleted route: %v", err)
	}

	shortCtx, shortCancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer shortCancel()

	if err := client.Request(shortCtx, http.MethodPost, "/channels/200/messages", nil, nil); err == nil {
		t.Fatal("expected deadline error while route is cooling down")
	}

	// Verify nested TraceLogger + Trace=true logged the rate-limit wait (not cleared by parent Trace=false).
	if limLogBuf.Len() == 0 {
		t.Fatal("expected nested RateLimiterConfig.TraceLogger to receive trace logs")
	}

	advance(60 * time.Millisecond)

	if err := client.Request(t.Context(), http.MethodPost, "/channels/200/messages", nil, nil); err != nil {
		t.Fatalf("expected immediate request after 60ms with ResetFudge=-1: %v", err)
	}

	// Idle buckets reclaim after nested IdleTTL (1m).
	advance(2 * time.Minute)

	if removed := client.Limiter().Sweep(0); removed == 0 {
		t.Fatal("expected Sweep(0) to remove expired buckets from nested IdleTTL")
	}

	// Verify small MaxBuckets/MaxRoutes without explicit Shards auto-scales shards so cap holds.
	tinyRL := rest.NewRateLimiter(rest.RateLimiterConfig{MaxBuckets: 8, MaxRoutes: 8})
	for i := range 24 {
		cb, err := tinyRL.Wait(t.Context(), http.MethodGet, "/guilds/"+strconv.Itoa(i+1)+"/channels")
		if err != nil {
			t.Fatalf("tinyRL.Wait: %v", err)
		}
		cb(nil)
	}

	if tinyB, tinyR := tinyRL.Size(); tinyB > 8 || tinyR > 8 {
		t.Fatalf("expected tiny MaxBuckets/MaxRoutes <= 8, got buckets=%d routes=%d", tinyB, tinyR)
	}

	// Ensure Trace=true without custom TraceLogger works cleanly for both client and standalone limiter.
	_ = rest.NewRateLimiter(rest.RateLimiterConfig{Trace: true})
	_ = rest.NewRestClient(rest.RestClientConfig{Token: testToken("778"), Trace: true})
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read boom")
}

func TestCustomDoAndPayloadParity(t *testing.T) {
	t.Parallel()

	t.Run("Request and Do produce identical wire JSON across small and nested payloads", func(t *testing.T) {
		var mu sync.Mutex
		var bodies [][]byte

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			mu.Lock()
			bodies = append(bodies, b)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("888"),
			BaseURL: srv.URL + "/api/v10",
		})

		type nestedOption struct {
			Label string `json:"label"`
			Value string `json:"value"`
		}
		type nestedNode struct {
			Content    string         `json:"content,omitzero"`
			CustomID   string         `json:"custom_id,omitzero"`
			Required   *bool          `json:"required,omitzero"`
			Components []nestedNode   `json:"components,omitzero"`
			Options    []nestedOption `json:"options,omitzero"`
			Type       uint8          `json:"type"`
		}

		cv2Like := struct {
			Components []nestedNode `json:"components"`
			Flags      uint64       `json:"flags"`
		}{
			Flags: 1 << 15,
			Components: []nestedNode{{
				Type: 17,
				Components: []nestedNode{
					{Type: 10, Content: "Header text"},
					{
						Type:     18,
						CustomID: "lbl",
						Required: new(true),
						Options:  []nestedOption{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}},
					},
				},
			}},
		}

		rawCV2, err := json.Marshal(cv2Like)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		if err := client.Request(t.Context(), http.MethodPost, "/channels/1/messages", cv2Like, nil); err != nil {
			t.Fatalf("Request: %v", err)
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/channels/1/messages",
			Body:   cv2Like,
		}); err != nil {
			t.Fatalf("Do struct: %v", err)
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/channels/1/messages",
			Body:   jsontext.Value(rawCV2),
		}); err != nil {
			t.Fatalf("Do jsontext.Value: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		if len(bodies) != 3 {
			t.Fatalf("expected 3 captured request bodies, got %d", len(bodies))
		}

		if !bytes.Equal(bodies[0], bodies[1]) || !bytes.Equal(bodies[1], bodies[2]) {
			t.Fatalf("wire mismatch:\nRequest: %s\nDo(struct): %s\nDo(raw): %s", bodies[0], bodies[1], bodies[2])
		}
	})

	t.Run("Do custom content-type reader failure and HTTPError", func(t *testing.T) {
		var sawCT string
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			sawCT = r.Header.Get("Content-Type")
			_, _ = io.Copy(io.Discard, r.Body)

			if r.URL.Path == "/api/v10/err" {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"Missing Permissions"}`))
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("889"),
			BaseURL: srv.URL + "/api/v10",
		})

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/fail-read",
			Body:   io.NopCloser(failingReader{}),
		}); err == nil {
			t.Fatal("expected error for failing io.Reader body")
		}

		if hits.Load() != 0 {
			t.Fatalf("expected read error before dialing server, got %d requests", hits.Load())
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method:      http.MethodPatch,
			Route:       "/merge-patch",
			Body:        map[string]string{"name": "updated"},
			ContentType: "application/merge-patch+json",
		}); err != nil {
			t.Fatalf("Do custom content type: %v", err)
		}

		if sawCT != "application/merge-patch+json" {
			t.Fatalf("expected custom content type, got %q", sawCT)
		}

		err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodGet,
			Route:  "/err",
		})
		var httpErr *rest.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 *HTTPError from Do, got %v", err)
		}

		if err := client.Do(t.Context(), rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/fail-file",
			Files:  []rest.File{{Name: "bad.bin", Reader: failingReader{}}},
		}); err == nil {
			t.Fatal("expected multipart file error from failing reader")
		}
	})
}

func TestBufferPoolDiscardsOversizedAndReturnsToBaseline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)

		if r.URL.Path == "/api/v10/huge-resp" {
			// Return >64 KiB JSON response to exercise streaming decode path without pooling bloat.
			w.Header().Set("Content-Type", rest.ContentTypeJSON)
			_, _ = w.Write([]byte(`{"data":"`))
			_, _ = w.Write(bytes.Repeat([]byte("A"), 256*1024))
			_, _ = w.Write([]byte(`"}`))
			return
		}

		w.Header().Set("Content-Type", rest.ContentTypeJSON)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := rest.NewRestClient(rest.RestClientConfig{
		Token:             testToken("900"),
		BaseURL:           srv.URL + "/api/v10",
		MaxFileUploadSize: -1,
	})

	// Warm normal small pools before taking baseline heap snapshot.
	for range 16 {
		if err := client.Request(t.Context(), http.MethodPost, "/channels/1/messages", map[string]string{"content": "warm"}, nil); err != nil {
			t.Fatalf("warmup request: %v", err)
		}
	}

	runtime.GC()
	runtime.GC()
	var baseMS runtime.MemStats
	runtime.ReadMemStats(&baseMS)

	// Trigger absurdly oversized buffers (>64 KiB JSON reader, >512 KiB multipart, >64 KiB JSON response).
	bigBody := bytes.Repeat([]byte("B"), 2*1024*1024)
	bigFile := bytes.Repeat([]byte("F"), 1024*1024)

	for range 4 {
		if err := client.Do(t.Context(), rest.CustomRequest{
			Method:      http.MethodPost,
			Route:       "/huge-reader",
			Body:        io.NopCloser(bytes.NewReader(bigBody)),
			ContentType: rest.ContentTypeOctetStream,
		}); err != nil {
			t.Fatalf("oversized Do io.Reader: %v", err)
		}

		if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/huge-multipart", map[string]string{"c": "big"}, []rest.File{{
			Name:   "big.bin",
			Reader: bytes.NewReader(bigFile),
		}}, nil); err != nil {
			t.Fatalf("oversized RequestWithFiles: %v", err)
		}

		var bigResp struct {
			Data string `json:"data"`
		}
		if err := client.Request(t.Context(), http.MethodGet, "/huge-resp", nil, &bigResp); err != nil {
			t.Fatalf("oversized JSON response: %v", err)
		}

		if len(bigResp.Data) != 256*1024 {
			t.Fatalf("unexpected decoded length: %d", len(bigResp.Data))
		}
	}

	runtime.GC()
	runtime.GC()
	var afterMS runtime.MemStats
	runtime.ReadMemStats(&afterMS)

	// Two full GCs drain sync.Pool and oversized buffers were never kept in pool.
	const maxRetainedDelta = 2 * 1024 * 1024
	if afterMS.HeapAlloc > baseMS.HeapAlloc+maxRetainedDelta {
		t.Fatalf("oversized buffers retained in heap: baseline=%d after=%d (delta=%d)",
			baseMS.HeapAlloc, afterMS.HeapAlloc, afterMS.HeapAlloc-baseMS.HeapAlloc)
	}
}

func TestRateLimiterBurstUniqueEndpointsGlobal50AndMemoryReclaim(t *testing.T) {
	t.Run("unique endpoints obey global 50 req/s cap", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := w.Header()
			hdr.Set("Content-Type", rest.ContentTypeJSON)
			hdr.Set("X-RateLimit-Bucket", "u-bucket")
			hdr.Set("X-RateLimit-Limit", "50")
			hdr.Set("X-RateLimit-Remaining", "49")
			hdr.Set("X-RateLimit-Reset-After", "1.0")
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("950"),
			BaseURL: srv.URL + "/api/v10",
			RateLimiterConfig: rest.RateLimiterConfig{
				GlobalRate: 50,
				IdleTTL:    100 * time.Millisecond,
			},
		})

		const totalCalls = 75
		routes := make([]struct {
			method string
			path   string
		}, totalCalls)
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete}

		for i := range totalCalls {
			id := strconv.Itoa(100000 + i)
			switch i % 5 {
			case 0:
				routes[i] = struct{ method, path string }{methods[0], "/users/" + id}
			case 1:
				routes[i] = struct{ method, path string }{methods[1], "/channels/" + id + "/messages"}
			case 2:
				routes[i] = struct{ method, path string }{methods[2], "/guilds/" + id}
			case 3:
				routes[i] = struct{ method, path string }{methods[3], "/channels/" + id + "/reactions/custom%3A1/@me"}
			default:
				routes[i] = struct{ method, path string }{methods[4], "/guilds/" + id + "/bans/" + strconv.Itoa(500+i)}
			}
		}

		start := time.Now()
		var wg sync.WaitGroup
		var errs atomic.Int32

		for i := range totalCalls {
			wg.Go(func() {
				if err := client.Request(t.Context(), routes[i].method, routes[i].path, nil, nil); err != nil {
					errs.Add(1)
				}
			})
		}
		wg.Wait()
		elapsed := time.Since(start)

		if errs.Load() != 0 {
			t.Fatalf("expected 0 request errors under global 50 req/s limit, got %d", errs.Load())
		}

		// Burst capacity = 50; remaining 25 requests at 50/s require ~500ms.
		if elapsed < 350*time.Millisecond {
			t.Fatalf("expected 75 requests at GlobalRate=50 to take >= 350ms, finished too fast in %s", elapsed)
		}

		buckets, _ := client.Limiter().Size()
		if buckets == 0 {
			t.Fatal("expected unique endpoints to populate rate limiter buckets during burst")
		}
	})

	t.Run("burst of thousands of endpoints clears automatically and releases heap when idle", func(t *testing.T) {
		rl := rest.NewRateLimiter(rest.RateLimiterConfig{
			Shards:     16,
			IdleTTL:    250 * time.Millisecond,
			ResetFudge: -1,
			MaxBuckets: 16384,
			MaxRoutes:  16384,
		})

		runtime.GC()
		runtime.GC()
		var baseMS runtime.MemStats
		runtime.ReadMemStats(&baseMS)

		const uniqueEndpoints = 4096
		hdr := http.Header{}
		hdr.Set("X-RateLimit-Bucket", "burst-bucket")
		hdr.Set("X-RateLimit-Limit", "5")
		hdr.Set("X-RateLimit-Remaining", "4")
		hdr.Set("X-RateLimit-Reset-After", "0.01")

		for i := range uniqueEndpoints {
			path := "/channels/" + strconv.Itoa(1_000_000+i) + "/messages/" + strconv.Itoa(i)
			cb, err := rl.Wait(t.Context(), http.MethodPost, path)
			if err != nil {
				t.Fatalf("Wait(%d): %v", i, err)
			}
			cb(hdr)
		}

		peakBuckets, peakRoutes := rl.Size()
		if peakBuckets != uniqueEndpoints || peakRoutes != uniqueEndpoints {
			t.Fatalf("expected %d tracked buckets/routes during burst, got buckets=%d routes=%d",
				uniqueEndpoints, peakBuckets, peakRoutes)
		}

		var peakMS runtime.MemStats
		runtime.ReadMemStats(&peakMS)

		// Stop bombarding and wait for automatic idle cleanup with zero manual Sweep() calls.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			b, r := rl.Size()
			if b == 0 && r == 0 {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}

		afterBuckets, afterRoutes := rl.Size()
		if afterBuckets != 0 || afterRoutes != 0 {
			t.Fatalf("expected auto-sweep to clear all idle entries after bombardment, got buckets=%d routes=%d",
				afterBuckets, afterRoutes)
		}

		runtime.GC()
		runtime.GC()
		var reclaimedMS runtime.MemStats
		runtime.ReadMemStats(&reclaimedMS)

		if reclaimedMS.HeapAlloc >= peakMS.HeapAlloc {
			t.Fatalf("expected heap allocation to shrink after idle cleanup (baseline=%d peak=%d reclaimed=%d)",
				baseMS.HeapAlloc, peakMS.HeapAlloc, reclaimedMS.HeapAlloc)
		}
	})
}

func TestHTTPErrorCodeAndBody(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		status     string
		body       string
		wantSubstr string
		wantMsg    string
		statusCode int
		wantCode   int
	}{
		{
			name:       "nested 50035 preserves raw JSON body",
			statusCode: http.StatusBadRequest,
			status:     "400 Bad Request",
			body:       `{"code":50035,"message":"Invalid Form Body","errors":{"data":{"components":{"0":{"components":{"0":{"_errors":[{"code":"BASE_TYPE_MAX_LENGTH","message":"Must be 5 or fewer"}]}}}}}}}`,
			wantCode:   50035,
			wantMsg:    "Invalid Form Body",
			wantSubstr: `Discord API error 50035 (400 Bad Request): {"code":50035,"message":"Invalid Form Body","errors":`,
		},
		{
			name:       "simple JSON code",
			statusCode: http.StatusForbidden,
			status:     "403 Forbidden",
			body:       `{"code":50013,"message":"Missing Permissions"}`,
			wantCode:   50013,
			wantMsg:    "Missing Permissions",
			wantSubstr: `Discord API error 50013 (403 Forbidden): {"code":50013,"message":"Missing Permissions"}`,
		},
		{
			name:       "plain non-JSON body",
			statusCode: http.StatusNotFound,
			status:     "404 Not Found",
			body:       `Not Found`,
			wantCode:   0,
			wantSubstr: "request failed with status 404 Not Found: Not Found",
		},
		{
			name:       "malformed JSON body falls back to raw error",
			statusCode: http.StatusBadRequest,
			status:     "400 Bad Request",
			body:       `{not-valid-json`,
			wantCode:   0,
			wantSubstr: "request failed with status 400 Bad Request: {not-valid-json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := rest.NewRestClient(rest.RestClientConfig{
				Token:   testToken("960"),
				BaseURL: srv.URL + "/api/v10",
			})

			err := client.Request(t.Context(), http.MethodPost, "/channels/1/messages", nil, nil)
			if err == nil {
				t.Fatal("expected *HTTPError")
			}

			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("Error() mismatch:\ngot:  %q\nwant substring: %q", err.Error(), tc.wantSubstr)
			}

			var httpErr *rest.HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("expected *rest.HTTPError, got %T", err)
			}

			if httpErr.Code != tc.wantCode || httpErr.StatusCode != tc.statusCode || httpErr.Message != tc.wantMsg {
				t.Fatalf("got Code=%d Message=%q StatusCode=%d, want Code=%d Message=%q StatusCode=%d",
					httpErr.Code, httpErr.Message, httpErr.StatusCode, tc.wantCode, tc.wantMsg, tc.statusCode)
			}

			if string(httpErr.Body) != tc.body {
				t.Fatalf("Body = %q, want %q", string(httpErr.Body), tc.body)
			}
		})
	}
}

func TestRateLimiterDisgoEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("X-RateLimit-Reset fallback and deadline short-circuit", func(t *testing.T) {
		var nowMu sync.Mutex
		cur := time.Unix(1_700_000_000, 0)
		nowFn := func() time.Time {
			nowMu.Lock()
			defer nowMu.Unlock()
			return cur
		}
		advance := func(d time.Duration) {
			nowMu.Lock()
			cur = cur.Add(d)
			nowMu.Unlock()
		}

		rl := rest.NewRateLimiter(rest.RateLimiterConfig{
			Now:        nowFn,
			Shards:     1,
			ResetFudge: -1,
		})

		done, err := rl.Wait(t.Context(), http.MethodPost, "/channels/55/messages")
		if err != nil {
			t.Fatal(err)
		}

		// Provide Unix epoch X-RateLimit-Reset (+10s) without X-RateLimit-Reset-After.
		hdr := http.Header{}
		hdr.Set("X-RateLimit-Bucket", "unix-reset")
		hdr.Set("X-RateLimit-Limit", "1")
		hdr.Set("X-RateLimit-Remaining", "0")
		hdr.Set("X-RateLimit-Reset", strconv.FormatInt(cur.Add(10*time.Second).Unix(), 10))
		done(hdr)

		// Waiting with a 50ms deadline must fail immediately in < 15ms instead of blocking the full 50ms.
		shortCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err = rl.Wait(shortCtx, http.MethodPost, "/channels/55/messages")
		elapsed := time.Since(start)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded, got %v", err)
		}

		if elapsed > 20*time.Millisecond {
			t.Fatalf("expected immediate short-circuit (< 20ms), took %s", elapsed)
		}

		// Advancing past X-RateLimit-Reset opens the bucket cleanly.
		advance(11 * time.Second)

		okDone, err := rl.Wait(t.Context(), http.MethodPost, "/channels/55/messages")
		if err != nil {
			t.Fatalf("expected Wait to succeed after X-RateLimit-Reset elapsed: %v", err)
		}
		okDone(nil)
	})

	t.Run("Cloudflare non-JSON 429 without Via header triggers global pause", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("error code: 1015"))
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("961"),
			BaseURL:    srv.URL + "/api/v10",
			MaxRetries: 2,
		})

		ctx, cancel := context.WithTimeout(t.Context(), 60*time.Millisecond)
		defer cancel()

		// Non-JSON 429 without Via triggers a ~5s global pause, so retry short-circuits immediately on ctx deadline.
		start := time.Now()
		err := client.Request(ctx, http.MethodGet, "/channels/1/messages", nil, nil)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected immediate DeadlineExceeded from global pause short-circuit, got %v", err)
		}

		if time.Since(start) > 45*time.Millisecond {
			t.Fatalf("expected deadline short-circuit against Cloudflare global pause, took %s", time.Since(start))
		}
	})

	t.Run("Disgo endpoint catalog RouteKey normalization", func(t *testing.T) {
		cases := []struct {
			method string
			path   string
			want   string
		}{
			{http.MethodGet, "/api/v10/users/1234567890", "GET /users/{id}"},
			{http.MethodGet, "/api/v10/guilds/111/members/222", "GET /guilds/111/members/{id}"},
			{http.MethodPut, "/api/v10/channels/333/reactions/%F0%9F%94%A5/@me", "PUT /channels/333/reactions/{emoji}/@me"},
			{http.MethodDelete, "/api/v10/channels/333/reactions/%F0%9F%94%A5/999", "DELETE /channels/333/reactions/{emoji}/{id}"},
			{http.MethodPatch, "/api/v10/webhooks/444/secret-token/messages/@original?wait=true", "PATCH /webhooks/444/{token}/messages/@original"},
			{http.MethodPost, "/api/v10/interactions/555/itx-token/callback", "POST /interactions/{id}/{token}/callback"},
		}

		for _, tc := range cases {
			if got := rest.RouteKey(tc.method, tc.path); got != tc.want {
				t.Fatalf("RouteKey(%s, %s) = %q, want %q", tc.method, tc.path, got, tc.want)
			}
		}
	})
}

type chunkedSlowReader struct {
	sent atomic.Bool
}

func (r *chunkedSlowReader) Read(p []byte) (int, error) {
	if !r.sent.Swap(true) {
		n := copy(p, `{"partial":`)
		return n, nil
	}

	time.Sleep(5 * time.Millisecond)
	return copy(p, " "), nil
}

func TestContextDeadlineAtEveryStage(t *testing.T) {
	t.Parallel()

	t.Run("already expired deadline fails before dialing", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("970"),
			BaseURL: srv.URL + "/api/v10",
		})

		expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		if err := client.Request(expiredCtx, http.MethodGet, "/users/@me", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Request expired ctx: got %v, want DeadlineExceeded", err)
		}

		if err := client.RequestWithFiles(expiredCtx, http.MethodPost, "/channels/1/messages", nil, []rest.File{{Name: "a.txt", Reader: strings.NewReader("x")}}, nil); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("RequestWithFiles expired ctx: got %v, want DeadlineExceeded", err)
		}

		if err := client.Do(expiredCtx, rest.CustomRequest{Method: http.MethodPost, Route: "/channels/1/messages", Body: map[string]int{"x": 1}}); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Do expired ctx: got %v, want DeadlineExceeded", err)
		}

		if hits.Load() != 0 {
			t.Fatalf("expected 0 requests sent with expired ctx, got %d", hits.Load())
		}
	})

	t.Run("deadline expires while streaming custom io.Reader or multipart file", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:   testToken("971"),
			BaseURL: srv.URL + "/api/v10",
		})

		readerCtx, cancelReader := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancelReader()

		if err := client.Do(readerCtx, rest.CustomRequest{
			Method: http.MethodPost,
			Route:  "/channels/1/messages",
			Body:   &chunkedSlowReader{},
		}); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded streaming Do io.Reader, got %v", err)
		}

		fileCtx, cancelFile := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancelFile()

		if err := client.RequestWithFiles(fileCtx, http.MethodPost, "/channels/1/messages", nil, []rest.File{{
			Name:   "slow.bin",
			Reader: &chunkedSlowReader{},
		}}, nil); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded streaming multipart File, got %v", err)
		}

		if hits.Load() != 0 {
			t.Fatalf("expected 0 requests sent when upload stream timed out, got %d", hits.Load())
		}
	})

	t.Run("deadline expires while waiting on response headers without extra retry", func(t *testing.T) {
		var attempts atomic.Int32
		releaseServer := make(chan struct{})

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			select {
			case <-releaseServer:
			case <-r.Context().Done():
			}
		}))
		defer func() {
			close(releaseServer)
			srv.Close()
		}()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("972"),
			BaseURL:    srv.URL + "/api/v10",
			MaxRetries: 3,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()

		var out map[string]any
		if err := client.Request(ctx, http.MethodGet, "/slow-headers", nil, &out); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded waiting for headers, got %v", err)
		}

		if attempts.Load() != 1 {
			t.Fatalf("expected single attempt when ctx deadline expired, got %d", attempts.Load())
		}
	})

	t.Run("deadline expires mid-response body across JSON raw bytes Writer and nil target", func(t *testing.T) {
		releaseBody := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", rest.ContentTypeJSON)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"1",`))

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			select {
			case <-releaseBody:
			case <-r.Context().Done():
			}
		}))
		defer func() {
			close(releaseBody)
			srv.Close()
		}()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("973"),
			BaseURL:    srv.URL + "/api/v10",
			MaxRetries: 3,
		})

		for _, tc := range []struct {
			call func(ctx context.Context) error
			name string
		}{
			{
				name: "json decode target",
				call: func(ctx context.Context) error {
					var out map[string]any
					return client.Request(ctx, http.MethodGet, "/partial-body", nil, &out)
				},
			},
			{
				name: "raw bytes target",
				call: func(ctx context.Context) error {
					var raw []byte
					return client.Request(ctx, http.MethodGet, "/partial-body", nil, &raw)
				},
			},
			{
				name: "io.Writer target",
				call: func(ctx context.Context) error {
					var buf bytes.Buffer
					return client.Do(ctx, rest.CustomRequest{Method: http.MethodGet, Route: "/partial-body", Into: &buf})
				},
			},
			{
				name: "nil target drain",
				call: func(ctx context.Context) error {
					return client.Request(ctx, http.MethodGet, "/partial-body", nil, nil)
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
				defer cancel()

				if err := tc.call(ctx); !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected DeadlineExceeded mid-body (%s), got %v", tc.name, err)
				}
			})
		}
	})

	t.Run("deadline expires during retry backoff sleep and stops cleanly", func(t *testing.T) {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("974"),
			BaseURL:    srv.URL + "/api/v10",
			MaxRetries: 3,
		})

		// First attempt returns 503 immediately; backoff sleep is 250-350ms, so 50ms deadline triggers mid-sleep.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		if err := client.Request(ctx, http.MethodGet, "/retry-sleep", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded during retry sleep, got %v", err)
		}

		if attempts.Load() != 1 {
			t.Fatalf("expected retry loop to stop after attempt 1 on ctx deadline, got %d attempts", attempts.Load())
		}
	})

	t.Run("MaxWaitTime bounds background context cleanly", func(t *testing.T) {
		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}))
		defer func() {
			close(release)
			srv.Close()
		}()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:       testToken("975"),
			BaseURL:     srv.URL + "/api/v10",
			MaxWaitTime: 50 * time.Millisecond,
			MaxRetries:  1,
		})

		if err := client.Request(context.Background(), http.MethodGet, "/budget", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected MaxWaitTime DeadlineExceeded with context.Background(), got %v", err)
		}
	})
}

type badReadSeeker struct{}

func (badReadSeeker) Read(p []byte) (int, error) { return 0, io.EOF }
func (badReadSeeker) Seek(int64, int) (int64, error) {
	return 0, errors.New("seek broken")
}

func TestRestCoverageEdgePaths(t *testing.T) {
	t.Parallel()

	t.Run("marshal errors invalid URL method seek error and empty files", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		// Custom HTTPClient with nil Transport falls back to http.DefaultTransport cleanly.
		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("980"),
			BaseURL:    srv.URL + "/api/v10",
			HTTPClient: &http.Client{Timeout: time.Second},
		})

		unmarshalable := make(chan int)
		if err := client.Request(t.Context(), http.MethodPost, "/channels/1/messages", unmarshalable, nil); err == nil {
			t.Fatal("expected JSON marshal error on Request")
		}

		if err := client.Do(t.Context(), rest.CustomRequest{Method: http.MethodPost, Route: "/channels/1/messages", Body: unmarshalable}); err == nil {
			t.Fatal("expected JSON marshal error on Do")
		}

		if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/channels/1/messages", unmarshalable, []rest.File{{Name: "a.txt", Reader: strings.NewReader("x")}}, nil); err == nil {
			t.Fatal("expected JSON marshal error on RequestWithFiles")
		}

		if err := client.Request(t.Context(), http.MethodGet, "/bad\x7f?x=1", nil, nil); err == nil {
			t.Fatal("expected url.Parse error for invalid route")
		}

		if err := client.Request(t.Context(), "BAD METHOD", "/users/@me", nil, nil); err == nil {
			t.Fatal("expected http.NewRequestWithContext error for invalid method")
		}

		if err := client.Do(t.Context(), rest.CustomRequest{Method: http.MethodPost, Route: "/seek", Body: badReadSeeker{}}); err == nil {
			t.Fatal("expected seek error from badReadSeeker")
		}

		if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/channels/1/messages", nil, []rest.File{{Name: "a.txt", Reader: badReadSeeker{}}}, nil); err == nil {
			t.Fatal("expected seek error from file badReadSeeker")
		}

		if err := client.RequestWithFiles(t.Context(), http.MethodPost, "/channels/1/messages", nil, nil, nil); err != nil {
			t.Fatalf("expected RequestWithFiles with empty files to succeed: %v", err)
		}

		if hits.Load() != 1 {
			t.Fatalf("expected 1 server hit from empty-files fallback, got %d", hits.Load())
		}
	})

	t.Run("header global 429 retry-after 429 floor 429 retry exhaustion and malformed JSON responses", func(t *testing.T) {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v10/header-global-ra":
				if attempts.Add(1) == 1 {
					w.Header().Set("X-RateLimit-Global", "true")
					w.Header().Set("Retry-After", "0.01")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/header-global-nora":
				if attempts.Add(1) == 1 {
					w.Header().Set("X-RateLimit-Scope", "global")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/via-retry-after":
				if attempts.Add(1) == 1 {
					w.Header().Set("Via", "1.1 google")
					w.Header().Set("Retry-After", "0.01")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/via-floor":
				if attempts.Add(1) == 1 {
					w.Header().Set("Via", "1.1 google")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusNoContent)

			case "/api/v10/exhaust-get":
				w.WriteHeader(http.StatusServiceUnavailable)

			case "/api/v10/bad-json-cl":
				w.Header().Set("Content-Type", rest.ContentTypeJSON)
				w.Header().Set("Content-Length", "3")
				_, _ = w.Write([]byte("{bad"))

			case "/api/v10/bad-json-stream":
				w.Header().Set("Content-Type", rest.ContentTypeJSON)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{bad-stream"))

			case "/api/v10/truncated-raw":
				w.Header().Set("Content-Length", "50")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("short"))
			}
		}))
		defer srv.Close()

		client := rest.NewRestClient(rest.RestClientConfig{
			Token:      testToken("981"),
			BaseURL:    srv.URL + "/api/v10",
			MaxRetries: 2,
		})

		for _, route := range []string{"/header-global-ra", "/header-global-nora", "/via-retry-after", "/via-floor"} {
			attempts.Store(0)
			if err := client.Request(t.Context(), http.MethodPost, route, nil, nil); err != nil {
				t.Fatalf("route %s failed: %v", route, err)
			}
		}

		if err := client.Request(t.Context(), http.MethodGet, "/exhaust-get", nil, nil); err == nil {
			t.Fatal("expected error after exhausting MaxRetries")
		}

		var out map[string]any
		if err := client.Request(t.Context(), http.MethodGet, "/bad-json-cl", nil, &out); err == nil {
			t.Fatal("expected JSON decode error for small invalid payload")
		}

		if err := client.Request(t.Context(), http.MethodGet, "/bad-json-stream", nil, &out); err == nil {
			t.Fatal("expected JSON decode error for streaming invalid payload")
		}

		var raw []byte
		if err := client.Request(t.Context(), http.MethodGet, "/truncated-raw", nil, &raw); err == nil {
			t.Fatal("expected error reading truncated Content-Length body")
		}
	})

	t.Run("expired entry batch eviction", func(t *testing.T) {
		var nowMu sync.Mutex
		cur := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		nowFn := func() time.Time {
			nowMu.Lock()
			defer nowMu.Unlock()
			return cur
		}
		advance := func(d time.Duration) {
			nowMu.Lock()
			cur = cur.Add(d)
			nowMu.Unlock()
		}

		rl := rest.NewRateLimiter(rest.RateLimiterConfig{
			Now:           nowFn,
			Shards:        1,
			MaxBuckets:    2,
			MaxRoutes:     2,
			IdleTTL:       time.Second,
			SweepInterval: 1000,
		})

		for i := range 2 {
			cb, err := rl.Wait(t.Context(), http.MethodGet, "/guilds/"+strconv.Itoa(i+1)+"/channels")
			if err != nil {
				t.Fatal(err)
			}
			h := http.Header{}
			h.Set("X-RateLimit-Bucket", "b"+strconv.Itoa(i))
			h.Set("X-RateLimit-Limit", "5")
			h.Set("X-RateLimit-Remaining", "4")
			h.Set("X-RateLimit-Reset-After", "0.1")
			cb(h)
		}

		advance(5 * time.Second)

		// Inserting a 3rd guild triggers expired-bucket and expired-route paths inside evictBatchLocked.
		cb3, err := rl.Wait(t.Context(), http.MethodGet, "/guilds/3/channels")
		if err != nil {
			t.Fatal(err)
		}
		h3 := http.Header{}
		h3.Set("X-RateLimit-Bucket", "b3")
		h3.Set("X-RateLimit-Limit", "5")
		h3.Set("X-RateLimit-Remaining", "4")
		h3.Set("X-RateLimit-Reset-After", "0.1")
		cb3(h3)

		if b3, r3 := rl.Size(); b3 > 2 || r3 > 2 {
			t.Fatalf("expected MaxBuckets/MaxRoutes <= 2, got buckets=%d routes=%d", b3, r3)
		}

		if removed := rl.Sweep(0); removed != 1 {
			t.Fatalf("expected Sweep(0) to remove the remaining expired guild, removed=%d", removed)
		}

		if bAfter, rAfter := rl.Size(); bAfter != 1 || rAfter != 1 {
			t.Fatalf("expected 1 fresh bucket/route remaining, got buckets=%d routes=%d", bAfter, rAfter)
		}
	})
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
