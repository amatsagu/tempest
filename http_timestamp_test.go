package tempest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestTimestampWindowRejectsReplay proves the optional freshness window rejects a
// stale signature and still accepts a fresh one.
func TestTimestampWindowRejectsReplay(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	client, err := New(Config{
		HTTPAddr:        "127.0.0.1:0",
		PublicKey:       hex.EncodeToString(pub),
		TimestampWindow: time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	body := `{"id":"1","application_id":"2","type":1,"token":"t"}`

	send := func(timestamp string) int {
		sig := ed25519.Sign(priv, append([]byte(timestamp), body...))

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
		req.Header.Set("X-Signature-Timestamp", timestamp)

		rec := httptest.NewRecorder()
		client.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := send("1700000000"); code != http.StatusUnauthorized {
		t.Fatalf("stale timestamp status = %d, want 401", code)
	}

	fresh := strconv.FormatInt(time.Now().Unix(), 10)
	if code := send(fresh); code != http.StatusOK {
		t.Fatalf("fresh timestamp status = %d, want 200", code)
	}
}
