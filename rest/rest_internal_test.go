package rest

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultTransportHTTP2AndIdleConfig(t *testing.T) {
	t.Parallel()

	client := NewRestClient(RestClientConfig{
		Token: "MTIz.test.sig",
	})

	rt, ok := client.HTTPClient.Transport.(*rateLimitTransport)
	if !ok {
		t.Fatalf("expected *rateLimitTransport wrapper, got %T", client.HTTPClient.Transport)
	}

	tr, ok := rt.innerTransport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", rt.innerTransport)
	}

	if tr.IdleConnTimeout != 60*time.Second {
		t.Fatalf("expected default IdleConnTimeout 60s, got %s", tr.IdleConnTimeout)
	}

	if tr.ResponseHeaderTimeout != 10*time.Second {
		t.Fatalf("expected default ResponseHeaderTimeout 10s, got %s", tr.ResponseHeaderTimeout)
	}

	if tr.HTTP2 == nil {
		t.Fatal("expected default HTTP2Config to be initialized")
	}

	if tr.HTTP2.SendPingTimeout != 15*time.Second || tr.HTTP2.PingTimeout != 10*time.Second || tr.HTTP2.WriteByteTimeout != 15*time.Second {
		t.Fatalf("unexpected default HTTP2 health timeouts: %+v", tr.HTTP2)
	}

	h1Only := NewRestClient(RestClientConfig{
		Token:                 "MTIz.test.sig",
		ForceAttemptHTTP2:     new(false),
		ResponseHeaderTimeout: -1,
	})
	h1Tr := h1Only.HTTPClient.Transport.(*rateLimitTransport).innerTransport.(*http.Transport)

	if h1Tr.HTTP2 != nil {
		t.Fatalf("expected nil HTTP2 config when ForceAttemptHTTP2=false, got %+v", h1Tr.HTTP2)
	}

	if h1Tr.ResponseHeaderTimeout != 0 {
		t.Fatalf("expected ResponseHeaderTimeout=0 when disabled (< 0), got %s", h1Tr.ResponseHeaderTimeout)
	}

	customH2 := &http.HTTP2Config{
		SendPingTimeout:  5 * time.Second,
		PingTimeout:      3 * time.Second,
		WriteByteTimeout: 4 * time.Second,
	}
	customClient := NewRestClient(RestClientConfig{
		Token: "MTIz.test.sig",
		HTTP2: customH2,
	})
	customTr := customClient.HTTPClient.Transport.(*rateLimitTransport).innerTransport.(*http.Transport)

	if customTr.HTTP2 == nil || customTr.HTTP2 == customH2 {
		t.Fatal("expected HTTP2 config to be cloned rather than aliasing caller pointer")
	}

	if customTr.HTTP2.SendPingTimeout != 5*time.Second || customTr.HTTP2.PingTimeout != 3*time.Second {
		t.Fatalf("unexpected custom HTTP2 config: %+v", customTr.HTTP2)
	}
}

func TestHTTP2ConnectionReuseIdleCleanupAndDeadPeerRecovery(t *testing.T) {
	t.Parallel()

	var newConns atomic.Int32
	var h2Requests atomic.Int32

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 {
			h2Requests.Add(1)
		}

		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	srv.EnableHTTP2 = true
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConns.Add(1)
		}
	}
	srv.StartTLS()
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())

	client := NewRestClient(RestClientConfig{
		Token:           "MTIz.test.sig",
		BaseURL:         srv.URL + "/api/v10",
		IdleConnTimeout: 120 * time.Millisecond,
		HTTP2: &http.HTTP2Config{
			SendPingTimeout:  25 * time.Millisecond,
			PingTimeout:      50 * time.Millisecond,
			WriteByteTimeout: time.Second,
		},
	})
	tr := client.HTTPClient.Transport.(*rateLimitTransport).innerTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
	}

	// Phase 1: Reuse a single HTTP/2 connection across repeated requests.
	for range 10 {
		var out struct {
			OK bool `json:"ok"`
		}
		if err := client.Request(t.Context(), http.MethodPost, "/channels/1/messages", map[string]string{"content": "h2"}, &out); err != nil {
			t.Fatalf("HTTP/2 initial request failed: %v", err)
		}

		if !out.OK {
			t.Fatal("expected ok=true response")
		}
	}

	if h2Requests.Load() != 10 {
		t.Fatalf("expected all 10 requests over HTTP/2, got %d", h2Requests.Load())
	}

	if newConns.Load() != 1 {
		t.Fatalf("expected HTTP/2 connection reuse (1 conn), got %d new conns", newConns.Load())
	}

	// Phase 2: Abruptly drop active client connection and pause briefly beyond SendPingTimeout,
	// then ensure both GET and non-idempotent POST recover transparently via fresh HTTP/2 connection.
	srv.CloseClientConnections()
	time.Sleep(40 * time.Millisecond)

	if err := client.Request(t.Context(), http.MethodGet, "/users/@me", nil, nil); err != nil {
		t.Fatalf("expected transparent GET recovery after closed idle HTTP/2 conn: %v", err)
	}

	if err := client.Do(t.Context(), CustomRequest{
		Method: http.MethodPost,
		Route:  "/channels/1/messages",
		Body:   map[string]string{"content": "after-drop"},
	}); err != nil {
		t.Fatalf("expected POST recovery after closed idle HTTP/2 conn: %v", err)
	}

	connsAfterDrop := newConns.Load()
	if connsAfterDrop < 2 {
		t.Fatalf("expected replacement HTTP/2 connection after peer close, got total conns=%d", connsAfterDrop)
	}

	// Phase 3: Sit idle longer than IdleConnTimeout so client proactively closes idle conn and redials cleanly.
	time.Sleep(180 * time.Millisecond)

	if err := client.Request(t.Context(), http.MethodPost, "/channels/2/messages", map[string]string{"content": "after-idle"}, nil); err != nil {
		t.Fatalf("expected request after IdleConnTimeout expiry to redial cleanly: %v", err)
	}

	if newConns.Load() <= connsAfterDrop {
		t.Fatalf("expected new connection after IdleConnTimeout cleanup (before=%d after=%d)", connsAfterDrop, newConns.Load())
	}
}

