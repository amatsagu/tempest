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

func NewStringSelect(customID, placeholder string, options ...SelectOption) StringSelect {
	return StringSelect{
		Type:        ComponentStringSelect,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

func NewUserSelect(customID, placeholder string) SelectMenu {
	return SelectMenu{
		Type:        ComponentUserSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewRoleSelect(customID, placeholder string) SelectMenu {
	return SelectMenu{
		Type:        ComponentRoleSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewMentionableSelect(customID, placeholder string) SelectMenu {
	return SelectMenu{
		Type:        ComponentMentionableSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewChannelSelect(customID, placeholder string, channelTypes ...ChannelType) SelectMenu {
	return SelectMenu{
		Type:         ComponentChannelSelect,
		CustomID:     customID,
		Placeholder:  placeholder,
		ChannelTypes: channelTypes,
	}
}
