package discord

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// https://docs.discord.com/developers/components/reference#label-label-structure
type Label struct {
	TextInput     *TextInput     `json:"-"`
	StringSelect  *StringSelect  `json:"-"`
	SelectMenu    *SelectMenu    `json:"-"`
	FileUpload    *FileUpload    `json:"-"`
	RadioGroup    *RadioGroup    `json:"-"`
	CheckboxGroup *CheckboxGroup `json:"-"`
	Checkbox      *Checkbox      `json:"-"`
	Label         string         `json:"label"`
	Description   string         `json:"description,omitzero"`
	ID            uint32         `json:"id,omitzero"`
	Type          ComponentType  `json:"type"`
}

func (l Label) ComponentType() ComponentType { return ComponentLabel }

func (l Label) MarshalJSONTo(enc *jsontext.Encoder) error {
	type rawLabel Label

	switch {
	case l.TextInput != nil:
		return json.MarshalEncode(enc, struct {
			Component *TextInput `json:"component"`
			rawLabel
		}{Component: l.TextInput, rawLabel: rawLabel(l)})
	case l.StringSelect != nil:
		return json.MarshalEncode(enc, struct {
			Component *StringSelect `json:"component"`
			rawLabel
		}{Component: l.StringSelect, rawLabel: rawLabel(l)})
	case l.SelectMenu != nil:
		return json.MarshalEncode(enc, struct {
			Component *SelectMenu `json:"component"`
			rawLabel
		}{Component: l.SelectMenu, rawLabel: rawLabel(l)})
	case l.FileUpload != nil:
		return json.MarshalEncode(enc, struct {
			Component *FileUpload `json:"component"`
			rawLabel
		}{Component: l.FileUpload, rawLabel: rawLabel(l)})
	case l.RadioGroup != nil:
		return json.MarshalEncode(enc, struct {
			Component *RadioGroup `json:"component"`
			rawLabel
		}{Component: l.RadioGroup, rawLabel: rawLabel(l)})
	case l.CheckboxGroup != nil:
		return json.MarshalEncode(enc, struct {
			Component *CheckboxGroup `json:"component"`
			rawLabel
		}{Component: l.CheckboxGroup, rawLabel: rawLabel(l)})
	case l.Checkbox != nil:
		return json.MarshalEncode(enc, struct {
			Component *Checkbox `json:"component"`
			rawLabel
		}{Component: l.Checkbox, rawLabel: rawLabel(l)})
	default:
		return json.MarshalEncode(enc, rawLabel(l))
	}
}

func (l *Label) UnmarshalJSON(data []byte) error {
	type rawLabel Label
	var raw struct {
		rawLabel
		Component jsontext.Value `json:"component,omitzero"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*l = Label(raw.rawLabel)
	if len(raw.Component) > 0 {
		var peek struct {
			Type ComponentType `json:"type"`
		}
		if err := json.Unmarshal(raw.Component, &peek); err != nil {
			return err
		}
		switch peek.Type {
		case ComponentTextInput:
			var input TextInput
			if err := json.Unmarshal(raw.Component, &input); err != nil {
				return err
			}
			l.TextInput = &input
		case ComponentStringSelect:
			var sel StringSelect
			if err := json.Unmarshal(raw.Component, &sel); err != nil {
				return err
			}
			l.StringSelect = &sel
		case ComponentUserSelect, ComponentRoleSelect, ComponentMentionableSelect, ComponentChannelSelect:
			var sel SelectMenu
			if err := json.Unmarshal(raw.Component, &sel); err != nil {
				return err
			}
			l.SelectMenu = &sel
		case ComponentFileUpload:
			var up FileUpload
			if err := json.Unmarshal(raw.Component, &up); err != nil {
				return err
			}
			l.FileUpload = &up
		case ComponentRadioGroup:
			var r RadioGroup
			if err := json.Unmarshal(raw.Component, &r); err != nil {
				return err
			}
			l.RadioGroup = &r
		case ComponentCheckboxGroup:
			var chk CheckboxGroup
			if err := json.Unmarshal(raw.Component, &chk); err != nil {
				return err
			}
			l.CheckboxGroup = &chk
		case ComponentCheckbox:
			var chk Checkbox
			if err := json.Unmarshal(raw.Component, &chk); err != nil {
				return err
			}
			l.Checkbox = &chk
		}
	}
	return nil
}

// https://docs.discord.com/developers/components/reference#file-upload-file-upload-structure
type FileUpload struct {
	MinValues *uint8        `json:"min_values,omitzero"`
	MaxValues *uint8        `json:"max_values,omitzero"`
	CustomID  string        `json:"custom_id"`
	FileTypes []string      `json:"file_types,omitzero"`
	ID        uint32        `json:"id,omitzero"`
	Type      ComponentType `json:"type"`
	Required  bool          `json:"required,omitzero"`
}

// https://docs.discord.com/developers/components/reference#radio-group-option-structure
type RadioGroupOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitzero"`
	Default     bool   `json:"default,omitzero"`
}

// https://docs.discord.com/developers/components/reference#radio-group
type RadioGroup struct {
	CustomID string             `json:"custom_id"`
	Options  []RadioGroupOption `json:"options"`
	ID       uint32             `json:"id,omitzero"`
	Type     ComponentType      `json:"type"`
	Required bool               `json:"required,omitzero"`
}

// https://docs.discord.com/developers/components/reference#checkbox-group-option-structure
type CheckboxGroupOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitzero"`
	Default     bool   `json:"default,omitzero"`
}

