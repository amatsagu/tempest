package tempest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

// TestVerifySignature covers the hand-rolled hex decode: valid, uppercase, wrong
// message, wrong length, non-hex, and a missing key all answer correctly.
func TestVerifySignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	msg := []byte(`1700000000{"id":"1"}`)
	good := hex.EncodeToString(ed25519.Sign(priv, msg))

	if !verifySignature(pub, msg, good) {
		t.Fatal("valid signature rejected")
	}

	if !verifySignature(pub, msg, strings.ToUpper(good)) {
		t.Fatal("uppercase hex rejected")
	}

	if verifySignature(pub, []byte(`1700000000{"id":"2"}`), good) {
		t.Fatal("signature accepted for a different message")
	}

	if verifySignature(pub, msg, good[:len(good)-2]) {
		t.Fatal("short signature accepted")
	}

	if verifySignature(pub, msg, strings.Repeat("g", len(good))) {
		t.Fatal("non-hex signature accepted")
	}

	if verifySignature(nil, msg, good) {
		t.Fatal("signature accepted with no public key")
	}
}
