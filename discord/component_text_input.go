package discord

// https://docs.discord.com/developers/components/reference#text-input-text-input-styles
type TextInputStyle uint8

const (
	TextInputShort TextInputStyle = iota + 1
	TextInputParagraph
)

// https://docs.discord.com/developers/components/reference#text-input-text-input-structure
type TextInput struct {
	MinLength   *uint32        `json:"min_length,omitzero"`
	MaxLength   *uint32        `json:"max_length,omitzero"`
	CustomID    string         `json:"custom_id"`
	Label       string         `json:"label,omitzero"`
	Value       string         `json:"value,omitzero"`
	Placeholder string         `json:"placeholder,omitzero"`
	ID          uint32         `json:"id,omitzero"`
	Type        ComponentType  `json:"type"`
	Style       TextInputStyle `json:"style"`
	Required    bool           `json:"required,omitzero"`
}

func NewTextInput(customID, placeholder string, style TextInputStyle) TextInput {
	return TextInput{
		Type:        ComponentTextInput,
		Style:       style,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewShortTextInput(customID, placeholder string) TextInput {
	return NewTextInput(customID, placeholder, TextInputShort)
}

func NewParagraphTextInput(customID, placeholder string) TextInput {
	return NewTextInput(customID, placeholder, TextInputParagraph)
}
