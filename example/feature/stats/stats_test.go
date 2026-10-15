package stats

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amatsagu/tempest"
)

// TestStatsRendersComponentsV2 drives /stats through the client's HTTP endpoint
// the same way Discord does, and checks the reply is a Components V2 message
// carrying the expected sections.
func TestStatsRendersComponentsV2(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	client, err := tempest.New(tempest.Config{
		PublicKey: hex.EncodeToString(pub),
		HTTPAddr:  "127.0.0.1:0",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cmd := newStatsCommand(client, NewStats())
	if err := client.Command("stats", cmd.handle); err != nil {
		t.Fatalf("Command: %v", err)
	}

	body := `{"id":"1","application_id":"2","type":2,"token":"t","data":{"id":"3","name":"stats","type":1}}`
	ts := "1700000000"
	sig := hex.EncodeToString(ed25519.Sign(priv, append([]byte(ts), body...)))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)

	rec := httptest.NewRecorder()
	client.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	out := rec.Body.String()
	if !strings.Contains(out, `"flags":32768`) {
		t.Errorf("reply does not set MessageFlagIsComponentsV2: %s", out)
	}
	for _, label := range []string{"Uptime", "Interactions since start", "Memory", "Latency", "HTTP mode"} {
		if !strings.Contains(out, label) {
			t.Errorf("reply is missing %q: %s", label, out)
		}
	}
}
