package discord

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// https://docs.discord.com/developers/components/reference#action-row-action-row-structure
type ActionRow struct {
	StringSelect *StringSelect `json:"-"`
	SelectMenu   *SelectMenu   `json:"-"`
	Buttons      []Button      `json:"-"`
	ID           uint32        `json:"id,omitzero"`
	Type         ComponentType `json:"type"`
}

func (a ActionRow) ComponentType() ComponentType { return ComponentActionRow }

func (a ActionRow) MarshalJSONTo(enc *jsontext.Encoder) error {
	type rawActionRow ActionRow

	switch {
	case len(a.Buttons) > 0:
		return json.MarshalEncode(enc, struct {
			Components []Button `json:"components"`
			rawActionRow
		}{Components: a.Buttons, rawActionRow: rawActionRow(a)})
	case a.StringSelect != nil:
		return json.MarshalEncode(enc, struct {
			Components []*StringSelect `json:"components"`
			rawActionRow
		}{Components: []*StringSelect{a.StringSelect}, rawActionRow: rawActionRow(a)})
	case a.SelectMenu != nil:
		return json.MarshalEncode(enc, struct {
			Components []*SelectMenu `json:"components"`
			rawActionRow
		}{Components: []*SelectMenu{a.SelectMenu}, rawActionRow: rawActionRow(a)})
	default:
		return json.MarshalEncode(enc, rawActionRow(a))
	}
}

func (a *ActionRow) UnmarshalJSON(data []byte) error {
	type rawActionRow ActionRow
	var raw struct {
		Components []jsontext.Value `json:"components"`
		rawActionRow
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*a = ActionRow(raw.rawActionRow)
	for _, c := range raw.Components {
		var peek struct {
			Type ComponentType `json:"type"`
		}

		if err := json.Unmarshal(c, &peek); err != nil {
			return err
		}

		switch peek.Type {
		case ComponentButton:
			var btn Button
			if err := json.Unmarshal(c, &btn); err != nil {
				return err
			}
			a.Buttons = append(a.Buttons, btn)
		case ComponentStringSelect:
			var sel StringSelect
			if err := json.Unmarshal(c, &sel); err != nil {
				return err
			}
			a.StringSelect = &sel
		case ComponentUserSelect, ComponentRoleSelect, ComponentMentionableSelect, ComponentChannelSelect:
			var sel SelectMenu
			if err := json.Unmarshal(c, &sel); err != nil {
				return err
			}
			a.SelectMenu = &sel
		}
	}
	return nil
}

// https://docs.discord.com/developers/components/reference#section-section-structure
type Section struct {
	Button     *Button       `json:"-"`
	Thumbnail  *Thumbnail    `json:"-"`
	Components []TextDisplay `json:"components"`
	ID         uint32        `json:"id,omitzero"`
	Type       ComponentType `json:"type"`
}

func (s Section) ComponentType() ComponentType { return ComponentSection }

func (s Section) MarshalJSONTo(enc *jsontext.Encoder) error {
	type rawSection Section

	switch {
	case s.Button != nil:
		return json.MarshalEncode(enc, struct {
			Accessory *Button `json:"accessory"`
			rawSection
		}{Accessory: s.Button, rawSection: rawSection(s)})
	case s.Thumbnail != nil:
		return json.MarshalEncode(enc, struct {
			Accessory *Thumbnail `json:"accessory"`
			rawSection
		}{Accessory: s.Thumbnail, rawSection: rawSection(s)})
	default:
		return json.MarshalEncode(enc, rawSection(s))
	}
}

func (s *Section) UnmarshalJSON(data []byte) error {
	type rawSection Section
	var raw struct {
		Accessory jsontext.Value `json:"accessory,omitzero"`
		rawSection
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = Section(raw.rawSection)
	if len(raw.Accessory) > 0 {
		var peek struct {
			Type ComponentType `json:"type"`
		}

		if err := json.Unmarshal(raw.Accessory, &peek); err != nil {
			return err
		}

		switch peek.Type {
		case ComponentButton:
			var btn Button
			if err := json.Unmarshal(raw.Accessory, &btn); err != nil {
				return err
			}
			s.Button = &btn
		case ComponentThumbnail:
			var thumb Thumbnail
			if err := json.Unmarshal(raw.Accessory, &thumb); err != nil {
				return err
			}
			s.Thumbnail = &thumb
		}
	}
	return nil
}

// https://docs.discord.com/developers/components/reference#container-container-structure
type Container struct {
	Components  Components    `json:"components"`
	ID          uint32        `json:"id,omitzero"`
	AccentColor uint32        `json:"accent_color,omitzero"`
	Type        ComponentType `json:"type"`
	Spoiler     bool          `json:"spoiler,omitzero"`
}

func (c Container) ComponentType() ComponentType { return ComponentContainer }

// https://docs.discord.com/developers/components/reference#separator-separator-structure
type Separator struct {
	ID      uint32        `json:"id,omitzero"`
	Type    ComponentType `json:"type"`
	Spacing uint8         `json:"spacing,omitzero"`
	Divider bool          `json:"divider,omitzero"`
}

func (s Separator) ComponentType() ComponentType { return ComponentSeparator }
