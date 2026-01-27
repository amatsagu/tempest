package tempest

import (
	"encoding/json"
)

func (msg *Message) UnmarshalJSON(data []byte) error {
	type alias Message
	var raw struct {
		alias
		Components []json.RawMessage `json:"components"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*msg = Message(raw.alias)
	for _, comp := range raw.Components {
		parsed, err := UnmarshalComponent(comp)
		if err != nil {
			return err
		}

		// TODO: This should always implement MessageComponent, as Discord silently rejects payloads with invalid components;
		// we should arguably return an error of some sort here
		if cmp, ok := parsed.(MessageComponent); ok {
			msg.Components = append(msg.Components, cmp)
		}
	}

	raw.Components = nil
	return nil
}
