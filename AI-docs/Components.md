# Message Components v2

New Discord layout system. Requires `Flags: tempest.IS_COMPONENTS_V2_MESSAGE_FLAG` in the response/message data.

## Layouts
### `ActionRowComponent` (Type 1)
Max 5 buttons or 1 select.
```go
type ActionRowComponent struct {
	Type       ComponentType             `json:"type"`
	Components []ActionRowChildComponent `json:"components,omitzero"`
}
```

### `SectionComponent` (Type 9)
Text + 1 accessory.
```go
type SectionComponent struct {
	Type       ComponentType          `json:"type"`
	Components []TextDisplayComponent `json:"components,omitzero"`
	Accessory  AccessoryComponent     `json:"accessory,omitempty"`
}
```

### `ContainerComponent` (Type 17)
Visual grouping + accent color.
```go
type ContainerComponent struct {
	Type        ComponentType             `json:"type"`
	Components  []ContainerChildComponent `json:"components,omitzero"`
	AccentColor uint32                    `json:"accent_color,omitempty"`
}
```

## Elements
### `ButtonComponent` (Type 2)
### `StringSelectComponent` (Type 3)

## Interfaces (Duck-typing)
- `MessageComponent`: Top-level message.
- `ModalComponent`: Top-level modal.
- `ActionRowChildComponent`: In ActionRow.
- `ContainerChildComponent`: In Container.
- `AccessoryComponent`: In Section (Button/Thumbnail).

## Helpers
```go
func FindInteractiveComponent[CC AnyComponent, T InteractiveComponent](components []CC, filter func(T) bool) (component T, found bool)
```
