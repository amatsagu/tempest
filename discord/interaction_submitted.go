package discord

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-component-interaction-response-structures
type SubmittedComponent interface {
	ComponentType() ComponentType
	ComponentID() uint32
}

// https://docs.discord.com/developers/components/reference#action-row-action-row-structure
type SubmittedActionRow struct {
	Components []SubmittedComponent `json:"components"`
	ID         uint32               `json:"id,omitzero"`
	Type       ComponentType        `json:"type"`
}

func (s SubmittedActionRow) ComponentType() ComponentType { return ComponentActionRow }
func (s SubmittedActionRow) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#label-label-interaction-response-structure
type SubmittedLabel struct {
	Component SubmittedComponent `json:"component"`
	ID        uint32             `json:"id,omitzero"`
	Type      ComponentType      `json:"type"`
}

func (s SubmittedLabel) ComponentType() ComponentType { return ComponentLabel }
func (s SubmittedLabel) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#text-input-text-input-interaction-response-structure
type SubmittedTextInput struct {
	CustomID string        `json:"custom_id"`
	Value    string        `json:"value"`
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"`
}

func (s SubmittedTextInput) ComponentType() ComponentType { return ComponentTextInput }
func (s SubmittedTextInput) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#string-select-string-select-interaction-response-structure
type SubmittedStringSelect struct {
	CustomID         string        `json:"custom_id"`
	Values           []string      `json:"values"`
	ID               uint32        `json:"id,omitzero"`
	Type             ComponentType `json:"type,omitzero"`
	RawComponentType ComponentType `json:"component_type,omitzero"`
}

func (s SubmittedStringSelect) ComponentType() ComponentType { return ComponentStringSelect }
func (s SubmittedStringSelect) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#user-select-user-select-interaction-response-structure
type SubmittedSelect struct {
	CustomID         string        `json:"custom_id"`
	Values           []Snowflake   `json:"values"`
	ID               uint32        `json:"id,omitzero"`
	Type             ComponentType `json:"type,omitzero"`
	RawComponentType ComponentType `json:"component_type,omitzero"`
}

func (s SubmittedSelect) ComponentType() ComponentType {
	if s.Type != 0 {
		return s.Type
	}
	return s.RawComponentType
}
func (s SubmittedSelect) ComponentID() uint32 { return s.ID }

// https://docs.discord.com/developers/components/reference#radio-group-interaction-response-structure
type SubmittedRadioGroup struct {
	CustomID string        `json:"custom_id"`
	Value    string        `json:"value"`
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"`
}

func (s SubmittedRadioGroup) ComponentType() ComponentType { return ComponentRadioGroup }
func (s SubmittedRadioGroup) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#checkbox-group-interaction-response-structure
type SubmittedCheckboxGroup struct {
	CustomID string        `json:"custom_id"`
	Values   []string      `json:"values"`
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"`
}

func (s SubmittedCheckboxGroup) ComponentType() ComponentType { return ComponentCheckboxGroup }
func (s SubmittedCheckboxGroup) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#checkbox-interaction-response-structure
type SubmittedCheckbox struct {
	CustomID string        `json:"custom_id"`
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"`
	Value    bool          `json:"value"`
}

func (s SubmittedCheckbox) ComponentType() ComponentType { return ComponentCheckbox }
func (s SubmittedCheckbox) ComponentID() uint32          { return s.ID }

// https://docs.discord.com/developers/components/reference#file-upload-file-upload-interaction-response-structure
type SubmittedFileUpload struct {
	CustomID string         `json:"custom_id"`
	Values   []AttachmentID `json:"values"`
	ID       uint32         `json:"id,omitzero"`
	Type     ComponentType  `json:"type"`
}

func (s SubmittedFileUpload) ComponentType() ComponentType { return ComponentFileUpload }
func (s SubmittedFileUpload) ComponentID() uint32          { return s.ID }

func UnmarshalSubmittedComponent(data []byte) (SubmittedComponent, error) {
	var header struct {
		Type          ComponentType `json:"type"`
		ComponentType ComponentType `json:"component_type"`
	}

	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}

	typ := header.Type
	if typ == 0 {
		typ = header.ComponentType
	}

	switch typ {
	case ComponentActionRow:
		var raw struct {
			Components []jsontext.Value `json:"components"`
			ID         uint32           `json:"id"`
			Type       ComponentType    `json:"type"`
		}

		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		row := SubmittedActionRow{
			Type:       raw.Type,
			ID:         raw.ID,
			Components: make([]SubmittedComponent, 0, len(raw.Components)),
		}

		for _, child := range raw.Components {
			c, err := UnmarshalSubmittedComponent(child)
			if err != nil {
				return nil, err
			}
			row.Components = append(row.Components, c)
		}
		return row, nil

	case ComponentLabel:
		var raw struct {
			Component jsontext.Value `json:"component"`
			ID        uint32         `json:"id"`
			Type      ComponentType  `json:"type"`
		}

		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		child, err := UnmarshalSubmittedComponent(raw.Component)
		if err != nil {
			return nil, err
		}
		return SubmittedLabel{
			Type:      raw.Type,
			ID:        raw.ID,
			Component: child,
		}, nil

	case ComponentTextInput:
		var txt SubmittedTextInput
		err := json.Unmarshal(data, &txt)
		return txt, err

	case ComponentStringSelect:
		var sel SubmittedStringSelect
		err := json.Unmarshal(data, &sel)
		return sel, err

	case ComponentUserSelect, ComponentRoleSelect, ComponentMentionableSelect, ComponentChannelSelect:
		var sel SubmittedSelect
		err := json.Unmarshal(data, &sel)
		return sel, err

	case ComponentRadioGroup:
		var r SubmittedRadioGroup
		err := json.Unmarshal(data, &r)
		return r, err

	case ComponentCheckboxGroup:
		var chk SubmittedCheckboxGroup
		err := json.Unmarshal(data, &chk)
		return chk, err

	case ComponentCheckbox:
		var chk SubmittedCheckbox
		err := json.Unmarshal(data, &chk)
		return chk, err

	case ComponentFileUpload:
		var up SubmittedFileUpload
		err := json.Unmarshal(data, &up)
		return up, err
	}

	return nil, fmt.Errorf("unknown submitted component type: %d", typ)
}
