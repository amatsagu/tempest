package tempest

// https://docs.discord.com/developers/components/reference#button-button-styles
type ButtonStyle uint8

const (
	PRIMARY_BUTTON_STYLE   ButtonStyle = iota + 1 // blurple (custom_id field is required)
	SECONDARY_BUTTON_STYLE                        // grey (custom_id field is required)
	SUCCESS_BUTTON_STYLE                          // green (custom_id field is required)
	DANGER_BUTTON_STYLE                           // red (custom_id field is required)
	LINK_BUTTON_STYLE                             // grey, navigate to URL (url field is required)
	PREMIUM_BUTTON_STYLE                          // By default same as primary but will automatically use SKU icon, name & price (sky_id field is required)
)

// https://docs.discord.com/developers/components/reference#component-object
type ComponentType uint8

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
	FILE_UPLOAD_COMPONENT_TYPE                                 // Layout component for Modals
	_                                                          //
	RADIO_GROUP_COMPONENT_TYPE                                 // Interactive component for Modals
	CHECKBOX_GROUP_COMPONENT_TYPE                              // Interactive component for Modals
	CHECKBOX_COMPONENT_TYPE                                    // Interactive component for Modals
)

// https://docs.discord.com/developers/components/reference#text-input-text-input-styles
type TextInputStyle uint8

const (
	SHORT_TEXT_INPUT_STYLE     TextInputStyle = iota + 1 // A single-line input.
	PARAGRAPH_TEXT_INPUT_STYLE                           // A multi-line input.
)

// https://docs.discord.com/developers/components/reference#user-select-select-default-value-structure
type DefaultValueType string

const (
	USER_DEFAULT_VALUE    DefaultValueType = "user"
	ROLE_DEFAULT_VALUE    DefaultValueType = "role"
	CHANNEL_DEFAULT_VALUE DefaultValueType = "channel"
)

// An ActionRowComponent groups other related components within a message or modal.
//
// https://docs.discord.com/developers/components/reference#action-row-action-row-structure
type ActionRowComponent struct {
	Components []ActionRowChildComponent `json:"components,omitzero"` // Up to 5 interactive [ButtonComponent]s or a single [SelectComponent]
	ID         uint32                    `json:"id,omitempty"`
	Type       ComponentType             `json:"type"` // Always = ACTION_ROW_COMPONENT_TYPE (1)
}

// https://docs.discord.com/developers/components/reference#anatomy-of-a-component
// type UnknownComponent struct {
// 	Type ComponentType `json:"type"`
// 	ID   uint32        `json:"id,omitempty"`
// }

// A ButtonComponent displays a clickable element that users can interact with,
// sending an interaction to your app when pressed.
//
// They must be placed within an [ActionRowComponent] or a [SectionComponent]'s accessory field,
// both of which are only valid inside messages.
//
// https://docs.discord.com/developers/components/reference#button
type ButtonComponent struct {
	Emoji *Emoji `json:"emoji,omitempty"` // It may only contain id, name, and animated from regular Emoji struct.
	Label string `json:"label,omitempty"`
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID string        `json:"custom_id,omitempty"`
	URL      string        `json:"url,omitempty"`
	SkuID    Snowflake     `json:"sku_id,omitempty"` // Identifier for a purchasable SKU, only available when using premium-style buttons. Premium buttons do not send an interaction to your app when clicked.
	ID       uint32        `json:"id,omitempty"`
	Type     ComponentType `json:"type"` // Always = BUTTON_COMPONENT_TYPE (2)
	Style    ButtonStyle   `json:"style"`
	Disabled bool          `json:"disabled"`
}

// A StringSelectComponent displays a dropdown menu for users to select one or more pre-defined options.
//
// They are available in both messages and modals, but
// must be placed inside an [ActionRowComponent] or [LabelComponent] respectively.
//
// https://docs.discord.com/developers/components/reference#string-select
type StringSelectComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID    string             `json:"custom_id,omitempty"`
	Placeholder string             `json:"placeholder,omitempty"`
	Options     []SelectMenuOption `json:"options,omitzero"`
	Values      []string           `json:"values,omitzero"` // The values of the options that were selected by the user. Should not be provided during message or modal creation.
	ID          uint32             `json:"id,omitempty"`
	Type        ComponentType      `json:"type"`                 // Always = STRING_SELECT_COMPONENT_TYPE (3). Omitted inside interaction responses from messages.
	MinValues   uint8              `json:"min_values,omitempty"` // The minimum number of options that must be chosen; defaults to 1 and must be between 0 and 25. Can only be 0 if required is set to false.
	MaxValues   uint8              `json:"max_values,omitempty"` // The maximum number of options that can be chosen; defaults to 1 and must be between 1 and 25.
	Disabled    bool               `json:"disabled"`             // Whether the select menu is disabled inside a message; default false. Will result in an error if used inside a modal!
	Required    bool               `json:"required"`             // Whether a selection is required to submit the modal; defaults to true. Will result in an error if used inside a message!

	// The following 2 fields are sent by Discord's API upon a successful interaction response.
	// They should not be sent by developers when sending a message or modal, being either ignored or causing runtime payload rejection.
	// Source: https://docs.discord.com/developers/components/reference#string-select-string-select-interaction-response-structure
	// TODO: Have someone try throwing an invalid payload with these fields at discord to see how they respond (and update the above comment accordingly)

	ComponentType ComponentType `json:"component_type,omitempty"` // Always = STRING_SELECT_COMPONENT_TYPE (3). Omitted in anything BUT interaction responses from messages.
}