func TestInternalPoolFallbacksAndTrace(t *testing.T) {
	t.Parallel()

	// Zero-value RestClient exercises defensive nil-pool and disabled-trace fallbacks.
	var empty RestClient
	empty.tracef("noop %d", 1)

	b := empty.acquireBuffer()
	if b == nil {
		t.Fatal("expected fallback bytes.Buffer")
	}
	empty.releaseBuffer(nil)
	empty.releaseBuffer(b)

	mb := empty.acquireMultipartBuffer()
	if mb == nil {
		t.Fatal("expected fallback multipart bytes.Buffer")
	}
	empty.releaseMultipartBuffer(nil)
	empty.releaseMultipartBuffer(mb)

	r := empty.acquireReader([]byte("ok"))
	if r == nil {
		t.Fatal("expected fallback bytes.Reader")
	}
	empty.releaseReader(r)

	cb := empty.acquireCopyBuf()
	if cb == nil || len(*cb) != 32*1024 {
		t.Fatal("expected fallback 32KiB copy buffer")
	}
	empty.releaseCopyBuf(cb)

	if !isASCIIDigits("123456") || isASCIIDigits("") || isASCIIDigits("12a") {
		t.Fatal("unexpected isASCIIDigits result")
	}

	if err := sleepContext(t.Context(), 0); err != nil {
		t.Fatalf("sleepContext(0): %v", err)
	}
}

func TestRateLimiterShardCompaction(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	rl := NewRateLimiter(RateLimiterConfig{
		Shards:        1,
		MaxBuckets:    40,
		MaxRoutes:     40,
		IdleTTL:       10 * time.Second,
		SweepInterval: 1_000_000,
		Now:           func() time.Time { return now },
	})

	hdr := http.Header{}
	hdr.Set("X-RateLimit-Bucket", "compaction-bucket")
	hdr.Set("X-RateLimit-Limit", "5")
	hdr.Set("X-RateLimit-Remaining", "4")
	hdr.Set("X-RateLimit-Reset-After", "1.0")

	ctx := t.Context()

	// Exceed MaxBuckets*2 evictions on the single shard so compactShardMapsLocked triggers.
	for i := range 200 {
		route := "/channels/" + strconv.Itoa(1000+i) + "/messages"
		done, err := rl.Wait(ctx, http.MethodPost, route)
		if err != nil {
			t.Fatalf("Wait(%s): %v", route, err)
		}
		done(hdr)
	}

	sh := &rl.shards[0]
	sh.mu.Lock()
	bucketsLen := len(sh.buckets)
	routesLen := len(sh.routes)
	staleRoutes := 0
	for _, b := range sh.routes {
		if b.evicted {
			staleRoutes++
		}
	}
	sh.mu.Unlock()

	if bucketsLen > 40 || routesLen > 40 {
		t.Fatalf("expected shard maps bounded <= 40, got buckets=%d routes=%d", bucketsLen, routesLen)
	}
	if staleRoutes >= 40 {
		t.Fatalf("expected stale evicted routes to be pruned periodically, got %d", staleRoutes)
	}

	// Advance clock past IdleTTL, add 2 fresh entries, and Sweep: should exercise partial shrink compaction.
	now = now.Add(20 * time.Second)
	for i := range 2 {
		route := "/channels/" + strconv.Itoa(9000+i) + "/messages"
		done, err := rl.Wait(ctx, http.MethodPost, route)
		if err != nil {
			t.Fatalf("Wait(%s): %v", route, err)
		}
		done(hdr)
	}

	if removed := rl.Sweep(0); removed == 0 {
		t.Fatal("expected Sweep to evict expired churn buckets")
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.grown {
		t.Fatalf("expected grown flag to clear after partial shrink compaction (buckets=%d routes=%d)", len(sh.buckets), len(sh.routes))
	}
	if len(sh.buckets) != 2 || len(sh.routes) != 2 {
		t.Fatalf("expected 2 live buckets and routes after partial sweep, got buckets=%d routes=%d", len(sh.buckets), len(sh.routes))
	}
}
