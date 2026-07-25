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

		// TODO: This should always implement ModalComponent, as Discord silently rejects payloads with invalid components;
		// we should arguably return an error of some sort here
		if cmp, ok := parsed.(ModalComponent); ok {
			msd.Components = append(msd.Components, cmp)
		}
	}

	raw.Components = nil
	return nil
}
