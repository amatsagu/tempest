package tempest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/ratelimit"
	"github.com/amatsagu/tempest/rest"
)

// fileRequest is one request a fileRecorder served, with its Content-Type.
type fileRequest struct {
	method      string
	path        string
	contentType string
	body        string
}

// fileRecorder records every request and answers with a fixed body.
type fileRecorder struct {
	response string
	reqs     []fileRequest
	mu       sync.Mutex
}

func (rec *fileRecorder) serveHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	rec.mu.Lock()
	rec.reqs = append(rec.reqs, fileRequest{
		method:      r.Method,
		path:        r.URL.Path,
		contentType: r.Header.Get("Content-Type"),
		body:        string(body),
	})
	response := rec.response
	rec.mu.Unlock()

	if response == "" {
		response = "{}"
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func (rec *fileRecorder) at(i int) (fileRequest, bool) {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	if i < 0 || i >= len(rec.reqs) {
		return fileRequest{}, false
	}
	return rec.reqs[i], true
}

// newFileRecorder starts a recording REST server and a client pointed at it.
func newFileRecorder(t *testing.T) (*fileRecorder, *rest.Client) {
	t.Helper()

	rec := &fileRecorder{}
	server := httptest.NewServer(http.HandlerFunc(rec.serveHTTP))
	t.Cleanup(server.Close)

	return rec, rest.New("t", rest.Options{
		BaseURL: server.URL,
		Limiter: ratelimit.New(ratelimit.Options{GlobalRate: 1e9}),
	})
}

// TestReplyWithFileDefersThenPatchesMultipart proves an attachment reply first
// acknowledges with a deferred callback, then PATCHes the original as multipart.
func TestReplyWithFileDefersThenPatchesMultipart(t *testing.T) {
	rec, restClient := newFileRecorder(t)
	c := newGatewayClient(t, Config{})
	c.rest = restClient

	done := make(chan error, 1)
	if err := c.Command("upload", func(_ context.Context, itx *discord.CommandInteraction) error {
		err := itx.Reply(discord.ResponseMessageData{Content: "with file"}, true,
			discord.File{Name: "a.txt", ContentType: "text/plain", Data: []byte("hello")})
		done <- err
		return err
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	c.handleGatewayInteraction(c.runCtx, 0, []byte(`{"id":"1","application_id":"2","type":2,"token":"tok","data":{"id":"3","name":"upload","type":1}}`))

	if err := <-done; err != nil {
		t.Fatalf("Reply: %v", err)
	}

	first, ok := rec.at(0)
	if !ok {
		t.Fatal("no requests were recorded")
	}

	if first.method != http.MethodPost || first.path != "/interactions/1/tok/callback" {
		t.Fatalf("first request = %s %s, want POST /interactions/1/tok/callback", first.method, first.path)
	}

	wantType := fmt.Sprintf(`"type":%d`, discord.ResponseDeferredChannelMessageWithSource)
	if !strings.Contains(first.body, wantType) {
		t.Fatalf("callback body = %s, want %s", first.body, wantType)
	}

	wantFlags := fmt.Sprintf(`"flags":%d`, discord.MessageFlagEphemeral)
	if !strings.Contains(first.body, wantFlags) {
		t.Fatalf("callback body = %s, want %s", first.body, wantFlags)
	}

	second, ok := rec.at(1)
	if !ok {
		t.Fatal("only one request was recorded, want the multipart edit")
	}

	if second.method != http.MethodPatch || second.path != "/webhooks/2/tok/messages/@original" {
		t.Fatalf("second request = %s %s, want PATCH /webhooks/2/tok/messages/@original", second.method, second.path)
	}

	if !strings.HasPrefix(second.contentType, "multipart/form-data") {
		t.Fatalf("second Content-Type = %q, want multipart/form-data", second.contentType)
	}

	if !strings.Contains(second.body, "hello") {
		t.Fatalf("second body does not carry the file bytes: %s", second.body)
	}

	if !strings.Contains(second.body, "a.txt") {
		t.Fatalf("second body does not carry the file name: %s", second.body)
	}
}

// TestFollowUpPostsToWebhookAndReturnsMessage proves FollowUp posts to the token
// webhook and decodes the returned message.
func TestFollowUpPostsToWebhookAndReturnsMessage(t *testing.T) {
	rec, restClient := newFileRecorder(t)
	rec.response = `{"id":"9","channel_id":"1","content":"more"}`

	c := newGatewayClient(t, Config{})
	c.rest = restClient

	var got *discord.Message
	done := make(chan error, 1)
	if err := c.Command("more", func(_ context.Context, itx *discord.CommandInteraction) error {
		msg, err := itx.FollowUp(discord.ResponseMessageData{Content: "more"}, false)
		got = msg
		done <- err
		return err
	}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	c.handleGatewayInteraction(c.runCtx, 0, []byte(`{"id":"1","application_id":"2","type":2,"token":"tok","data":{"id":"3","name":"more","type":1}}`))

	if err := <-done; err != nil {
		t.Fatalf("FollowUp: %v", err)
	}

	if got == nil || got.ID.String() != "9" {
		t.Fatalf("FollowUp returned %+v, want message 9", got)
	}

	first, ok := rec.at(0)
	if !ok || first.method != http.MethodPost || first.path != "/webhooks/2/tok" {
		t.Fatalf("first request = %+v, want POST /webhooks/2/tok", first)
	}
}
