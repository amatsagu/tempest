package discord

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
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

// UnmarshalJSONFrom decodes the modal data in a single pass, reading each
// submitted component as a raw value through the shared decoder instead of
// buffering the whole components array into intermediate jsontext.Values.
func (m *ModalInteractionData) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() != '{' {
		return fmt.Errorf("modal interaction data: expected JSON object")
	}

	if _, err := dec.ReadToken(); err != nil {
		return err
	}

	for dec.PeekKind() != '}' {
		name, err := dec.ReadToken()
		if err != nil {
			return err
		}

		switch name.String() {
		case "custom_id":
			if err := json.UnmarshalDecode(dec, &m.CustomID); err != nil {
				return err
			}
		case "resolved":
			if err := json.UnmarshalDecode(dec, &m.Resolved); err != nil {
				return err
			}
		case "components":
			if err := m.decodeComponents(dec); err != nil {
				return err
			}
		default:
			if err := dec.SkipValue(); err != nil {
				return err
			}
		}
	}

	_, err := dec.ReadToken()
	return err
}

func (m *ModalInteractionData) decodeComponents(dec *jsontext.Decoder) error {
	if dec.PeekKind() != '[' {
		return fmt.Errorf("modal interaction data: expected JSON array for components")
	}

	if _, err := dec.ReadToken(); err != nil {
		return err
	}

	for dec.PeekKind() != ']' {
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}

		comp, err := UnmarshalSubmittedComponent(raw)
		if err != nil {
			return err
		}

		m.Components = append(m.Components, comp)
	}

	_, err := dec.ReadToken()
	return err
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
