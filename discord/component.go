package discord

// https://docs.discord.com/developers/components/reference#button-button-styles
type ButtonStyle uint8

const (
	PrimaryButtonStyle   ButtonStyle = iota + 1 // blurple (custom_id field is required)
	SecondaryButtonStyle                        // grey (custom_id field is required)
	SuccessButtonStyle                          // green (custom_id field is required)
	DangerButtonStyle                           // red (custom_id field is required)
	LinkButtonStyle                             // grey, navigate to URL (url field is required)
	PremiumButtonStyle                          // By default same as primary but will automatically use SKU icon, name & price (sku_id field is required)
)

// https://docs.discord.com/developers/components/reference#component-object
type ComponentType uint8

const (
	ActionRowComponentType         ComponentType = iota + 1 // Layout component for Messages
	ButtonComponentType                                     // Interactive component for Messages
	StringSelectComponentType                               // Interactive component for Messages and Modals
	TextInputComponentType                                  // Interactive component for Modals
	UserSelectComponentType                                 // Interactive component for Messages
	RoleSelectComponentType                                 // Interactive component for Messages
	MentionableSelectComponentType                          // Interactive component for Messages
	ChannelSelectComponentType                              // Interactive component for Messages
	SectionComponentType                                    // Layout component for Messages
	TextDisplayComponentType                                // Content component for Messages
	ThumbnailComponentType                                  // Content component for Messages
	MediaGalleryComponentType                               // Content component for Messages
	FileComponentType                                       // Content component for Messages
	SeparatorComponentType                                  // Layout component for Messages
	_                                                       //
	_                                                       //
	ContainerComponentType                                  // Layout component for Messages
	LabelComponentType                                      // Layout component for Modals
	FileUploadComponentType                                 // Layout component for Modals
	_                                                       //
	RadioGroupComponentType                                 // Interactive component for Modals
	CheckboxGroupComponentType                              // Interactive component for Modals
	CheckboxComponentType                                   // Interactive component for Modals
)

// https://docs.discord.com/developers/components/reference#text-input-text-input-styles
type TextInputStyle uint8

const (
	ShortTextInputStyle     TextInputStyle = iota + 1 // A single-line input.
	ParagraphTextInputStyle                           // A multi-line input.
)

// https://docs.discord.com/developers/components/reference#user-select-select-default-value-structure
type DefaultValueType string

const (
	UserDefaultValue    DefaultValueType = "user"
	RoleDefaultValue    DefaultValueType = "role"
	ChannelDefaultValue DefaultValueType = "channel"
)

// https://docs.discord.com/developers/components/reference#separator-separator-structure
type SeparatorSpacing uint8

const (
	SmallSeparatorSpacing SeparatorSpacing = iota + 1 // Small vertical padding (default).
	LargeSeparatorSpacing                             // Large vertical padding.
)

// An ActionRowComponent groups other related components within a message or modal.
//
// https://docs.discord.com/developers/components/reference#action-row-action-row-structure
type ActionRowComponent struct {
	Components []ActionRowChildComponent `json:"components,omitzero"` // Up to 5 interactive [ButtonComponent]s or a single [SelectComponent]
	ID         uint32                    `json:"id,omitzero"`
	Type       ComponentType             `json:"type"` // Always = [ActionRowComponentType] (1)
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
// https://docs.discord.com/developers/components/reference#button-button-structure
type ButtonComponent struct {
	Emoji *Emoji `json:"emoji,omitzero"` // It may only contain id, name, and animated from regular Emoji struct.
	Label string `json:"label,omitzero"`
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	CustomID string        `json:"custom_id,omitzero"`
	URL      string        `json:"url,omitzero"`
	SkuID    SkuID         `json:"sku_id,omitzero"` // Identifier for a purchasable SKU, only available when using premium-style buttons. Premium buttons do not send an interaction to your app when clicked.
	ID       uint32        `json:"id,omitzero"`
	Type     ComponentType `json:"type"` // Always = [ButtonComponentType] (2)
	Style    ButtonStyle   `json:"style"`
	Disabled bool          `json:"disabled,omitzero"`
}

// A StringSelectComponent displays a dropdown menu for users to select one or more pre-defined options.
//
// They are available in both messages and modals, but
// must be placed inside an [ActionRowComponent] or [LabelComponent] respectively.
//
// https://docs.discord.com/developers/components/reference#string-select-string-select-structure
type StringSelectComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	MinValues     *uint8             `json:"min_values,omitzero"` // The minimum number of options that must be chosen; defaults to 1 and must be between 0 and 25. Can only be 0 if required is set to false.
	Required      *bool              `json:"required,omitzero"`   // Whether a selection is required to submit the modal (defaults to true when nil/omitted). Only valid inside modals; leave nil in messages!
	CustomID      string             `json:"custom_id,omitzero"`
	Placeholder   string             `json:"placeholder,omitzero"`
	Options       []SelectMenuOption `json:"options,omitzero"`
	Values        []string           `json:"values,omitzero"` // The values of the options that were selected by the user. Should not be provided during message or modal creation.
	ID            uint32             `json:"id,omitzero"`
	Type          ComponentType      `json:"type"`                    // Always = [StringSelectComponentType] (3). Omitted inside interaction responses from messages.
	ComponentType ComponentType      `json:"component_type,omitzero"` // Always = [StringSelectComponentType] (3). Omitted in anything BUT interaction responses from messages.
	MaxValues     uint8              `json:"max_values,omitzero"`     // The maximum number of options that can be chosen; defaults to 1 and must be between 1 and 25.
	Disabled      bool               `json:"disabled,omitzero"`       // Whether the select menu is disabled inside a message; default false. Will result in an error if used inside a modal!
}

