package tempest

// Go doesn't support union types so we'll use partially hidden interfaces to create pseudo groups to duck type it.
// (please make pull request with better solution if you have any)

// AnyComponent is a union interface representing all possible Discord components.
// All component types must implement this interface.
//
// Avoid using this interface directly;
// instead look at its child interfaces -
// [MessageComponent], [ModalComponent], [LayoutComponent], [InteractiveComponent], [ContentComponent] and [AccessoryComponent].
type AnyComponent interface {
	_kind() ComponentType
}

// LabelChildComponent represents components that can be used as the child of a [LabelComponent].
// Only one child component is allowed per label.
//
// Currently valid types: [TextInputComponent], [StringSelectComponent] and [SelectComponent] (user/role/mentionable/channel selects).
//
// https://discord.com/developers/docs/components/reference#label-label-child-components
type LabelChildComponent interface {
	AnyComponent
	_lblcmp()
}

// ModalComponent represents components that can be used inside the top level of modals, such as Text Displays and Labels.
//
// See https://discord.com/developers/docs/components/reference#component-object-component-types for a complete list of valid types.
type ModalComponent interface {
	AnyComponent
	_modalcmp()
}

// MessageComponent represents components that can be used inside the top level of messages, such as Action Rows and Buttons.
//
// See https://discord.com/developers/docs/components/reference#component-object-component-types for a complete list of valid types.
type MessageComponent interface {
	AnyComponent
	_messagecmp()
}

// LayoutComponent represents message layout containers like Action Rows, Sections, Separators & Containers.
//
// These are used to control the final look of your custom message/embed.
type LayoutComponent interface {
	AnyComponent
	_lcmp()
}

// InteractiveComponent is a component that can trigger interactions, like buttons or select menus.
// These can be placed inside Action Rows or used as accessories in sections.
// Use when you need a component that triggers a callback or response.
type InteractiveComponent interface {
	AnyComponent
	_icmp()
}

// ContentComponent represents non-interactive visual components such as text, media, or files.
// These are used to display static content in sections or containers.
type ContentComponent interface {
	AnyComponent
	_ccmp()
}

// AccessoryComponent is a special subset of components that can be used as an accessory inside SectionComponent.
// Only one accessory is allowed per section.
//
// Currently valid types: ButtonComponent & ThumbnailComponent.
//
// https://discord.com/developers/docs/components/reference#section-section-structure
type AccessoryComponent interface {
	AnyComponent
	_acmp()
}

func (cmp ActionRowComponent) _kind() ComponentType { return cmp.Type }
func (cmp ActionRowComponent) _lcmp()               {}
func (cmp ActionRowComponent) _messagecmp()         {}

func (cmp ButtonComponent) _kind() ComponentType { return cmp.Type }
func (cmp ButtonComponent) _icmp()               {}
func (cmp ButtonComponent) _acmp()               {}
func (cmp ButtonComponent) _messagecmp()         {}

func (cmp StringSelectComponent) _kind() ComponentType { return cmp.Type }
func (cmp StringSelectComponent) _icmp()               {}
func (cmp StringSelectComponent) _lblcmp()             {}
func (cmp StringSelectComponent) _messagecmp()         {}
func (cmp StringSelectComponent) _modalcmp()           {}

func (cmp TextInputComponent) _kind() ComponentType { return cmp.Type }
func (cmp TextInputComponent) _icmp()               {}
func (cmp TextInputComponent) _lblcmp()             {}
func (cmp TextInputComponent) _modalcmp()           {}

func (cmp SelectComponent) _kind() ComponentType { return cmp.Type }
func (cmp SelectComponent) _icmp()               {}
func (cmp SelectComponent) _messagecmp()         {}
func (cmp SelectComponent) _modalcmp()           {}

func (cmp SectionComponent) _kind() ComponentType { return cmp.Type }
func (cmp SectionComponent) _lcmp()               {}
func (cmp SectionComponent) _messagecmp()         {}

func (cmp TextDisplayComponent) _kind() ComponentType { return cmp.Type }
func (cmp TextDisplayComponent) _ccmp()               {}
func (cmp TextDisplayComponent) _messagecmp()         {}
func (cmp TextDisplayComponent) _modalcmp()           {}

func (cmp ThumbnailComponent) _kind() ComponentType { return cmp.Type }
func (cmp ThumbnailComponent) _ccmp()               {}
func (cmp ThumbnailComponent) _acmp()               {}
func (cmp ThumbnailComponent) _messagecmp()         {}

func (cmp MediaGalleryComponent) _kind() ComponentType { return cmp.Type }
func (cmp MediaGalleryComponent) _ccmp()               {}
func (cmp MediaGalleryComponent) _messagecmp()         {}

func (cmp FileComponent) _kind() ComponentType { return cmp.Type }
func (cmp FileComponent) _ccmp()               {}
func (cmp FileComponent) _messagecmp()         {}

func (cmp SeparatorComponent) _kind() ComponentType { return cmp.Type }
func (cmp SeparatorComponent) _lcmp()               {}
func (cmp SeparatorComponent) _messagecmp()         {}

func (cmp ContainerComponent) _kind() ComponentType { return cmp.Type }
func (cmp ContainerComponent) _lcmp()               {}
func (cmp ContainerComponent) _messagecmp()         {}

func (cmp LabelComponent) _kind() ComponentType { return cmp.Type }
func (cmp LabelComponent) _lcmp()               {}
func (cmp LabelComponent) _modalcmp()           {}

// TODO: Add file upload components (which should not implement either message/modal interfaces since they can only go inside labels)