// A SelectMenuOption represents a single option within a [StringSelectComponent].
//
// https://docs.discord.com/developers/components/reference#string-select-select-option-structure
type SelectMenuOption struct {
	Emoji       *Emoji `json:"emoji,omitempty"`       // It may only contain id, name, and animated from regular Emoji struct.
	Label       string `json:"label"`                 // Text label that appears on the option label, max 80 characters.
	Value       string `json:"value"`                 // Value to return back to app once clicked, max 100 characters.
	Description string `json:"description,omitempty"` // An additional description of the option, max 100 characters.
	Default     bool   `json:"default"`               // Whether to render this option as selected by default.
}

// A TextInputComponent displays a field for the user to input free-form text.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#text-input
type TextInputComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID    string         `json:"custom_id,omitempty"`
	Label       string         `json:"label,omitempty"`       // Deprecated: use `label` and `description` on a Label component instead
	Value       string         `json:"value,omitempty"`       // Pre-filled value for this component; max 4000 characters. Once the user submits the modal, this will be populated with their input.
	Placeholder string         `json:"placeholder,omitempty"` // Placeholder text to display when no text is present. Max: 100 characters
	ID          uint32         `json:"id,omitempty"`
	MinLength   uint16         `json:"min_length,omitempty"` // min: 0, max: 4000 characters
	MaxLength   uint16         `json:"max_length,omitempty"` // min: 1, max: 4000 characters
	Type        ComponentType  `json:"type"`                 // Always = TEXT_INPUT_COMPONENT_TYPE (4)
	Style       TextInputStyle `json:"style"`
	Required    bool           `json:"required"` // Whether this component is required to be filled, defaults to true
}

// A SelectComponent displays a drop-down menu for a user to select either Users, Roles, Mentionables or Channels.
// It encapsulates the [User Select], [Role Select], [Mentionable Select] and [Channel Select] components from Discord.
//
// They are available in both messages and modals, but
// must be placed inside an [ActionRowComponent] or [LabelComponent] respectively.
//
// [User Select]: https://docs.discord.com/developers/components/reference#user-select-user-select-structure
// [Role Select]: https://docs.discord.com/developers/components/reference#role-select-role-select-structure
// [Mentionable Select]: https://docs.discord.com/developers/components/reference#mentionable-select-mentionable-select-structure
// [Channel Select]: https://docs.discord.com/developers/components/reference#channel-select-channel-select-structure
type SelectComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID      string               `json:"custom_id,omitempty"`
	Placeholder   string               `json:"placeholder,omitempty"`   // Placeholder text if nothing is selected, max: 150 characters.
	ChannelTypes  []ChannelType        `json:"channel_types,omitzero"`  // List of channel types to include in the channel select component; should be omitted for all other select types.
	DefaultValues []DefaultValueOption `json:"default_values,omitzero"` // List of default values for auto-populated select menu components; must have between MinValues and MaxValues entries.
	ID            uint32               `json:"id,omitempty"`
	Type          ComponentType        `json:"type"`                 // Either USER_SELECT_COMPONENT_TYPE, ROLE_SELECT_COMPONENT_TYPE, MENTIONABLE_SELECT_COMPONENT_TYPE or CHANNEL_SELECT_COMPONENT_TYPE
	MinValues     uint8                `json:"min_values,omitempty"` // The minimum number of items that must be chosen; defaults to 1 and must be between 0 and 25.
	MaxValues     uint8                `json:"max_values,omitempty"` // The maximum number of items that can be chosen; defaults to 1 and must be between 0 and 25.
	Disabled      bool                 `json:"disabled"`             // Whether the select menu is disabled inside a message; default false. Will result in an error if used inside a modal!
}

// https://docs.discord.com/developers/components/reference#user-select-select-default-value-structure
type DefaultValueOption struct {
	Type DefaultValueType `json:"type"` // The type of value that the option represents. Should be consistent across all default value options.
	ID   Snowflake        `json:"id"`   // Snowflake ID of a user, role, or channel
}