// https://docs.discord.com/developers/components/reference#checkbox-group
type CheckboxGroup struct {
	MinValues *uint8                `json:"min_values,omitzero"`
	MaxValues *uint8                `json:"max_values,omitzero"`
	CustomID  string                `json:"custom_id"`
	Options   []CheckboxGroupOption `json:"options"`
	ID        uint32                `json:"id,omitzero"`
	Type      ComponentType         `json:"type"`
	Required  bool                  `json:"required,omitzero"`
}

// https://docs.discord.com/developers/components/reference#checkbox
type Checkbox struct {
	CustomID string        `json:"custom_id"`
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"`
	Default  bool          `json:"default,omitzero"`
}

func NewLabel(label string, input TextInput) Label {
	return Label{
		Type:      ComponentLabel,
		Label:     label,
		TextInput: &input,
	}
}

func NewLabelWithStringSelect(label string, selectMenu StringSelect) Label {
	return Label{
		Type:         ComponentLabel,
		Label:        label,
		StringSelect: &selectMenu,
	}
}

func NewLabelWithSelect(label string, selectMenu SelectMenu) Label {
	return Label{
		Type:       ComponentLabel,
		Label:      label,
		SelectMenu: &selectMenu,
	}
}

func NewLabelWithRadioGroup(label string, radio RadioGroup) Label {
	return Label{
		Type:       ComponentLabel,
		Label:      label,
		RadioGroup: &radio,
	}
}

func NewLabelWithCheckboxGroup(label string, chk CheckboxGroup) Label {
	return Label{
		Type:          ComponentLabel,
		Label:         label,
		CheckboxGroup: &chk,
	}
}

func NewLabelWithCheckbox(label string, chk Checkbox) Label {
	return Label{
		Type:     ComponentLabel,
		Label:    label,
		Checkbox: &chk,
	}
}

func NewLabelWithFileUpload(label string, up FileUpload) Label {
	return Label{
		Type:       ComponentLabel,
		Label:      label,
		FileUpload: &up,
	}
}

func NewRadioGroup(customID string, options ...RadioGroupOption) RadioGroup {
	return RadioGroup{
		Type:     ComponentRadioGroup,
		CustomID: customID,
		Options:  options,
	}
}

func NewCheckboxGroup(customID string, options ...CheckboxGroupOption) CheckboxGroup {
	return CheckboxGroup{
		Type:     ComponentCheckboxGroup,
		CustomID: customID,
		Options:  options,
	}
}

func NewCheckbox(customID string, defaultChecked bool) Checkbox {
	return Checkbox{
		Type:     ComponentCheckbox,
		CustomID: customID,
		Default:  defaultChecked,
	}
}

func NewFileUpload(customID string, fileTypes ...string) FileUpload {
	return FileUpload{
		Type:      ComponentFileUpload,
		CustomID:  customID,
		FileTypes: fileTypes,
	}
}
