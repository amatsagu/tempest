package discord

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
)

func (msd *ModalInteractionData) UnmarshalJSON(data []byte) error {
	return unmarshalWithDecoder(data, msd.UnmarshalJSONFrom)
}

func (msd *ModalInteractionData) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	isNull, err := consumeJSONObjectStart(dec)
	if err != nil || isNull {
		if isNull {
			*msd = ModalInteractionData{}
		}
		return err
	}

	*msd = ModalInteractionData{}
	for dec.PeekKind() != '}' {
		key, err := readJSONKeyBytes(dec)
		if err != nil {
			return err
		}

		switch string(key) {
		case "custom_id":
			msd.CustomID, err = readJSONString(dec)
		case "components":
			var list componentSlice[ModalComponent]
			if err := list.UnmarshalJSONFrom(dec); err != nil {
				return err
			}
			msd.Components = []ModalComponent(list)
		case "resolved":
			if dec.PeekKind() == 'n' {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				msd.Resolved = nil
				continue
			}
			var resolved InteractionDataResolved
			if err := json.UnmarshalDecode(dec, &resolved); err != nil {
				return err
			}
			msd.Resolved = &resolved
		default:
			err = dec.SkipValue()
		}
		if err != nil {
			return err
		}
	}

	_, err = dec.ReadToken()
	return err
}
