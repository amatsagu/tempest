package tempest

// https://discord.com/developers/docs/components/reference#button-button-styles
type ButtonStyle uint8

const (
	PRIMARY_BUTTON_STYLE   ButtonStyle = iota + 1 // blurple (custom_id field is required)
	SECONDARY_BUTTON_STYLE                        // grey (custom_id field is required)
	SUCCESS_BUTTON_STYLE                          // green (custom_id field is required)
	DANGER_BUTTON_STYLE                           // red (custom_id field is required)
	LINK_BUTTON_STYLE                             // grey, navigate to URL (url field is required)
	PREMIUM_BUTTON_STYLE                          // By default same as primary but will automatically use SKU icon, name & price (sky_id field is required)
)

type ComponentType uint8

// https://discord.com/developers/docs/interactions/message-components#component-object-component-types
const (
	ACTION_ROW_COMPONENT_TYPE         ComponentType = iota + 1 // Layout component for Messages
	BUTTON_COMPONENT_TYPE                                      // Interactive component for Messages
	STRING_SELECT_COMPONENT_TYPE                               // Interactive component for Messages and Modals
	TEXT_INPUT_COMPONENT_TYPE                                  // Interactive component for Modals
	USER_SELECT_COMPONENT_TYPE                                 // Interactive component for Messages
	ROLE_SELECT_COMPONENT_TYPE                                 // Interactive component for Messages
	MENTIONABLE_SELECT_COMPONENT_TYPE                          // Interactive component for Messages
	CHANNEL_SELECT_COMPONENT_TYPE                              // Interactive component for Messages
	SECTION_COMPONENT_TYPE                                     // Layout component for Messages
	TEXT_DISPLAY_COMPONENT_TYPE                                // Content component for Messages
	THUMBNAIL_COMPONENT_TYPE                                   // Content component for Messages
	MEDIA_GALLERY_COMPONENT_TYPE                               // Content component for Messages
	FILE_COMPONENT_TYPE                                        // Content component for Messages
	SEPARATOR_COMPONENT_TYPE                                   // Layout component for Messages
	_                                                          //
	_                                                          //
	CONTAINER_COMPONENT_TYPE                                   // Layout component for Messages
	LABEL_COMPONENT_TYPE                                       // Layout component for Modals
)

// https://discord.com/developers/docs/interactions/message-components#text-inputs-text-input-styles
type TextInputStyle uint8

const (
	SHORT_TEXT_INPUT_STYLE     TextInputStyle = iota + 1 // 	A single-line input.
	PARAGRAPH_TEXT_INPUT_STYLE                           // A multi-line input.
)

// https://discord.com/developers/docs/components/reference#user-select-select-default-value-structure
type DefaultValueType string

const (
	USER_DEFAULT_VALUE    DefaultValueType = "user"
	ROLE_DEFAULT_VALUE    DefaultValueType = "role"
	CHANNEL_DEFAULT_VALUE DefaultValueType = "channel"
)

// https://discord.com/developers/docs/components/reference#action-row-action-row-structure
type ActionRowComponent struct {
	Type       ComponentType          `json:"type"` // Always = ACTION_ROW_COMPONENT_TYPE (1)
	ID         uint32                 `json:"id,omitempty"`
	Components []InteractiveComponent `json:"components"` // Up to 5 interactive button components or a single select component
}

// https://discord.com/developers/docs/components/reference#anatomy-of-a-component
// type UnknownComponent struct {
// 	Type ComponentType `json:"type"`
// 	ID   uint32        `json:"id,omitempty"`
// }

// A Button is an interactive component that can only be used in messages.
// It creates clickable elements that users can interact with, sending an interaction to your app when clicked.
//
// Buttons must be placed inside an Action Row or a Section's accessory field.
//
// https://discord.com/developers/docs/components/reference#button
type ButtonComponent struct {
	Type     ComponentType `json:"type"` // Always = BUTTON_COMPONENT_TYPE (2)
	ID       uint32        `json:"id,omitempty"`
	Style    ButtonStyle   `json:"style"`
	Label    string        `json:"label,omitempty"`
	Emoji    *Emoji        `json:"emoji,omitempty"` // It may only contain id, name, and animated from regular Emoji struct.
	CustomID string        `json:"custom_id,omitempty"`
	SkuID    Snowflake     `json:"sku_id,omitempty"` // Identifier for a purchasable SKU, only available when using premium-style buttons. Premium buttons do not send an interaction to your app when clicked.
	URL      string        `json:"url,omitempty"`
	Disabled bool          `json:"disabled"`
}

