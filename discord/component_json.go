package discord

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
)

// Inspects the "type" field and returns the proper component struct.
func UnmarshalComponent(data []byte) (AnyComponent, error) {
	compType, err := peekComponentType(data)
	if err != nil {
		return nil, err
	}

	switch compType {
	case ButtonComponentType:
		var cmp ButtonComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case StringSelectComponentType:
		var cmp StringSelectComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case TextInputComponentType:
		var cmp TextInputComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case UserSelectComponentType,
		RoleSelectComponentType,
		MentionableSelectComponentType,
		ChannelSelectComponentType:
		var cmp SelectComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case SectionComponentType:
		var cmp SectionComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case TextDisplayComponentType:
		var cmp TextDisplayComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case ThumbnailComponentType:
		var cmp ThumbnailComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case MediaGalleryComponentType:
		var cmp MediaGalleryComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case FileComponentType:
		var cmp FileComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case SeparatorComponentType:
		var cmp SeparatorComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case ContainerComponentType:
		var cmp ContainerComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case ActionRowComponentType:
		var cmp ActionRowComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case LabelComponentType:
		var cmp LabelComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case FileUploadComponentType:
		var cmp FileUploadComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case RadioGroupComponentType:
		var cmp RadioGroupComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case CheckboxGroupComponentType:
		var cmp CheckboxGroupComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case CheckboxComponentType:
		var cmp CheckboxComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	}

	return nil, fmt.Errorf("unknown component type: %d", compType)
}

type componentSlice[T AnyComponent] []T

func (s *componentSlice[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' {
		*s = nil
		return dec.SkipValue()
	}

	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '[' {
		return errInvalidJSONArray
	}

	dst := (*s)[:0]
	for dec.PeekKind() != ']' {
		if cap(dst) == 0 {
			dst = make([]T, 0, 4)
		}

		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}

		parsed, err := UnmarshalComponent(raw)
		if err != nil {
			return err
		}

		if cmp, ok := parsed.(T); ok {
			dst = append(dst, cmp)
		}
	}

	_, err = dec.ReadToken()
	*s = dst
	return err
}

func (c *LabelComponent) UnmarshalJSON(data []byte) error {
	return unmarshalWithDecoder(data, c.UnmarshalJSONFrom)
}

func (c *LabelComponent) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	isNull, err := consumeJSONObjectStart(dec)
	if err != nil || isNull {
		if isNull {
			*c = LabelComponent{}
		}
		return err
	}

	*c = LabelComponent{}
	for dec.PeekKind() != '}' {
		key, err := readJSONKeyBytes(dec)
		if err != nil {
			return err
		}

		switch string(key) {
		case "component":
			if dec.PeekKind() == 'n' {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				c.Component = nil
				continue
			}
			var raw jsontext.Value
			raw, err = dec.ReadValue()
			if err != nil {
				return err
			}
			var parsed AnyComponent
			parsed, err = UnmarshalComponent(raw)
			if err != nil {
				return err
			}
			if cmp, ok := parsed.(LabelChildComponent); ok {
				c.Component = cmp
			}
		case "label":
			c.Label, err = readJSONString(dec)
		case "description":
			c.Description, err = readJSONString(dec)
		case "id":
			c.ID, err = readJSONUint32(dec)
		case "type":
			c.Type, err = readJSONComponentType(dec)
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

func (c *ActionRowComponent) UnmarshalJSON(data []byte) error {
	return unmarshalWithDecoder(data, c.UnmarshalJSONFrom)
}

func (c *ActionRowComponent) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	isNull, err := consumeJSONObjectStart(dec)
	if err != nil || isNull {
		if isNull {
			*c = ActionRowComponent{}
		}
		return err
	}

	*c = ActionRowComponent{}
	for dec.PeekKind() != '}' {
		key, err := readJSONKeyBytes(dec)
		if err != nil {
			return err
		}

		switch string(key) {
		case "components":
			var list componentSlice[ActionRowChildComponent]
			if err := list.UnmarshalJSONFrom(dec); err != nil {
				return err
			}
			c.Components = []ActionRowChildComponent(list)
		case "id":
			c.ID, err = readJSONUint32(dec)
		case "type":
			c.Type, err = readJSONComponentType(dec)
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

func (c *ContainerComponent) UnmarshalJSON(data []byte) error {
	return unmarshalWithDecoder(data, c.UnmarshalJSONFrom)
}

func (c *ContainerComponent) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	isNull, err := consumeJSONObjectStart(dec)
	if err != nil || isNull {
		if isNull {
			*c = ContainerComponent{}
		}
		return err
	}

	*c = ContainerComponent{}
	for dec.PeekKind() != '}' {
		key, err := readJSONKeyBytes(dec)
		if err != nil {
			return err
		}

		switch string(key) {
		case "components":
			var list componentSlice[ContainerChildComponent]
			if err := list.UnmarshalJSONFrom(dec); err != nil {
				return err
			}
			c.Components = []ContainerChildComponent(list)
		case "id":
			c.ID, err = readJSONUint32(dec)
		case "accent_color":
			c.AccentColor, err = readJSONUint32(dec)
		case "type":
			c.Type, err = readJSONComponentType(dec)
		case "spoiler":
			c.Spoiler, err = readJSONBool(dec)
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

func (c *SectionComponent) UnmarshalJSON(data []byte) error {
	return unmarshalWithDecoder(data, c.UnmarshalJSONFrom)
}

func (c *SectionComponent) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	isNull, err := consumeJSONObjectStart(dec)
	if err != nil || isNull {
		if isNull {
			*c = SectionComponent{}
		}
		return err
	}

	*c = SectionComponent{}
	for dec.PeekKind() != '}' {
		key, err := readJSONKeyBytes(dec)
		if err != nil {
			return err
		}

		switch string(key) {
		case "components":
			if dec.PeekKind() == 'n' {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				c.Components = nil
				continue
			}
			var arrTok jsontext.Token
			arrTok, err = dec.ReadToken()
			if err != nil {
				return err
			}
			if arrTok.Kind() != '[' {
				return errInvalidJSONArray
			}
			for dec.PeekKind() != ']' {
				if cap(c.Components) == 0 {
					c.Components = make([]TextDisplayComponent, 0, 3)
				}
				var raw jsontext.Value
				raw, err = dec.ReadValue()
				if err != nil {
					return err
				}
				var compType ComponentType
				compType, err = peekComponentType(raw)
				if err != nil {
					return err
				}
				if compType == TextDisplayComponentType {
					var cmp TextDisplayComponent
					if err := json.Unmarshal(raw, &cmp); err != nil {
						return err
					}
					c.Components = append(c.Components, cmp)
					continue
				}
				if _, err := UnmarshalComponent(raw); err != nil {
					return err
				}
			}
			_, err = dec.ReadToken()
		case "accessory":
			if dec.PeekKind() == 'n' {
				if err := dec.SkipValue(); err != nil {
					return err
				}
				c.Accessory = nil
				continue
			}
			var raw jsontext.Value
			raw, err = dec.ReadValue()
			if err != nil {
				return err
			}
			var parsed AnyComponent
			parsed, err = UnmarshalComponent(raw)
			if err != nil {
				return err
			}
			if cmp, ok := parsed.(AccessoryComponent); ok {
				c.Accessory = cmp
			}
		case "id":
			c.ID, err = readJSONUint32(dec)
		case "type":
			c.Type, err = readJSONComponentType(dec)
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
