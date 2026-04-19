package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyRequest(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
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

		resBody, verified := verifyRequest(req, pub, 1024)
		if !verified {
			t.Error("expected verification to succeed")
		}
		if !bytes.Equal(resBody, body) {
			t.Errorf("expected body %s, got %s", body, resBody)
		}
	})

	t.Run("Invalid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", "invalid")
		req.Header.Set("X-Signature-Timestamp", timestamp)

		_, verified := verifyRequest(req, pub, 1024)
		if verified {
			t.Error("expected verification to fail")
		}
	})

	t.Run("Body too large", func(t *testing.T) {
		largeBody := make([]byte, 2048)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(largeBody))
		req.Header.Set("X-Signature-Ed25519", sigHex)
		req.Header.Set("X-Signature-Timestamp", timestamp)

		_, verified := verifyRequest(req, pub, 1024)
		if verified {
			// It should fail because ed25519.Verify will check truncated body against signature of full body
			t.Error("expected verification to fail due to truncated body")
		}
	})
}
