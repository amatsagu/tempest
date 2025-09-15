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

		if cmp, ok := parsed.(LayoutComponent); ok {
			msg.Components = append(msg.Components, cmp)
		}
	}

	raw.Components = nil
	return nil
}