// A SelectMenuOption represents a single option within a [StringSelectComponent].
//
// https://docs.discord.com/developers/components/reference#string-select-select-option-structure
type SelectMenuOption struct {
	Emoji       *Emoji `json:"emoji,omitzero"`       // It may only contain id, name, and animated from regular Emoji struct.
	Label       string `json:"label"`                // Text label that appears on the option label, max 80 characters.
	Value       string `json:"value"`                // Value to return back to app once clicked, max 100 characters.
	Description string `json:"description,omitzero"` // An additional description of the option, max 100 characters.
	Default     bool   `json:"default,omitzero"`     // Whether to render this option as selected by default.
}

// A TextInputComponent displays a field for the user to input free-form text.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#text-input-text-input-structure
type TextInputComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	Required    *bool          `json:"required,omitzero"` // Whether this component is required to be filled (defaults to true when nil/omitted).
	CustomID    string         `json:"custom_id,omitzero"`
	Label       string         `json:"label,omitzero"`       // Deprecated: use `label` and `description` on a Label component instead
	Value       string         `json:"value,omitzero"`       // Pre-filled value for this component; max 4000 characters. Once the user submits the modal, this will be populated with their input.
	Placeholder string         `json:"placeholder,omitzero"` // Placeholder text to display when no text is present. Max: 100 characters
	ID          uint32         `json:"id,omitzero"`
	MinLength   uint16         `json:"min_length,omitzero"` // min: 0, max: 4000 characters
	MaxLength   uint16         `json:"max_length,omitzero"` // min: 1, max: 4000 characters
	Type        ComponentType  `json:"type"`                // Always = [TextInputComponentType] (4)
	Style       TextInputStyle `json:"style"`
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
	MinValues     *uint8               `json:"min_values,omitzero"` // The minimum number of items that must be chosen; defaults to 1 and must be between 0 and 25.
	Required      *bool                `json:"required,omitzero"`   // Whether a selection is required to submit the modal (defaults to true when nil/omitted). Only valid inside modals; leave nil in messages!
	CustomID      string               `json:"custom_id,omitzero"`
	Placeholder   string               `json:"placeholder,omitzero"`    // Placeholder text if nothing is selected, max: 150 characters.
	ChannelTypes  []ChannelType        `json:"channel_types,omitzero"`  // List of channel types to include in the channel select component; should be omitted for all other select types.
	DefaultValues []DefaultValueOption `json:"default_values,omitzero"` // List of default values for auto-populated select menu components; must have between MinValues and MaxValues entries.
	Values        []Snowflake          `json:"values,omitzero"`
	ID            uint32               `json:"id,omitzero"`
	MaxValues     uint8                `json:"max_values,omitzero"` // The maximum number of items that can be chosen; defaults to 1 and must be between 0 and 25.
	Type          ComponentType        `json:"type"`                // Either [UserSelectComponentType], [RoleSelectComponentType], [MentionableSelectComponentType] or [ChannelSelectComponentType]
	Disabled      bool                 `json:"disabled,omitzero"`   // Whether the select menu is disabled inside a message; default false. Will result in an error if used inside a modal!
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
	Accessory  AccessoryComponent     `json:"accessory,omitzero"`  // A component contextually associated to the section's content.
	Components []TextDisplayComponent `json:"components,omitzero"` // 1-3 text display components representing the section's text content.
	ID         uint32                 `json:"id,omitzero"`
	Type       ComponentType          `json:"type"` // Always = [SectionComponentType] (9)
}