// https://discord.com/developers/docs/components/reference#string-select
type StringSelectComponent struct {
	Type        ComponentType      `json:"type"` // Always = STRING_SELECT_COMPONENT_TYPE (3). For responses, only provided for modal interactions
	ID          uint32             `json:"id,omitempty"`
	CustomID    string             `json:"custom_id,omitempty"`
	Options     []SelectMenuOption `json:"options,omitzero"`
	Placeholder string             `json:"placeholder,omitempty"`
	MinValues   uint8              `json:"min_values,omitempty"`
	MaxValues   uint8              `json:"max_values,omitempty"`
	Disabled    bool               `json:"disabled"`
	Required    bool               `json:"required"`

	// Below 2 fields are controlled by API and should be readonly for us, developers.
	// https://discord.com/developers/docs/components/reference#string-select-string-select-interaction-response-structure
	ComponentType ComponentType `json:"component_type,omitempty"` // This field is ignored and provided by the API as part of the response (only provided for message interaction)
	Values        []string      `json:"values,omitzero"`          // This field is ignored and provided by the API as part of the response.
}

// https://discord.com/developers/docs/components/reference#string-select-select-option-structure
type SelectMenuOption struct {
	Label       string `json:"label"`                 // Text label that appears on the option label, max 80 characters.
	Value       string `json:"value"`                 // Value to return back to app once clicked, max 100 characters.
	Description string `json:"description,omitempty"` // An additional description of the option, max 100 characters.
	Emoji       *Emoji `json:"emoji,omitempty"`       // It may only contain id, name, and animated from regular Emoji struct.
	Default     bool   `json:"default"`               // Whether to render this option as selected by default.
}

// https://discord.com/developers/docs/components/reference#text-input
type TextInputComponent struct {
	Type        ComponentType  `json:"type"` // Always = TEXT_INPUT_COMPONENT_TYPE (4)
	ID          uint32         `json:"id,omitempty"`
	CustomID    string         `json:"custom_id,omitempty"`
	Style       TextInputStyle `json:"style"`
	Label       string         `json:"label,omitempty"`       // Deprecated: use `label` and `description` on the Label component
	MinLength   uint16         `json:"min_length,omitempty"`  // min: 0, max: 4000 characters
	MaxLength   uint16         `json:"max_length,omitempty"`  // min: 1, max: 4000 characters
	Required    bool           `json:"required"`              // Whether this component is required to be filled, defaults to true
	Value       string         `json:"value,omitempty"`       // Pre-filled value for this component, max 4000 characters.
	Placeholder string         `json:"placeholder,omitempty"` // max: 100 characters
}

// This component can be used for User, Role, Mentionable & Channel select components.
//
// https://discord.com/developers/docs/components/reference#user-select-user-select-structure
//
// https://discord.com/developers/docs/components/reference#role-select-role-select-structure
//
// https://discord.com/developers/docs/components/reference#mentionable-select-mentionable-select-structure
//
// https://discord.com/developers/docs/components/reference#channel-select-channel-select-structure
type SelectComponent struct {
	Type          ComponentType        `json:"type"`
	ID            uint32               `json:"id,omitempty"`
	CustomID      string               `json:"custom_id,omitempty"`
	ChannelTypes  []ChannelType        `json:"channel_types,omitzero"` // List of channel types to include in the channel select component
	Placeholder   string               `json:"placeholder,omitempty"`  // Placeholder text if nothing is selected, max: 150 characters
	DefaultValues []DefaultValueOption `json:"default_values,omitzero"`
	MinValues     uint8                `json:"min_values,omitempty"`
	MaxValues     uint8                `json:"max_values,omitempty"`
	Disabled      bool                 `json:"disabled"`
}

// https://discord.com/developers/docs/components/reference#user-select-select-default-value-structure
type DefaultValueOption struct {
	ID   Snowflake        `json:"id"`   // Snowflake ID of a user, role, or channel
	Type DefaultValueType `json:"type"` // Type of value that id represents. Either "user", "role", or "channel"
}

// https://discord.com/developers/docs/components/reference#section-section-structure
type SectionComponent struct {
	Type       ComponentType          `json:"type"` // Always = SECTION_COMPONENT_TYPE (9)
	ID         uint32                 `json:"id,omitempty"`
	Components []TextDisplayComponent `json:"components,omitzero"` // One to three text components

	// AccessoryComponent is interface so it should't be a pointer in this case.

	Accessory AccessoryComponent `json:"accessory,omitempty"`
}

