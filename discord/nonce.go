package discord

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"unicode/utf8"
)

// Nonce is a message nonce. Discord sends it as either a string or an integer,
// so it is normalized to a string when decoding.
//
// https://docs.discord.com/developers/resources/message#message-object
type Nonce string

func (n *Nonce) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	switch dec.PeekKind() {
	case 'n':
		*n = ""
		return dec.SkipValue()
	case '"':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*n = Nonce(tok.String())
		return nil
	default:
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}
		*n = Nonce(raw)
		return nil
	}
}

func (n *Nonce) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, nullLiteral) {
		*n = ""
		return nil
	}

	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		inner := data[1 : len(data)-1]
		if isSimpleUnescapedJSONString(inner) {
			*n = Nonce(inner)
			return nil
		}
	}

	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}

		*n = Nonce(value)
		return nil
	}

	*n = Nonce(data)
	return nil
}

func isSimpleUnescapedJSONString(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, c := range b {
		if c == '"' || c == '\\' || c < 0x20 {
			return false
		}
	}
	return true
}
