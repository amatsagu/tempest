package tempest

import "encoding/json"

func (msd *ModalSubmitData) UnmarshalJSON(data []byte) error {
	type alias ModalSubmitData
	var raw struct {
		alias
		Components []json.RawMessage `json:"components"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*msd = ModalSubmitData(raw.alias)
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
