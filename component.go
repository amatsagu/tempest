package tempest

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

// Generic Component super struct
type Component struct {
	CustomId    string        `json:"custom_id,omitempty"`
	Type        ComponentType `json:"type"`
	Style       ButtonStyle   `json:"style,omitempty"`
	Disabled    bool          `json:"disabled,omitempty"`
	Label       string        `json:"label,omitempty"`
	Emoji       *Emoji        `json:"emoji,omitempty"`
	Url         string        `json:"url,omitempty"`
	Placeholder string        `json:"placeholder,omitempty"`
	MinValues   int           `json:"min_values,omitempty"`
	MaxValues   int           `json:"max_values,omitempty"`
	Options     []*Option     `json:"options,omitempty"`
	Components  []*Component  `json:"components,omitempty"`
}

type ButtonComponent struct {
	CustomId string        `json:"custom_id"`
	Type     ComponentType `json:"type"`            // It gonna always be = 2 for button components.
	Label    string        `json:"label,omitempty"` // Text label that appears on the button, max 80 characters.
	Emoji    *PartialEmoji `json:"emoji,omitempty"`
	Style    ButtonStyle   `json:"style"`
	Url      string        `json:"url,omitempty"` // A url for link-style buttons.
	Disabled bool          `json:"disabled,omitempty"`
}

type SelectMenuComponent struct {
	CustomId    string              `json:"custom_id"`
	Type        ComponentType       `json:"type"` // It gonna always be = 3 for select menu components.
	Disabled    bool                `json:"disabled,omitempty"`
	Placeholder string              `json:"placeholder,omitempty"` // Custom placeholder text if nothing is selected, max 150 characters
	MinValues   uint64              `json:"min_values,omitempty"`
	MaxValues   uint64              `json:"max_values,omitempty"`
	Options     []*SelectMenuOption `json:"options"`
}

type SelectMenuOption struct {
	Default     bool          `json:"default"`         // Whether to render this option as selected by default.
	Label       string        `json:"label,omitempty"` // Text label that appears on the option label, max 80 characters.
	Emoji       *PartialEmoji `json:"emoji,omitempty"`
	Description string        `json:"description,omitempty"` // An additional description of the option, max 100 characters.
	Value       string        `json:"value"`                 // Value to return back to app once clicked, max 100 characters.
}