// A TextDisplayComponent displays Markdown-formatted text content within a message or modal,
// similar to the 'content' field of a message.
//
// https://docs.discord.com/developers/components/reference#text-display-text-display-structure
type TextDisplayComponent struct {
	Content string        `json:"content"` // The Markdown content to display.
	ID      uint32        `json:"id,omitzero"`
	Type    ComponentType `json:"type"` // Always = [TextDisplayComponentType] (10)
}

// A ThumbnailComponent displays visual media as a small thumbnail within a message.
//
// They are only available as accessories inside [SectionComponent]s.
//
// https://docs.discord.com/developers/components/reference#thumbnail-thumbnail-structure
type ThumbnailComponent struct {
	Description string            `json:"description,omitzero"` // Alt text for the media, max 1024 characters
	Media       UnfurledMediaItem `json:"media"`
	ID          uint32            `json:"id,omitzero"`
	Type        ComponentType     `json:"type"` // Always = [ThumbnailComponentType] (11)
	Spoiler     bool              `json:"spoiler,omitzero"`
}

// https://docs.discord.com/developers/components/reference#unfurled-media-item-unfurled-media-item-flags
type UnfurledMediaItemFlags = BitSet

const (
	IsAnimatedUnfurledMediaItemFlag UnfurledMediaItemFlags = 1 << 0
)

// https://docs.discord.com/developers/components/reference#unfurled-media-item-unfurled-media-item-structure
type UnfurledMediaItem struct {
	URL                string                 `json:"url"` // Supports arbitrary urls and attachment://<filename> references
	ProxyURL           string                 `json:"proxy_url,omitzero"`
	ContentType        string                 `json:"content_type,omitzero"` // This field is ignored and provided by the API as part of the response
	Placeholder        string                 `json:"placeholder,omitzero"`
	AttachmentID       AttachmentID           `json:"attachment_id,omitzero"` // This field is ignored and provided by the API as part of the response
	Width              uint32                 `json:"width,omitzero"`
	Height             uint32                 `json:"height,omitzero"`
	PlaceholderVersion uint32                 `json:"placeholder_version,omitzero"`
	Flags              UnfurledMediaItemFlags `json:"flags,omitzero"`
}

// https://docs.discord.com/developers/components/reference#media-gallery-media-gallery-structure
type MediaGalleryComponent struct {
	Items []MediaGalleryItem `json:"items,omitzero"` // 1 to 10 media gallery items
	ID    uint32             `json:"id,omitzero"`
	Type  ComponentType      `json:"type"` // Always = [MediaGalleryComponentType] (12)
}

// https://docs.discord.com/developers/components/reference#media-gallery-media-gallery-item-structure
type MediaGalleryItem struct {
	Description string            `json:"description,omitzero"` // Alt text for the media, max 1024 characters
	Media       UnfurledMediaItem `json:"media"`
	Spoiler     bool              `json:"spoiler,omitzero"`
}

// https://docs.discord.com/developers/components/reference#file-file-structure
type FileComponent struct {
	// Below 2 fields are controlled by API and should be readonly for us, developers.

	Name    string            `json:"name,omitzero"` // This field is ignored and provided by the API as part of the response.
	File    UnfurledMediaItem `json:"file"`          // This unfurled media item is unique in that it only supports attachment references using the attachment://<filename> syntax.
	ID      uint32            `json:"id,omitzero"`
	Size    uint32            `json:"size,omitzero"` // The size of the file in bytes. This field is ignored and provided by the API as part of the response.
	Type    ComponentType     `json:"type"`          // Always = [FileComponentType] (13)
	Spoiler bool              `json:"spoiler,omitzero"`
}

// https://docs.discord.com/developers/components/reference#separator-separator-structure
type SeparatorComponent struct {
	Divider *bool            `json:"divider,omitzero"` // Whether a visual divider should be displayed in the component (defaults to true when nil/omitted).
	ID      uint32           `json:"id,omitzero"`
	Type    ComponentType    `json:"type"`             // Always = [SeparatorComponentType] (14)
	Spacing SeparatorSpacing `json:"spacing,omitzero"` // Size of separator padding—[SmallSeparatorSpacing] (1) or [LargeSeparatorSpacing] (2); defaults to 1.
}

// A ContainerComponent visually encapsulates one or more components inside a message,
// alongside an optional accent color bar.
//
// They can only be used inside messages.
//
// https://docs.discord.com/developers/components/reference#container-container-structure
type ContainerComponent struct {
	Components  []ContainerChildComponent `json:"components,omitzero"` // Child components to nest within the container.
	ID          uint32                    `json:"id,omitzero"`
	AccentColor uint32                    `json:"accent_color,omitzero"` // The accent color of the container, as an RGB value from 0x000000 to 0xFFFFFF. (Write as a hex literal for best results.)
	Type        ComponentType             `json:"type"`                  // Always = [ContainerComponentType] (17)
	Spoiler     bool                      `json:"spoiler,omitzero"`      // Whether to mark the container as a spoiler.
}

