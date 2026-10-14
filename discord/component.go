package discord

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// https://docs.discord.com/developers/components/reference#component-object-component-types
type ComponentType uint8

const (
	ComponentActionRow ComponentType = iota + 1
	ComponentButton
	ComponentStringSelect
	ComponentTextInput
	ComponentUserSelect
	ComponentRoleSelect
	ComponentMentionableSelect
	ComponentChannelSelect
	ComponentSection
	ComponentTextDisplay
	ComponentThumbnail
	ComponentMediaGallery
	ComponentFile
	ComponentSeparator
	_
	_
	ComponentContainer
	ComponentLabel
	ComponentFileUpload
	_
	ComponentRadioGroup
	ComponentCheckboxGroup
	ComponentCheckbox
)

// https://docs.discord.com/developers/components/reference#component-object
type Component interface {
	ComponentType() ComponentType
}

// Components is a list of polymorphic components. It exists because Go
// interfaces cannot be decoded on their own: encoding/json needs a concrete
// type, so this named slice supplies the discriminator-based decoder.
type Components []Component

func (c *Components) UnmarshalJSON(data []byte) error {
	var raw []jsontext.Value
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*c = make(Components, 0, len(raw))
	for _, item := range raw {
		comp, err := UnmarshalComponent(item)
		if err != nil {
			return err
		}
		*c = append(*c, comp)
	}

	return nil
}

// UnmarshalComponent decodes a single component based on its "type" field.
func UnmarshalComponent(data []byte) (Component, error) {
	var header struct {
		Type ComponentType `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}

	switch header.Type {
	case ComponentActionRow:
		var c ActionRow
		return c, json.Unmarshal(data, &c)
	case ComponentSection:
		var c Section
		return c, json.Unmarshal(data, &c)
	case ComponentTextDisplay:
		var c TextDisplay
		return c, json.Unmarshal(data, &c)
	case ComponentThumbnail:
		var c Thumbnail
		return c, json.Unmarshal(data, &c)
	case ComponentMediaGallery:
		var c MediaGallery
		return c, json.Unmarshal(data, &c)
	case ComponentFile:
		var c FileComponent
		return c, json.Unmarshal(data, &c)
	case ComponentSeparator:
		var c Separator
		return c, json.Unmarshal(data, &c)
	case ComponentContainer:
		var c Container
		return c, json.Unmarshal(data, &c)
	case ComponentLabel:
		var c Label
		return c, json.Unmarshal(data, &c)
	}

	return nil, fmt.Errorf("unsupported component type: %d", header.Type)
}
