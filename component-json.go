package tempest

import (
	"encoding/json"
	"fmt"
)

// UnmarshalComponent inspects the "type" field and returns the proper component struct.
func UnmarshalComponent(data []byte) (AnyComponent, error) {
	var partial struct {
		Type ComponentType `json:"type"`
	}

	if err := json.Unmarshal(data, &partial); err != nil {
		return nil, err
	}

	switch partial.Type {
	case BUTTON_COMPONENT_TYPE:
		var cmp ButtonComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case STRING_SELECT_COMPONENT_TYPE:
		var cmp StringSelectComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case TEXT_INPUT_COMPONENT_TYPE:
		var cmp TextInputComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case USER_SELECT_COMPONENT_TYPE,
		ROLE_SELECT_COMPONENT_TYPE,
		MENTIONABLE_SELECT_COMPONENT_TYPE,
		CHANNEL_SELECT_COMPONENT_TYPE:
		var cmp SelectComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case SECTION_COMPONENT_TYPE:
		var cmp SectionComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case TEXT_DISPLAY_COMPONENT_TYPE:
		var cmp TextDisplayComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case THUMBNAIL_COMPONENT_TYPE:
		var cmp ThumbnailComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case MEDIA_GALLERY_COMPONENT_TYPE:
		var cmp MediaGalleryComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case FILE_COMPONENT_TYPE:
		var cmp FileComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case SEPARATOR_COMPONENT_TYPE:
		var cmp SeparatorComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case CONTAINER_COMPONENT_TYPE:
		var cmp ContainerComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case ACTION_ROW_COMPONENT_TYPE:
		var cmp ActionRowComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	case LABEL_COMPONENT_TYPE:
		var cmp LabelComponent
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		return cmp, nil
	}

	return nil, fmt.Errorf("unknown component type: %d", partial.Type)
}

func (c *LabelComponent) UnmarshalJSON(data []byte) error {
	type alias LabelComponent
	var raw struct {
		alias
		Component json.RawMessage `json:"component"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*c = LabelComponent(raw.alias)
	parsed, err := UnmarshalComponent(raw.Component)
	if err != nil {
		return err
	}

	if cmp, ok := parsed.(LabelChildComponent); ok {
		c.Component = cmp
	}
	raw.Component = nil
	return nil
}

func (c *ActionRowComponent) UnmarshalJSON(data []byte) error {
	type alias ActionRowComponent
	var raw struct {
		alias
		Components []json.RawMessage `json:"components"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*c = ActionRowComponent(raw.alias)
	for _, comp := range raw.Components {
		parsed, err := UnmarshalComponent(comp)
		if err != nil {
			return err
		}

		if cmp, ok := parsed.(ActionRowChildComponent); ok {
			c.Components = append(c.Components, cmp)
		}
	}

	raw.Components = nil
	return nil
}

func (c *ContainerComponent) UnmarshalJSON(data []byte) error {
	type alias ContainerComponent
	var raw struct {
		alias
		Components []json.RawMessage `json:"components"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*c = ContainerComponent(raw.alias)
	for _, comp := range raw.Components {
		parsed, err := UnmarshalComponent(comp)
		if err != nil {
			return err
		}

		if cmp, ok := parsed.(ContainerChildComponent); ok {
			c.Components = append(c.Components, cmp)
		}
	}

	raw.Components = nil
	return nil
}

func (c *SectionComponent) UnmarshalJSON(data []byte) error {
	type alias SectionComponent
	var raw struct {
		alias
		Components []json.RawMessage `json:"components"`
		Accessory  *json.RawMessage  `json:"accessory,omitempty"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*c = SectionComponent(raw.alias)
	for _, comp := range raw.Components {
		parsed, err := UnmarshalComponent(comp)
		if err != nil {
			return err
		}

		if cmp, ok := parsed.(TextDisplayComponent); ok {
			c.Components = append(c.Components, cmp)
		}
	}
	if raw.Accessory != nil {
		parsed, err := UnmarshalComponent(*raw.Accessory)
		if err != nil {
			return err
		}

		if cmp, ok := parsed.(AccessoryComponent); ok {
			c.Accessory = cmp
		}
	}

	raw.Components = nil
	raw.Accessory = nil
	return nil
}