// A LabelComponent wraps a child component with text and an optional description.
//
// They can only be used inside modals.
//
// https://docs.discord.com/developers/components/reference#label-label-structure
type LabelComponent struct {
	Component   LabelChildComponent `json:"component"`            // The component nested within the label.
	Label       string              `json:"label"`                // The header text to show on the label; max 45 characters.
	Description string              `json:"description,omitzero"` // An additional description for the label; max 100 characters.
	ID          uint32              `json:"id,omitzero"`          // Optional identifier for component
	Type        ComponentType       `json:"type"`                 // Always = [LabelComponentType] (18)
}

// A FileUploadComponent allows users to upload one or more files in modals.
// The maximum file size a user can upload is based solely on the user’s upload limit in that channel.
//
// They can only be used inside [LabelComponent]s within modals.
//
// https://docs.discord.com/developers/components/reference#file-upload-file-upload-structure
type FileUploadComponent struct {
	// A unique, developer-defined identifier for this Component; must be between 1 and 100 characters long.
	//
	// Will be returned verbatim inside the response payload, and be used to maintain application state or store data as needed.
	MinValues *uint8 `json:"min_values,omitzero"` // The minimum number of files that must be uploaded; defaults to 1 and must be between 0 and 10. Can only be 0 if required is false.
	Required  *bool  `json:"required,omitzero"`   // Whether a file upload is required to submit the modal (defaults to true when nil/omitted).
	CustomID  string `json:"custom_id,omitzero"`
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
	FileTypes []string       `json:"file_types,omitzero"`
	Values    []AttachmentID `json:"values,omitzero"`     // IDs of the uploaded files found in the resolved data. Populated on modal submission.
	ID        uint32         `json:"id,omitzero"`         // Optional identifier for component
	Type      ComponentType  `json:"type"`                // Always = [FileUploadComponentType] (19)
	MaxValues uint8          `json:"max_values,omitzero"` // The maximum number of files that can be uploaded; defaults to 1 and must be between 1 and 10.
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
	Value    *string            `json:"value,omitzero"`    // The value of the selected option, or null if no option is selected. Populated on modal submission.
	Required *bool              `json:"required,omitzero"` // Whether a selection is required to submit the modal (defaults to true when nil/omitted).
	CustomID string             `json:"custom_id,omitzero"`
	Options  []RadioGroupOption `json:"options,omitzero"` // List of options to show; min 2, max 10
	ID       uint32             `json:"id,omitzero"`      // Optional identifier for component
	Type     ComponentType      `json:"type"`             // Always = [RadioGroupComponentType] (21)
}

// https://docs.discord.com/developers/components/reference#radio-group-option-structure
type RadioGroupOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitzero"`
	Default     bool   `json:"default,omitzero"` // Whether to render this option as selected by default.
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
	MinValues *uint8                `json:"min_values,omitzero"` // The minimum number of options that must be chosen; defaults to 1 and must be between 0 and 10. Can only be 0 if required is false.
	Required  *bool                 `json:"required,omitzero"`   // Whether a selection is required to submit the modal (defaults to true when nil/omitted).
	CustomID  string                `json:"custom_id,omitzero"`
	Options   []CheckboxGroupOption `json:"options,omitzero"`    // The options to show in this checkbox group; must be between 1 and 10 options.
	Values    []string              `json:"values,omitzero"`     // The values of the selected options. Populated on modal submission.
	ID        uint32                `json:"id,omitzero"`         // Optional identifier for component
	Type      ComponentType         `json:"type"`                // Always = [CheckboxGroupComponentType] (22)
	MaxValues uint8                 `json:"max_values,omitzero"` // The maximum number of options that can be chosen; must be between 1 and 10.
}

// https://docs.discord.com/developers/components/reference#checkbox-group-option-structure
type CheckboxGroupOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitzero"`
	Default     bool   `json:"default,omitzero"` // Whether to render this option as selected by default.
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
	CustomID string        `json:"custom_id,omitzero"`
	ID       uint32        `json:"id,omitzero"`      // Optional identifier for component
	Type     ComponentType `json:"type"`             // Always = [CheckboxComponentType] (23)
	Default  bool          `json:"default,omitzero"` // Whether to render this option as selected by default.
	Value    bool          `json:"value,omitzero"`   // The state of the checkbox (true if checked, false if unchecked). Populated on modal submission.
}
