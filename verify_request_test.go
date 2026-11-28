package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestVerifyRequest(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &Client{
		bufferPool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		readerPool: &sync.Pool{
			New: func() any {
				return new(io.LimitedReader)
			},
		},
	}

	timestamp := "1234567890"
	body := []byte(`{"type":1}`)
	msg := append([]byte(timestamp), body...)
	sig := ed25519.Sign(priv, msg)
	sigHex := hex.EncodeToString(sig)

	t.Run("Valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", sigHex)
		req.Header.Set("X-Signature-Timestamp", timestamp)

		resBody, buf, verified := client.verifyRequest(req, pub, 1024)
		if !verified {
			t.Fatal("expected verification to succeed")
		}
		if !bytes.Equal(resBody, body) {
			t.Fatalf("expected body %s, got %s", body, resBody)
		}
		client.releaseBuffer(buf)
	})

	t.Run("Invalid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", "invalid")
		req.Header.Set("X-Signature-Timestamp", timestamp)

		_, _, verified := client.verifyRequest(req, pub, 1024)
		if verified {
			t.Fatal("expected verification to fail")
		}
	})

	t.Run("Malformed hex or missing timestamp", func(t *testing.T) {
		badSig := make([]byte, ed25519.SignatureSize)
		badHex := hex.EncodeToString(badSig)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", badHex)
		if _, _, ok := client.verifyRequest(req, pub, 1024); ok {
			t.Fatal("expected failure without timestamp")
		}

		req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req2.Header.Set("X-Signature-Ed25519", "zz"+badHex[2:])
		req2.Header.Set("X-Signature-Timestamp", timestamp)
		if _, _, ok := client.verifyRequest(req2, pub, 1024); ok {
			t.Fatal("expected failure for non-hex signature")
		}
	})

	t.Run("Body too large", func(t *testing.T) {
		largeBody := make([]byte, 2048)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(largeBody))
		req.Header.Set("X-Signature-Ed25519", sigHex)
		req.Header.Set("X-Signature-Timestamp", timestamp)

		_, _, verified := client.verifyRequest(req, pub, 1024)
		if verified {
			t.Fatal("expected verification to fail due to truncated body")
		}
	})
}
