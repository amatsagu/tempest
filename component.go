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

// Generic Component super struct
type Component struct {
	Type        uint8        `json:"type"`
	CustomID    string       `json:"custom_id,omitempty"`
	Style       ButtonStyle  `json:"style,omitempty"`
	Disabled    bool         `json:"disabled,omitempty"`
	Label       string       `json:"label,omitempty"`
	Emoji       *Emoji       `json:"emoji,omitempty"`
	URL         string       `json:"url,omitempty"`
	Placeholder string       `json:"placeholder,omitempty"`
	MinValues   int          `json:"min_values,omitempty"`
	MaxValues   int          `json:"max_values,omitempty"`
	Options     []*Option    `json:"options,omitempty"`
	Components  []*Component `json:"components,omitempty"`
}

type ButtonComponent struct {
	CustomId   string        `json:"custom_id"`
	Text       string        `json:"label,omitempty"` // Text label that appears on the button, max 80 characters.
	Emoji      *PartialEmoji `json:"emoji,omitempty"`
	Style      ButtonStyle   `json:"style"`
	Url        string        `json:"url,omitempty"` // A url for link-style buttons.
	IsDisabled bool          `json:"disabled,omitempty"`
	Type       uint8         `json:"type"` // It gonna always be = 2 for button components.
}

type SelectMenuComponent struct {
	CustomId        string              `json:"custom_id"`
	IsDisabled      bool                `json:"disabled,omitempty"`
	PlaceholderText string              `json:"placeholder,omitempty"` // Custom placeholder text if nothing is selected, max 150 characters
	MaxValues       uint64              `json:"max_values,omitempty"`
	MinValues       uint64              `json:"min_values,omitempty"`
	Options         []*SelectMenuOption `json:"options"`
	Type            uint8               `json:"type"` // It gonna always be = 3 for select menu components.
}

type SelectMenuOption struct {
	IsDefault   bool          `json:"default"`         // Whether to render this option as selected by default.
	Text        string        `json:"label,omitempty"` // Text label that appears on the option label, max 80 characters.
	Emoji       *PartialEmoji `json:"emoji,omitempty"`
	Description string        `json:"description,omitempty"` // An additional description of the option, max 100 characters.
	Value       string        `json:"value"`                 // Value to return back to app once clicked, max 100 characters.
}
