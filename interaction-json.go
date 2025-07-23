package tempest

import "encoding/json"

func (msd *ModalInteractionData) UnmarshalJSON(data []byte) error {
	type alias ModalInteractionData
	var raw struct {
		alias
		Components []json.RawMessage `json:"components"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*msd = ModalInteractionData(raw.alias)
	for _, comp := range raw.Components {
		parsed, err := UnmarshalComponent(comp)
		if err != nil {
			return err
		}

		if cmp, ok := parsed.(LayoutComponent); ok {
			msd.Components = append(msd.Components, cmp)
		}
	}

	raw.Components = nil
	return nil
}
