package ashara

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

// Verifies incoming request if it's from Discord.
func verifyRequest(r *http.Request, key ed25519.PublicKey) bool {
	var msg bytes.Buffer

	signature := r.Header.Get("X-Signature-Ed25519")
	if signature == "" {
		return false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
		return false
	}

	timestamp := r.Header.Get("X-Signature-Timestamp")
	if timestamp == "" {
		return false
	}

	msg.WriteString(timestamp)

	defer r.Body.Close()
	var body bytes.Buffer

	// Copy the original body back into the request after finishing.
	defer func() {
		r.Body = io.NopCloser(&body)
	}()

	// Copy body into buffers
	_, err = io.Copy(&msg, io.TeeReader(r.Body, &body))
	if err != nil {
		return false
	}

	return ed25519.Verify(key, msg.Bytes(), sig)
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