// A SectionComponent associates text content with a given [AccessoryComponent].
//
// They can only be used inside messages.
//
// https://docs.discord.com/developers/components/reference#section-section-structure
type SectionComponent struct {
	Accessory  AccessoryComponent     `json:"accessory,omitempty"` // A component contextually associated to the section's content.
	Components []TextDisplayComponent `json:"components,omitzero"` // 1-3 text display components representing the section's text content.
	ID         uint32                 `json:"id,omitempty"`
	Type       ComponentType          `json:"type"` // Always = SECTION_COMPONENT_TYPE (9)
}

// A TextDisplayComponent displays Markdown-formatted text content within a message or modal,
// similar to the 'content' field of a message.
//
// https://docs.discord.com/developers/components/reference#text-display-text-display-structure
type TextDisplayComponent struct {
	Content string        `json:"content"` // The Markdown content to display.
	ID      uint32        `json:"id,omitempty"`
	Type    ComponentType `json:"type"` // Always = TEXT_DISPLAY_COMPONENT_TYPE (10)
}

// A ThumbnailComponent displays visual media as a small thumbnail within a message.
//
// They are only available as accessories inside [SectionComponent]s.
//
// https://docs.discord.com/developers/components/reference#thumbnail-thumbnail-structure
type ThumbnailComponent struct {
	Description string            `json:"description,omitempty"` // Alt text for the media, max 1024 characters
	Media       UnfurledMediaItem `json:"media"`
	ID          uint32            `json:"id,omitempty"`
	Type        ComponentType     `json:"type"` // Always = THUMBNAIL_COMPONENT_TYPE (11)
	Spoiler     bool              `json:"spoiler"`
}

// https://docs.discord.com/developers/components/reference#unfurled-media-item-structure
type UnfurledMediaItem struct {
	URL          string    `json:"url"` // 	Supports arbitrary urls and attachment://<filename> references
	ProxyURL     string    `json:"proxy_url,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`  // This field is ignored and provided by the API as part of the response
	AttachmentID Snowflake `json:"attachment_id,omitempty"` // This field is ignored and provided by the API as part of the response
	Width        uint32    `json:"width,omitempty"`
	Height       uint32    `json:"height,omitempty"`
}

// https://docs.discord.com/developers/components/reference#media-gallery-media-gallery-structure
type MediaGalleryComponent struct {
	Items []MediaGalleryItem `json:"items,omitzero"` // 1 to 10 media gallery items
	ID    uint32             `json:"id,omitempty"`
	Type  ComponentType      `json:"type"` // Always = MEDIA_GALLERY_COMPONENT_TYPE (12)
}

// https://docs.discord.com/developers/components/reference#media-gallery-media-gallery-item-structure
type MediaGalleryItem struct {
	Description string            `json:"description,omitempty"` // Alt text for the media, max 1024 characters
	Media       UnfurledMediaItem `json:"media"`
	Spoiler     bool              `json:"spoiler"`
}

// https://docs.discord.com/developers/components/reference#file-file-structure
type FileComponent struct {

	// Below 2 fields are controlled by API and should be readonly for us, developers.

	Name    string            `json:"name,omitempty"` // This field is ignored and provided by the API as part of the response.
	File    UnfurledMediaItem `json:"file"`           // This unfurled media item is unique in that it only supports attachment references using the attachment://<filename> syntax.
	ID      uint32            `json:"id,omitempty"`
	Size    uint32            `json:"size,omitempty"` // The size of the file in bytes. This field is ignored and provided by the API as part of the response.
	Type    ComponentType     `json:"type"`           // Always = FILE_COMPONENT_TYPE (13)
	Spoiler bool              `json:"spoiler"`
}

// https://docs.discord.com/developers/components/reference#separator-separator-structure
type SeparatorComponent struct {
	ID      uint32        `json:"id,omitempty"`
	Type    ComponentType `json:"type"`              // Always = SEPARATOR_COMPONENT_TYPE (14)
	Divider bool          `json:"divider"`           // Whether a visual divider should be displayed in the component (defaults to true).
	Spacing uint8         `json:"spacing,omitempty"` // Size of separator padding—1 for small padding, 2 for large padding (defaults to 1).
}

// A ContainerComponent visually encapsulates one or more components inside a message,
// alongside an optional accent color bar.
//
// They can only be used inside messages.
//
// https://docs.discord.com/developers/components/reference#container-container-structure
type ContainerComponent struct {
	Components  []ContainerChildComponent `json:"components,omitzero"` // Child components to nest within the container.
	ID          uint32                    `json:"id,omitempty"`
	AccentColor uint32                    `json:"accent_color,omitempty"` // The accent color of the container, as an RGB value from 0x000000 to 0xFFFFFF. (Write as a hex literal for best results.)
	Type        ComponentType             `json:"type"`                   // Always = CONTAINER_COMPONENT_TYPE (17)
	Spoiler     bool                      `json:"spoiler"`                // Whether to mark the container as a spoiler.
}

