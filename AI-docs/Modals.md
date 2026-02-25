# Modals

Structured input. Components must be inside `LabelComponent`.

## Structs
### `ResponseModalData`
```go
type ResponseModalData struct {
	CustomID   string           `json:"custom_id"`
	Title      string           `json:"title"`
	Components []ModalComponent `json:"components"`
}
```

### `LabelComponent` (Type 18)
Required input wrapper.
```go
type LabelComponent struct {
	Type        ComponentType       `json:"type"` // 18
	Label       string              `json:"label"`
	Description string              `json:"description,omitempty"`
	Component   LabelChildComponent `json:"component"`
}
```

## Input Components
`LabelChildComponent` implementations:
- `TextInputComponent`
- `StringSelectComponent`
- `SelectComponent` (User, Role, etc.)
- `FileUploadComponent`
- `RadioGroupComponent`
- `CheckboxGroupComponent`
- `CheckboxComponent`

## Accessing Data
```go
func (itx *ModalInteraction) GetInputValue(customID string) string
```
