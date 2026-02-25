# Modals

Input components must be inside `LabelComponent`.

## Required Wrapper
### LabelComponent (Type 18)
```go
type LabelComponent struct {
	Type        ComponentType
	Label       string
	Description string
	Component   LabelChildComponent
}
```

## Input Components
### TextInputComponent (Type 4)
```go
type TextInputComponent struct {
	Type        ComponentType
	CustomID    string
	Style       TextInputStyle
	Label       string
	MinLength   uint16
	MaxLength   uint16
	Required    bool
	Value       string
	Placeholder string
}
```

### FileUploadComponent (Type 19)
```go
type FileUploadComponent struct {
	Type      ComponentType
	CustomID  string
	MinValues uint8
	MaxValues uint8
	Required  bool
}
```

### RadioGroupComponent (Type 21)
```go
type RadioGroupComponent struct {
	Type     ComponentType
	CustomID string
	Options  []RadioGroupOption
	Required bool
}
```

### Checkbox Components (Types 22-23)
```go
type CheckboxGroupComponent struct {
	Type      ComponentType
	CustomID  string
	Options   []CheckboxGroupOption
	MinValues uint8
	MaxValues uint8
	Required  bool
}

type CheckboxComponent struct {
	Type     ComponentType
	CustomID string
	Default  bool
}
```

## Extraction
```go
func (itx *ModalInteraction) GetInputValue(customID string) string
```
