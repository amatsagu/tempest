package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/amatsagu/tempest/rest"
)

// TestRequestHeaders proves the response headers are reachable, which is what a
// header-driven flow such as pagination needs.
func TestRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://example.test/?after=9>; rel="next"`)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	body, header, err := client.RequestHeaders(context.Background(), http.MethodGet, "/x", nil, "")
	if err != nil {
		t.Fatalf("RequestHeaders: %v", err)
	}

	if string(body) != `[]` {
		t.Fatalf("body = %s", body)
	}

	if got := header.Get("Link"); got != `<https://example.test/?after=9>; rel="next"` {
		t.Fatalf("Link = %q", got)
	}
}

// TestRequestWithFilesRejectsOversize proves an oversized attachment is refused
// before the multipart body is built.
func TestRequestWithFilesRejectsOversize(t *testing.T) {
	client := rest.New("t", rest.Options{BaseURL: "http://127.0.0.1:1"})

	_, err := client.RequestWithFiles(context.Background(), http.MethodPost, "/x", nil,
		[]rest.File{{Name: "big.bin", Data: make([]byte, rest.MaxFileBytes+1)}}, "")
	if err == nil {
		t.Fatal("expected an error for an oversized file")
	}

	if !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want a limit message", err)
	}
}

// TestServerErrorNotRetriedForPost proves a 5xx on a non-idempotent request is
// returned instead of replayed, since it may already have been processed.
func TestServerErrorNotRetriedForPost(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	if _, err := client.Request(context.Background(), http.MethodPost, "/x", nil, ""); err == nil {
		t.Fatal("expected an error")
	}

	if attempts.Load() != 1 {
		t.Fatalf("attempts = %d, want 1 (POST is not replayed)", attempts.Load())
	}
}

// TestServerErrorRetriedForGet proves an idempotent request is still retried.
func TestServerErrorRetriedForGet(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	body, err := client.Request(context.Background(), http.MethodGet, "/x", nil, "")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	if string(body) != `{"ok":true}` || attempts.Load() != 2 {
		t.Fatalf("body = %s, attempts = %d; want ok after 2 attempts", body, attempts.Load())
	}
}
