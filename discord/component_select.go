package discord

// https://docs.discord.com/developers/components/reference#user-select-select-default-value-structure
type SelectDefaultValueType string

const (
	SelectDefaultUser    SelectDefaultValueType = "user"
	SelectDefaultRole    SelectDefaultValueType = "role"
	SelectDefaultChannel SelectDefaultValueType = "channel"
)

// https://docs.discord.com/developers/components/reference#user-select-select-default-value-structure
type SelectDefaultValue struct {
	Type SelectDefaultValueType `json:"type"`
	ID   Snowflake              `json:"id"`
}

// https://docs.discord.com/developers/components/reference#string-select-select-option-structure
type SelectOption struct {
	Emoji       *Emoji `json:"emoji,omitzero"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitzero"`
	Default     bool   `json:"default,omitzero"`
}

// https://docs.discord.com/developers/components/reference#string-select-string-select-structure
type StringSelect struct {
	MinValues   *uint8         `json:"min_values,omitzero"`
	MaxValues   *uint8         `json:"max_values,omitzero"`
	CustomID    string         `json:"custom_id"`
	Placeholder string         `json:"placeholder,omitzero"`
	Options     []SelectOption `json:"options"`
	ID          uint32         `json:"id,omitzero"`
	Type        ComponentType  `json:"type"`
	Disabled    bool           `json:"disabled,omitzero"`
	Required    bool           `json:"required,omitzero"`
}

// https://docs.discord.com/developers/components/reference#user-select-user-select-structure
type SelectMenu struct {
	MinValues     *uint8               `json:"min_values,omitzero"`
	MaxValues     *uint8               `json:"max_values,omitzero"`
	CustomID      string               `json:"custom_id"`
	Placeholder   string               `json:"placeholder,omitzero"`
	ChannelTypes  []ChannelType        `json:"channel_types,omitzero"`
	DefaultValues []SelectDefaultValue `json:"default_values,omitzero"`
	ID            uint32               `json:"id,omitzero"`
	Type          ComponentType        `json:"type"`
	Disabled      bool                 `json:"disabled,omitzero"`
	Required      bool                 `json:"required,omitzero"`
}
