package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"io"
	"net/http"
)

// Verifies incoming request if it's from Discord. Returns the body bytes if verification was successful.
func verifyRequest(r *http.Request, key ed25519.PublicKey, maxSize int64) ([]byte, bool) {
	signature := r.Header.Get("X-Signature-Ed25519")
	if signature == "" {
		return nil, false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, false
	}

	if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
		return nil, false
	}

	timestamp := r.Header.Get("X-Signature-Timestamp")
	if timestamp == "" {
		return nil, false
	}

	var msg bytes.Buffer
	msg.WriteString(timestamp)

	defer r.Body.Close()
	var body bytes.Buffer

	_, err = io.Copy(&msg, io.TeeReader(io.LimitReader(r.Body, maxSize), &body))
	if err != nil {
		return nil, false
	}

	if ed25519.Verify(key, msg.Bytes(), sig) {
		return body.Bytes(), true
	}

	return nil, false
}

func extractUserIDFromToken(token string) (Snowflake, error) {
	strs := strings.Split(token, ".")
	if len(strs) == 0 {
		return 0, errors.New("token is not in a valid format")
	}

	hexID := strings.Replace(strs[0], "Bot ", "", 1)

	byteID, err := base64.RawStdEncoding.DecodeString(hexID)
	if err != nil {
		return 0, err
	}

	return StringToSnowflake(string(byteID))
}
