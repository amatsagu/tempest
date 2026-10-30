package tempest

import (
	"bytes"
	"encoding/json"
)

// Nonce is a message nonce. Discord sends it as either a string or an integer,
// so it is normalized to a string when decoding.
//
// https://docs.discord.com/developers/resources/message#message-object
type Nonce string

func (n *Nonce) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*n = ""
		return nil
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
