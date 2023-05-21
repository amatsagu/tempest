package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"

	"io"
	"net/http"

	"github.com/sugawarayuuta/sonnet"
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

func terminateCommandInteraction(w http.ResponseWriter) {
	body, err := sonnet.Marshal(ResponseMessage{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE,
		Data: &ResponseMessageData{
			Content: "Oh snap! It looks like you tried to trigger (/) command which is not registered within local cache. Please report this bug to my master.",
			Flags:   64,
		},
	})

	if err != nil {
		panic("failed to parse json payload")
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(body)
}
