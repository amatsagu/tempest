package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

// BenchmarkServeHTTPCommand measures the whole HTTP endpoint for one command: body
// read, in-place Ed25519 verification, a single type probe, dispatch, and the JSON
// write.
func BenchmarkServeHTTPCommand(b *testing.B) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("generate key: %v", err)
	}

	client, err := New(Config{HTTPAddr: "127.0.0.1:0", PublicKey: hex.EncodeToString(pub)})
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	err = client.Command("ping", func(_ context.Context, itx *discord.CommandInteraction) error {
		return itx.Reply(discord.ResponseMessageData{Content: "pong"}, false)
	})
	if err != nil {
		b.Fatalf("Command: %v", err)
	}

	body := []byte(`{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`)
	ts := "1700000000"
	sig := hex.EncodeToString(ed25519.Sign(priv, append([]byte(ts), body...)))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", sig)
		req.Header.Set("X-Signature-Timestamp", ts)

		client.ServeHTTP(httptest.NewRecorder(), req)
	}
}

// BenchmarkServeHTTPPing is the same path with no handler work, so the difference
// against BenchmarkServeHTTPCommand attributes the allocation to dispatch.
func BenchmarkServeHTTPPing(b *testing.B) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("generate key: %v", err)
	}

	client, err := New(Config{HTTPAddr: "127.0.0.1:0", PublicKey: hex.EncodeToString(pub)})
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	body := []byte(`{"id":"1","application_id":"2","type":1,"token":"t"}`)
	ts := "1700000000"
	sig := hex.EncodeToString(ed25519.Sign(priv, append([]byte(ts), body...)))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", sig)
		req.Header.Set("X-Signature-Timestamp", ts)

		client.ServeHTTP(httptest.NewRecorder(), req)
	}
}

// BenchmarkVerifySignature measures the Ed25519 check alone, which dominates the
// endpoint cost.
func BenchmarkVerifySignature(b *testing.B) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("generate key: %v", err)
	}

	msg := []byte(`1700000000{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"ping","type":1}}`)
	sig := hex.EncodeToString(ed25519.Sign(priv, msg))

	b.ReportAllocs()

	for b.Loop() {
		if !verifySignature(pub, msg, sig) {
			b.Fatal("signature did not verify")
		}
	}
}