// https://discord.com/developers/docs/components/reference#text-display-text-display-structure
type TextDisplayComponent struct {
	Type    ComponentType `json:"type"` // Always = TEXT_DISPLAY_COMPONENT_TYPE (10)
	ID      uint32        `json:"id,omitempty"`
	Content string        `json:"content"`
}

// https://discord.com/developers/docs/components/reference#thumbnail-thumbnail-structure
type ThumbnailComponent struct {
	Type        ComponentType     `json:"type"` // Always = THUMBNAIL_COMPONENT_TYPE (11)
	ID          uint32            `json:"id,omitempty"`
	Media       UnfurledMediaItem `json:"media"`
	Description string            `json:"description,omitempty"` // Alt text for the media, max 1024 characters
	Spoiler     bool              `json:"spoiler"`
}

// https://discord.com/developers/docs/components/reference#unfurled-media-item-structure
type UnfurledMediaItem struct {
	URL          string    `json:"url"` // 	Supports arbitrary urls and attachment://<filename> references
	ProxyURL     string    `json:"proxy_url,omitempty"`
	Width        uint32    `json:"width,omitempty"`
	Height       uint32    `json:"height,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`  // This field is ignored and provided by the API as part of the response
	AttachmentID Snowflake `json:"attachment_id,omitempty"` // This field is ignored and provided by the API as part of the response
}

// https://discord.com/developers/docs/components/reference#media-gallery-media-gallery-structure
type MediaGalleryComponent struct {
	Type  ComponentType      `json:"type"` // Always = MEDIA_GALLERY_COMPONENT_TYPE (12)
	ID    uint32             `json:"id,omitempty"`
	Items []MediaGalleryItem `json:"items,omitzero"` // 1 to 10 media gallery items
}

// https://discord.com/developers/docs/components/reference#media-gallery-media-gallery-item-structure
type MediaGalleryItem struct {
	Media       UnfurledMediaItem `json:"media"`
	Description string            `json:"description,omitempty"` // Alt text for the media, max 1024 characters
	Spoiler     bool              `json:"spoiler"`
}

// https://discord.com/developers/docs/components/reference#file-file-structure
type FileComponent struct {
	Type    ComponentType     `json:"type"` // Always = FILE_COMPONENT_TYPE (13)
	ID      uint32            `json:"id,omitempty"`
	File    UnfurledMediaItem `json:"file"` // This unfurled media item is unique in that it only supports attachment references using the attachment://<filename> syntax.
	Spoiler bool              `json:"spoiler"`

	// Below 2 fields are controlled by API and should be readonly for us, developers.

	Name string `json:"name,omitempty"` // This field is ignored and provided by the API as part of the response.
	Size uint32 `json:"size,omitempty"` // The size of the file in bytes. This field is ignored and provided by the API as part of the response.
}

// https://discord.com/developers/docs/components/reference#separator-separator-structure
type SeparatorComponent struct {
	Type    ComponentType `json:"type"` // Always = SEPARATOR_COMPONENT_TYPE (14)
	ID      uint32        `json:"id,omitempty"`
	Divider bool          `json:"divider"`           // Whether a visual divider should be displayed in the component (defaults to true).
	Spacing uint8         `json:"spacing,omitempty"` // Size of separator padding—1 for small padding, 2 for large padding (defaults to 1).
}

// https://discord.com/developers/docs/components/reference#container-container-structure
type ContainerComponent struct {
	Type        ComponentType  `json:"type"` // Always = CONTAINER_COMPONENT_TYPE (15)
	ID          uint32         `json:"id,omitempty"`
	Components  []AnyComponent `json:"components,omitzero"`    // Components of the type action row, text display, section, media gallery, separator or file.
	AccentColor uint32         `json:"accent_color,omitempty"` // Color for the accent on the container as RGB from 0x000000 to 0xFFFFFF.
	Spoiler     bool           `json:"spoiler"`
}

// A top-level layout component that wraps modal components with text as a label and optional description
// https://discord.com/developers/docs/components/reference#label
type LabelComponent struct {
	Type        ComponentType       `json:"type"`                  // Always = LABEL_COMPONENT_TYPE (18)
	ID          uint32              `json:"id,omitempty"`          // Optional identifier for component
	Label       string              `json:"label"`                 // Text that appears on the label, max 45 characters.
	Description string              `json:"description,omitempty"` // Additional description for the label, max 100 characters.
	Component   LabelChildComponent `json:"component"`             // The component within the label/
}
