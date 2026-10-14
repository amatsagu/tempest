package discord

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"
)

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-resolved-data-structure
type InteractionDataResolved struct {
	Users       map[UserID]User             `json:"users,omitzero"`
	Members     map[UserID]Member           `json:"members,omitzero"`
	Roles       map[RoleID]Role             `json:"roles,omitzero"`
	Channels    map[ChannelID]Channel       `json:"channels,omitzero"`
	Messages    map[MessageID]Message       `json:"messages,omitzero"`
	Attachments map[AttachmentID]Attachment `json:"attachments,omitzero"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-application-command-interaction-data-option-structure
type CommandInteractionOption struct {
	Value   any                        `json:"value,omitzero"`
	Name    string                     `json:"name"`
	Options []CommandInteractionOption `json:"options,omitzero"`
	Type    OptionType                 `json:"type"`
	Focused bool                       `json:"focused,omitzero"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-application-command-data-structure
type CommandInteractionData struct {
	Resolved *InteractionDataResolved   `json:"resolved,omitzero"`
	Name     string                     `json:"name"`
	Options  []CommandInteractionOption `json:"options,omitzero"`
	ID       CommandID                  `json:"id"`
	GuildID  GuildID                    `json:"guild_id,omitzero"`
	TargetID Snowflake                  `json:"target_id,omitzero"`
	Type     CommandType                `json:"type"`
}

func (d CommandInteractionData) Option(name string) (CommandInteractionOption, bool) {
	for _, opt := range d.Options {
		if opt.Name == name {
			return opt, true
		}
	}
	return CommandInteractionOption{}, false
}

func (d CommandInteractionData) StringOption(name string) (string, bool) {
	opt, ok := d.Option(name)
	if !ok || opt.Value == nil {
		return "", false
	}
	s, ok := opt.Value.(string)
	return s, ok
}

func (d CommandInteractionData) IntOption(name string) (int64, bool) {
	opt, ok := d.Option(name)
	if !ok || opt.Value == nil {
		return 0, false
	}
	switch v := opt.Value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		return i, err == nil
	}
	return 0, false
}

func (d CommandInteractionData) BoolOption(name string) (bool, bool) {
	opt, ok := d.Option(name)
	if !ok || opt.Value == nil {
		return false, false
	}
	b, ok := opt.Value.(bool)
	return b, ok
}

func (d CommandInteractionData) SnowflakeOption(name string) (Snowflake, bool) {
	s, ok := d.StringOption(name)
	if !ok {
		return 0, false
	}
	sf, err := StringToSnowflake(s)
	return sf, err == nil
}

func (d CommandInteractionData) UserOption(name string) (UserID, bool) {
	sf, ok := d.SnowflakeOption(name)
	return UserID{Snowflake: sf}, ok
}

func (d CommandInteractionData) ChannelOption(name string) (ChannelID, bool) {
	sf, ok := d.SnowflakeOption(name)
	return ChannelID{Snowflake: sf}, ok
}

func (d CommandInteractionData) RoleOption(name string) (RoleID, bool) {
	sf, ok := d.SnowflakeOption(name)
	return RoleID{Snowflake: sf}, ok
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-message-component-data-structure
type ComponentInteractionData struct {
	Resolved *InteractionDataResolved `json:"resolved,omitzero"`
	CustomID string                   `json:"custom_id"`
	Values   []string                 `json:"values,omitzero"`
	Type     ComponentType            `json:"component_type"`
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-object-modal-submit-data-structure
type ModalInteractionData struct {
	Resolved   *InteractionDataResolved `json:"resolved,omitzero"`
	CustomID   string                   `json:"custom_id"`
	Components []SubmittedComponent     `json:"components,omitzero"`
}

func (m *ModalInteractionData) UnmarshalJSON(data []byte) error {
	type rawModalData struct {
		Resolved   *InteractionDataResolved `json:"resolved,omitzero"`
		CustomID   string                   `json:"custom_id"`
		Components []jsontext.Value         `json:"components,omitzero"`
	}

	var raw rawModalData
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.CustomID = raw.CustomID
	m.Resolved = raw.Resolved
	m.Components = make([]SubmittedComponent, 0, len(raw.Components))

	for _, rawComp := range raw.Components {
		comp, err := UnmarshalSubmittedComponent(rawComp)
		if err != nil {
			return err
		}
		m.Components = append(m.Components, comp)
	}

	return nil
}

func (m ModalInteractionData) FindComponent(customID string) (SubmittedComponent, bool) {
	return findInSubmittedComponents(m.Components, customID)
}

func findInSubmittedComponents(comps []SubmittedComponent, customID string) (SubmittedComponent, bool) {
	for _, c := range comps {
		switch v := c.(type) {
		case SubmittedActionRow:
			if found, ok := findInSubmittedComponents(v.Components, customID); ok {
				return found, true
			}
		case SubmittedLabel:
			if v.Component != nil {
				if found, ok := findInSubmittedComponents([]SubmittedComponent{v.Component}, customID); ok {
					return found, true
				}
			}
		case SubmittedTextInput:
			if v.CustomID == customID {
				return v, true
			}
		case SubmittedRadioGroup:
			if v.CustomID == customID {
				return v, true
			}
		case SubmittedCheckbox:
			if v.CustomID == customID {
				return v, true
			}
		case SubmittedCheckboxGroup:
			if v.CustomID == customID {
				return v, true
			}
		case SubmittedStringSelect:
			if v.CustomID == customID {
				return v, true
			}
		case SubmittedSelect:
			if v.CustomID == customID {
				return v, true
			}
		case SubmittedFileUpload:
			if v.CustomID == customID {
				return v, true
			}
		}
	}
	return nil, false
}

func (m ModalInteractionData) TextInput(customID string) (string, bool) {
	comp, ok := m.FindComponent(customID)
	if !ok {
		return "", false
	}
	if txt, ok := comp.(SubmittedTextInput); ok {
		return txt.Value, true
	}
	return "", false
}

func (m ModalInteractionData) Radio(customID string) (string, bool) {
	comp, ok := m.FindComponent(customID)
	if !ok {
		return "", false
	}
	if r, ok := comp.(SubmittedRadioGroup); ok {
		return r.Value, true
	}
	return "", false
}

func (m ModalInteractionData) Checkbox(customID string) bool {
	comp, ok := m.FindComponent(customID)
	if !ok {
		return false
	}
	if chk, ok := comp.(SubmittedCheckbox); ok {
		return chk.Value
	}
	return false
}

func (m ModalInteractionData) CheckboxGroup(customID string) ([]string, bool) {
	comp, ok := m.FindComponent(customID)
	if !ok {
		return nil, false
	}
	if chk, ok := comp.(SubmittedCheckboxGroup); ok {
		return chk.Values, true
	}
	return nil, false
}

func (m ModalInteractionData) Select(customID string) ([]string, bool) {
	comp, ok := m.FindComponent(customID)
	if !ok {
		return nil, false
	}
	if sel, ok := comp.(SubmittedStringSelect); ok {
		return sel.Values, true
	}
	return nil, false
}

func (m ModalInteractionData) FileUpload(customID string) ([]AttachmentID, bool) {
	comp, ok := m.FindComponent(customID)
	if !ok {
		return nil, false
	}
	if up, ok := comp.(SubmittedFileUpload); ok {
		return up.Values, true
	}
	return nil, false
}
