package rest_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amatsagu/tempest/ratelimit"
	"github.com/amatsagu/tempest/rest"
)

func TestRequestSendsAuthHeader(t *testing.T) {
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := rest.New("secret-token", rest.Options{BaseURL: server.URL})

	body, err := client.Request(context.Background(), http.MethodGet, "/users/@me", nil, "")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	if string(body) != `{"ok":true}` {
		t.Errorf("body = %s", body)
	}

	if auth != "Bot secret-token" {
		t.Errorf("Authorization = %q, want Bot secret-token", auth)
	}
}

func TestRequestRetriesTooManyRequests(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL, MaxRetries: 2})

	body, err := client.Request(context.Background(), http.MethodGet, "/x", nil, "")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	if attempts.Load() != 2 {
		t.Errorf("attempts = %d, want 2", attempts.Load())
	}

	if string(body) != `{"ok":true}` {
		t.Errorf("body = %s", body)
	}
}

func TestRequestReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Missing Permissions"}`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	_, err := client.Request(context.Background(), http.MethodGet, "/x", nil, "")
	var httpErr *rest.HTTPError
	if err == nil {
		t.Fatal("expected error")
	}

	if !asHTTPError(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
		t.Fatalf("err = %v, want 403 HTTPError", err)
	}
}

// TestInteractionCallbackSkipsGlobalPause proves a callback is not held behind a
// paused global bucket while an ordinary route still is.
func TestInteractionCallbackSkipsGlobalPause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	limiter := ratelimit.New(ratelimit.Options{GlobalRate: 1e9})
	limiter.Pause(time.Hour)

	client := rest.New("t", rest.Options{BaseURL: server.URL, Limiter: limiter})

	start := time.Now()
	if _, err := client.Request(context.Background(), http.MethodPost, "/interactions/1/tok/callback", map[string]any{"type": 7}, ""); err != nil {
		t.Fatalf("Request: %v", err)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("interaction callback was delayed by the global pause: %v", elapsed)
	}
}

func TestRequestIntoDecodesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"42","name":"tempest"}`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := client.RequestInto(context.Background(), http.MethodGet, "/x", nil, "", &out); err != nil {
		t.Fatalf("RequestInto: %v", err)
	}

	if out.ID != "42" || out.Name != "tempest" {
		t.Fatalf("decoded = %+v", out)
	}
}

func TestRequestIntoDiscardsNilTarget(t *testing.T) {
	var hit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})
	if err := client.RequestInto(context.Background(), http.MethodDelete, "/x", nil, "", nil); err != nil {
		t.Fatalf("RequestInto: %v", err)
	}

	if !hit {
		t.Fatal("server was not reached")
	}
}

func TestRequestIntoReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Unknown Resource"}`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	var out map[string]any
	err := client.RequestInto(context.Background(), http.MethodGet, "/x", nil, "", &out)
	var httpErr *rest.HTTPError
	if !asHTTPError(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v, want 404 HTTPError", err)
	}

	if string(httpErr.Body) != `{"message":"Unknown Resource"}` {
		t.Fatalf("error body = %s", httpErr.Body)
	}
}

// TestRequestWithFilesBuildsMultipart checks the payload_json field and files[0] part.
func TestRequestWithFilesBuildsMultipart(t *testing.T) {
	type capture struct {
		contentType string
		payload     string
		disposition string
		fileName    string
		fileData    []byte
	}

	captured := make(chan capture, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := capture{contentType: r.Header.Get("Content-Type")}

		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			data, _ := io.ReadAll(part)
			switch part.FormName() {
			case "payload_json":
				got.payload = string(data)
			case "files[0]":
				got.disposition = part.Header.Get("Content-Disposition")
				got.fileName = part.FileName()
				got.fileData = data
			}
		}

		captured <- got
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := rest.New("t", rest.Options{BaseURL: server.URL})

	payload := map[string]any{"content": "hello"}
	files := []rest.File{{Name: "rotated.png", ContentType: "image/png", Data: []byte{0x89, 0x50, 0x4e, 0x47}}}

	if _, err := client.RequestWithFiles(context.Background(), http.MethodPost, "/x", payload, files, ""); err != nil {
		t.Fatalf("RequestWithFiles: %v", err)
	}

	got := <-captured

	if !strings.HasPrefix(got.contentType, "multipart/form-data") {
		t.Errorf("Content-Type = %q, want multipart/form-data", got.contentType)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got.payload), &decoded); err != nil {
		t.Fatalf("payload_json %q is not valid JSON: %v", got.payload, err)
	}

	if decoded["content"] != "hello" {
		t.Errorf("payload_json content = %v, want hello", decoded["content"])
	}

	if !strings.Contains(got.disposition, `name="files[0]"`) || !strings.Contains(got.disposition, `filename="rotated.png"`) {
		t.Errorf("files[0] Content-Disposition = %q, want a quoted name and filename", got.disposition)
	}

	if got.fileName != "rotated.png" {
		t.Errorf("files[0] filename = %q, want rotated.png", got.fileName)
	}

	if string(got.fileData) != string(files[0].Data) {
		t.Errorf("files[0] data = %v, want %v", got.fileData, files[0].Data)
	}
}

func asHTTPError(err error, target **rest.HTTPError) bool {
	he, ok := err.(*rest.HTTPError)
	if ok {
		*target = he
	}
	return ok
}
