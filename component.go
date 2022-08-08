package tempest

// ==========================================================================================
// QUICK INFO
// Components are so messy because Discord API is really inconsistent in this section
// and Golang team doesn't make it easier by refusing to add unions which so many people are asking for!
// Feel free to make a pull request with better type definitions if you have idea how to improve it.
// ==========================================================================================

type ButtonStyle uint8

const (
	BUTTON_PRIMARY   ButtonStyle = iota + 1 // BUTTON_PRIMARY blurple
	BUTTON_SECONDARY                        // BUTTON_SECONDARY grey
	BUTTON_SUCCESS                          // BUTTON_SUCCESS green
	BUTTON_DANGER                           // BUTTON_DANGER red
	BUTTON_LINK                             // BUTTON_LINK grey, navigate to URL
)

type OptionType uint8

const (
	OPTION_SUB_COMMAND OptionType = iota + 1
	_                             // OPTION_SUB_COMMAND_GROUP (not supported)
	OPTION_STRING
	OPTION_INTEGER
	OPTION_BOOLEAN
	OPTION_USER
	OPTION_CHANNEL
	OPTION_ROLE
	OPTION_MENTIONABLE
	OPTION_NUMBER
	OPTION_ATTACHMENT
)

type ComponentType uint8

const (
	COMPONENT_ROW ComponentType = iota + 1
	COMPONENT_BUTTON
	COMPONENT_SELECT_MENU
	COMPONENT_TEXT_INPUT
)

type TextInputStyle uint8

const (
	TEXT_INPUT_SHORT     = iota + 1 // 	A single-line input.
	TEXT_INPUT_PARAGRAPH            // A multi-line input.
)

// Generic Component super struct! Use "ButtonComponent", "SelectMenuComponent" or "TextInputComponent" whenever possible and this super struct as "any" component.
type Component struct {
	CustomId    string              `json:"custom_id,omitempty"`
	Type        ComponentType       `json:"type"`
	Style       ButtonStyle         `json:"style,omitempty"`
	Label       string              `json:"label,omitempty"`
	Emoji       *PartialEmoji       `json:"emoji,omitempty"`
	Url         string              `json:"url,omitempty"`
	Disabled    bool                `json:"disabled,omitempty"`
	Placeholder string              `json:"placeholder,omitempty"`
	MinValues   uint64              `json:"min_values,omitempty"`
	MaxValues   uint64              `json:"max_values,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Options     []*SelectMenuOption `json:"options,omitempty"`
	Components  []*Component        `json:"components,omitempty"`
}

type ButtonComponent struct {
	CustomId string        `json:"custom_id"`
	Type     ComponentType `json:"type"` // It gonna always be = 2 for button components.
	Style    ButtonStyle   `json:"style"`
	Label    string        `json:"label,omitempty"` // Text label that appears on the button, max 80 characters.
	Emoji    *PartialEmoji `json:"emoji,omitempty"`
	Url      string        `json:"url,omitempty"` // A url for link-style buttons.
	Disabled bool          `json:"disabled,omitempty"`
}

type SelectMenuComponent struct {
	CustomId    string              `json:"custom_id"`
	Type        ComponentType       `json:"type"`                  // It gonna always be = 3 for select menu components.
	Placeholder string              `json:"placeholder,omitempty"` // Custom placeholder text if nothing is selected, max 150 characters.
	MinValues   uint64              `json:"min_values,omitempty"`
	MaxValues   uint64              `json:"max_values,omitempty"`
	Options     []*SelectMenuOption `json:"options"`
	Disabled    bool                `json:"disabled,omitempty"`
}

type SelectMenuOption struct {
	Label       string        `json:"label,omitempty"`       // Text label that appears on the option label, max 80 characters.
	Description string        `json:"description,omitempty"` // An additional description of the option, max 100 characters.
	Emoji       *PartialEmoji `json:"emoji,omitempty"`
	Value       string        `json:"value"`   // Value to return back to app once clicked, max 100 characters.
	Default     bool          `json:"default"` // Whether to render this option as selected by default.
}

type TextInputComponent struct {
	CustomId    string         `json:"custom_id"`
	Type        ComponentType  `json:"type"` // It gonna always be = 4 for text input components.
	Style       TextInputStyle `json:"style"`
	Label       string         `json:"label"`                 // Text label for text input, max 45 characters.
	Placeholder string         `json:"placeholder,omitempty"` // Custom placeholder text if the input is empty, max 100 characters.
	Value       string         `json:"value,omitempty"`       // A pre-filled value for this component, max 4000 characters.
	MinValues   uint64         `json:"min_values,omitempty"`
	MaxValues   uint64         `json:"max_values,omitempty"`
	Required    bool           `json:"required,omitempty"` // Whether this component is required to be filled, default = true.
}
