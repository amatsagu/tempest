# Message Components v2

Requires `Flags: tempest.IS_COMPONENTS_V2_MESSAGE_FLAG` in responses.

## Layouts
### SectionComponent (Type 9)
```go
type SectionComponent struct {
	Type       ComponentType
	Components []TextDisplayComponent
	Accessory  AccessoryComponent
}
```

### ContainerComponent (Type 17)
```go
type ContainerComponent struct {
	Type        ComponentType
	Components  []ContainerChildComponent
	AccentColor uint32
	Spoiler     bool
}
```

### ActionRowComponent (Type 1)
```go
type ActionRowComponent struct {
	Type       ComponentType
	Components []ActionRowChildComponent
}
```

## Elements
### ButtonComponent (Type 2)
```go
type ButtonComponent struct {
	Type     ComponentType
	Style    ButtonStyle
	Label    string
	Emoji    *Emoji
	CustomID string
	SkuID    Snowflake
	URL      string
	Disabled bool
}
```

### Select Components (Types 3, 5-8)
```go
type StringSelectComponent struct {
	Type        ComponentType
	CustomID    string
	Options     []SelectMenuOption
	Placeholder string
	MinValues   uint8
	MaxValues   uint8
	Disabled    bool
}

type SelectComponent struct {
	Type          ComponentType // USER, ROLE, MENTIONABLE, CHANNEL
	CustomID      string
	ChannelTypes  []ChannelType
	Placeholder   string
	DefaultValues []DefaultValueOption
	MinValues     uint8
	MaxValues     uint8
}
```

## Sub-structures
```go
type SelectMenuOption struct {
	Label       string
	Value       string
	Description string
	Emoji       *Emoji
	Default     bool
}

type TextDisplayComponent struct {
	Type    ComponentType
	Content string
}

type ThumbnailComponent struct {
	Type        ComponentType
	Media       UnfurledMediaItem
	Description string
	Spoiler     bool
}

type UnfurledMediaItem struct {
	URL      string
	ProxyURL string
	Width    uint32
	Height   uint32
}
```

## Constants
### ButtonStyle
```go
const (
	PRIMARY_BUTTON_STYLE ButtonStyle = iota + 1
	SECONDARY_BUTTON_STYLE
	SUCCESS_BUTTON_STYLE
	DANGER_BUTTON_STYLE
	LINK_BUTTON_STYLE
	PREMIUM_BUTTON_STYLE
)
```

## Helpers
```go
func FindInteractiveComponent[CC AnyComponent, T InteractiveComponent](components []CC, filter func(T) bool) (component T, found bool)
```