// A LabelComponent wraps a child component with text and an optional description.
//
// They can only be used inside modals.
//
// https://docs.discord.com/developers/components/reference#label
type LabelComponent struct {
	Component   LabelChildComponent `json:"component"`             // The component nested within the label.
	Label       string              `json:"label"`                 // The header text to show on the label; max 45 characters.
	Description string              `json:"description,omitempty"` // An additional description for the label; max 100 characters.
	ID          uint32              `json:"id,omitempty"`          // Optional identifier for component
	Type        ComponentType       `json:"type"`                  // Always = LABEL_COMPONENT_TYPE (18)
}

// A FileUploadComponent allows users to upload one or more files in modals.
// The maximum file size a user can upload is based solely on the user’s upload limit in that channel.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#file-upload
type FileUploadComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID string `json:"custom_id,omitempty"`
	// A list of max 10 (discord supported) file type extensions that you want this component to accept.
	//
	// For example: .png, .jpg, .qt, .mp3, .wav
	// It also supports 3 groups (by typing "image", "video" or "audio" as type, please do not hardcode these lists as they are subject to change):
	//
	// IMAGE_EXTENSIONS = ('.png', '.gif', '.jpg', '.jpeg', '.jfif', '.webp', '.avif')
	//
	// VIDEO_EXTENSIONS = ('.mp4', '.mov', '.qt', '.webm')
	//
	// AUDIO_EXTENSIONS = ('.mp3', '.m4a', '.wav', '.ogg', '.opus', '.flac')
	//
	// We recommend using the provided file groups. if you are specifying only extensions,
	// you must include .jpg for image uploads, and both .mp4 and .mov for video uploads,
	// due to mobile schenanigans
	//
	// This feature only checks the extension on the filename - it does not actually inspect
	// the contents of the file. You still need to make sure that the file is valid.
	FileTypes []string      `json:"file_types,omitzero"`
	ID        uint32        `json:"id,omitempty"`         // Optional identifier for component
	Type      ComponentType `json:"type"`                 // Always = FILE_UPLOAD_COMPONENT_TYPE (19)
	MinValues uint8         `json:"min_values,omitempty"` // The minimum number of files that must be uploaded; defaults to 1 and must be between 0 and 10. Can only be 0 if required is set to false.
	MaxValues uint8         `json:"max_values,omitempty"` // The maximum number of files that can be uploaded; defaults to 1 and must be between 1 and 10.
	Required  bool          `json:"required"`             // Whether a file upload is required to submit the modal.
}

// A RadioGroupComponent allows selecting exactly one option from a defined list of choices.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#radio-group
type RadioGroupComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID string             `json:"custom_id,omitempty"`
	Options  []RadioGroupOption `json:"options,omitzero"` // List of options to show; min 2, max 10
	ID       uint32             `json:"id,omitempty"`     // Optional identifier for component
	Type     ComponentType      `json:"type"`             // Always = RADIO_GROUP_COMPONENT_TYPE (21)
	Required bool               `json:"required"`         // Whether a selection is required to submit the modal (defaults to true)
}

// https://docs.discord.com/developers/components/reference#radio-group-option-structure
type RadioGroupOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Default     bool   `json:"default"` // Whether to render this option as selected by default.
}

// A CheckboxGroupComponent displays a list of one or more options selectable via checkboxes.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#checkbox-group
type CheckboxGroupComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID  string                `json:"custom_id,omitempty"`
	Options   []CheckboxGroupOption `json:"options,omitzero"`     // The options to show in this checkbox group; must be between 1 and 10 options.
	ID        uint32                `json:"id,omitempty"`         // Optional identifier for component
	Type      ComponentType         `json:"type"`                 // Always = CHECKBOX_GROUP_COMPONENT_TYPE (22)
	MinValues uint8                 `json:"min_values,omitempty"` // The minimum number of options that must be chosen; defaults to 1 and must be between 0 and 10. Can only be 0 if required is set to false.
	MaxValues uint8                 `json:"max_values,omitempty"` // The maximum number of options that can be chosen; must be between 1 and 10.
	Required  bool                  `json:"required"`             // Whether a selection is required to submit the modal (defaults to true).
}

// https://docs.discord.com/developers/components/reference#checkbox-group-option-structure
type CheckboxGroupOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Default     bool   `json:"default"` // Whether to render this option as selected by default.
}

// A CheckboxComponent displays a single, non-required yes/no style question.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#checkbox
type CheckboxComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID string        `json:"custom_id,omitempty"`
	ID       uint32        `json:"id,omitempty"` // Optional identifier for component
	Type     ComponentType `json:"type"`         // Always = CHECKBOX_COMPONENT_TYPE (23)
	Default  bool          `json:"default"`      // Whether to render this option as selected by default.
}
