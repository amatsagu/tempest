package tempest

// ==========================================================================================
// QUICK INFO
// Components are so messy because Discord API is really inconsistent in this section
// and Golang team doesn't make it easier by refusing to add unions which so many people are asking for!
// Feel free to make a pull request with better type definitions if you have idea how to improve it.
// ==========================================================================================

// https://discord.com/developers/docs/interactions/message-components#button-object-button-styles
type ButtonStyle uint8

const (
	PRIMARY_BUTTON_STYLE   ButtonStyle = iota + 1 // blurple
	SECONDARY_BUTTON_STYLE                        // grey
	SUCCESS_BUTTON_STYLE                          // green
	DANGER_BUTTON_STYLE                           // red
	LINK_BUTTON_STYLE                             // grey, navigate to URL
)

type ComponentType uint8

// https://discord.com/developers/docs/interactions/message-components#component-object-component-types
const (
	ROW_COMPONENT_TYPE ComponentType = iota + 1
	BUTTON_COMPONENT_TYPE
	SELECT_MENU_COMPONENT_TYPE
	TEXT_INPUT_COMPONENT_TYPE
	USER_SELECT_COMPONENT_TYPE
	ROLE_SELECT_COMPONENT_TYPE
	MENTIONABLE_SELECT_COMPONENT_TYPE
	CHANNEL_SELECT_COMPONENT_TYPE
)

// https://discord.com/developers/docs/interactions/message-components#text-inputs-text-input-styles
type TextInputStyle uint8

const (
	SHORT_TEXT_INPUT_STYLE     TextInputStyle = iota + 1 // 	A single-line input.
	PARAGRAPH_TEXT_INPUT_STYLE                           // A multi-line input.
)

// Generic Component super struct (because Go doesn't support unions)!
//
// https://discord.com/developers/docs/interactions/message-components#button-object-button-structure
//
// https://discord.com/developers/docs/interactions/message-components#select-menu-object-select-menu-structure
//
// https://discord.com/developers/docs/interactions/message-components#text-inputs-text-input-structure
type Component struct {
	Type         ComponentType       `json:"type"`
	CustomID     string              `json:"custom_id,omitempty"`
	Style        uint8               `json:"style,omitempty"` // Either ButtonStyle or TextInputStyle.
	Label        string              `json:"label,omitempty"`
	Emoji        *PartialEmoji       `json:"emoji,omitempty"`
	URL          string              `json:"url,omitempty"`
	Disabled     bool                `json:"disabled,omitempty"`
	Placeholder  string              `json:"placeholder,omitempty"`
	MinValues    uint64              `json:"min_values,omitempty"`
	MaxValues    uint64              `json:"max_values,omitempty"`
	Required     bool                `json:"required,omitempty"`
	Options      []*SelectMenuOption `json:"options,omitempty"`
	Value        string              `json:"value,omitempty"`         // Contains menu choice or text input value from user modal submit.
	ChannelTypes []*ChannelType      `json:"channel_types,omitempty"` // Only available for 8th ComponentType.
}

// https://discord.com/developers/docs/interactions/message-components#select-menu-object-select-option-structure
type SelectMenuOption struct {
	Label       string        `json:"label,omitempty"`       // Text label that appears on the option label, max 80 characters.
	Description string        `json:"description,omitempty"` // An additional description of the option, max 100 characters.
	Emoji       *PartialEmoji `json:"emoji,omitempty"`
	Value       string        `json:"value"`   // Value to return back to app once clicked, max 100 characters.
	Default     bool          `json:"default"` // Whether to render this option as selected by default.
}

// https://discord.com/developers/docs/interactions/message-components#action-rows
type ComponentRow struct {
	Type       ComponentType `json:"type"` // Always 1
	Components []*Component  `json:"components"`
}
