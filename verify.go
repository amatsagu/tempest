package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"strings"
)

var errBadPublicKey = errors.New("tempest: public key must be a 32-byte hex-encoded Ed25519 key")

func parsePublicKey(hexKey string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil {
		return nil, errBadPublicKey
	}

	if len(raw) != ed25519.PublicKeySize {
		return nil, errBadPublicKey
	}
	return ed25519.PublicKey(raw), nil
}

// Checks Discord's Ed25519 signature over an already-assembled
// timestamp||body. The signature is decoded into a stack array and the canonical
// top-three-bits check runs first, so neither path allocates and clearly invalid
// input skips the curve work.
func verifySignature(key ed25519.PublicKey, signed []byte, signatureHex string) bool {
	if len(key) != ed25519.PublicKeySize || len(signatureHex) != ed25519.SignatureSize*2 {
		return false
	}

	var sig [ed25519.SignatureSize]byte
	if !decodeHex(sig[:], signatureHex) {
		return false
	}

	if sig[ed25519.SignatureSize-1]&224 != 0 {
		return false
	}

	return ed25519.Verify(key, signed, sig[:])
}

// Decodes src into dst, which must be half of src's length. It exists
// because hex.DecodeString allocates, and signature checking is the hottest path
// on the HTTP endpoint.
func decodeHex(dst []byte, src string) bool {
	if len(src) != len(dst)*2 {
		return false
	}

	for i := range dst {
		high, ok := hexNibble(src[i*2])
		if !ok {
			return false
		}

		low, ok := hexNibble(src[i*2+1])
		if !ok {
			return false
		}
		dst[i] = high<<4 | low
	}

	return true
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}
