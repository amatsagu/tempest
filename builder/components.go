// Package builder provides ergonomic constructors for the discord data model.
// Every builder is a thin wrapper that fills the corresponding discord struct;
// nothing here changes the wire format, so a built value encodes exactly like a
// hand-written literal.
package builder

import "github.com/amatsagu/tempest/discord"

// Buttons.

func NewButton(customID, label string, style discord.ButtonStyle) discord.Button {
	return discord.Button{
		Type:     discord.ComponentButton,
		Style:    style,
		Label:    label,
		CustomID: customID,
	}
}

func NewLinkButton(url, label string) discord.Button {
	return discord.Button{
		Type:  discord.ComponentButton,
		Style: discord.ButtonLink,
		Label: label,
		URL:   url,
	}
}

func NewPremiumButton(skuID discord.SkuID) discord.Button {
	return discord.Button{
		Type:  discord.ComponentButton,
		Style: discord.ButtonPremium,
		SKUID: skuID,
	}
}

// Content components.

func NewTextDisplay(content string) discord.TextDisplay {
	return discord.TextDisplay{
		Type:    discord.ComponentTextDisplay,
		Content: content,
	}
}

func NewThumbnail(url, description string) discord.Thumbnail {
	return discord.Thumbnail{
		Type: discord.ComponentThumbnail,
		Media: discord.UnfurledMediaItem{
			URL: url,
		},
		Description: description,
	}
}

func NewMediaGallery(items ...discord.MediaGalleryItem) discord.MediaGallery {
	return discord.MediaGallery{
		Type:  discord.ComponentMediaGallery,
		Items: items,
	}
}

func NewMediaGalleryItem(url string) discord.MediaGalleryItem {
	return discord.MediaGalleryItem{
		Media: discord.UnfurledMediaItem{URL: url},
	}
}

func NewUnfurledMediaItem(url string) discord.UnfurledMediaItem {
	return discord.UnfurledMediaItem{URL: url}
}

func NewFileComponent(url string) discord.FileComponent {
	return discord.FileComponent{
		Type: discord.ComponentFile,
		File: discord.UnfurledMediaItem{
			URL: url,
		},
	}
}

// Layout components.

func NewActionRow(buttons ...discord.Button) discord.ActionRow {
	return discord.ActionRow{
		Type:    discord.ComponentActionRow,
		Buttons: buttons,
	}
}

func NewActionRowWithStringSelect(selectMenu discord.StringSelect) discord.ActionRow {
	return discord.ActionRow{
		Type:         discord.ComponentActionRow,
		StringSelect: &selectMenu,
	}
}

func NewActionRowWithSelect(selectMenu discord.SelectMenu) discord.ActionRow {
	return discord.ActionRow{
		Type:       discord.ComponentActionRow,
		SelectMenu: &selectMenu,
	}
}

func NewSection(components ...discord.TextDisplay) discord.Section {
	return discord.Section{
		Type:       discord.ComponentSection,
		Components: components,
	}
}

func NewSectionWithButton(button discord.Button, components ...discord.TextDisplay) discord.Section {
	return discord.Section{
		Type:       discord.ComponentSection,
		Button:     &button,
		Components: components,
	}
}

func NewSectionWithThumbnail(thumbnail discord.Thumbnail, components ...discord.TextDisplay) discord.Section {
	return discord.Section{
		Type:       discord.ComponentSection,
		Thumbnail:  &thumbnail,
		Components: components,
	}
}

func NewContainer(accentColor uint32, children ...discord.Component) discord.Container {
	return discord.Container{
		Type:        discord.ComponentContainer,
		AccentColor: accentColor,
		Components:  discord.Components(children),
	}
}

// NewContainerNoAccent builds a container with no accent color. It is equivalent
// to NewContainer(0, children...) because accent_color is omitted when zero.
func NewContainerNoAccent(children ...discord.Component) discord.Container {
	return NewContainer(0, children...)
}

func NewSeparator(divider bool) discord.Separator {
	return discord.Separator{
		Type:    discord.ComponentSeparator,
		Divider: divider,
	}
}

// Select components.

func NewStringSelect(customID, placeholder string, options ...discord.SelectOption) discord.StringSelect {
	return discord.StringSelect{
		Type:        discord.ComponentStringSelect,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

func NewSelectOption(label, value string) discord.SelectOption {
	return discord.SelectOption{
		Label: label,
		Value: value,
	}
}

func NewUserSelect(customID, placeholder string) discord.SelectMenu {
	return discord.SelectMenu{
		Type:        discord.ComponentUserSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewRoleSelect(customID, placeholder string) discord.SelectMenu {
	return discord.SelectMenu{
		Type:        discord.ComponentRoleSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewMentionableSelect(customID, placeholder string) discord.SelectMenu {
	return discord.SelectMenu{
		Type:        discord.ComponentMentionableSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewChannelSelect(customID, placeholder string, channelTypes ...discord.ChannelType) discord.SelectMenu {
	return discord.SelectMenu{
		Type:         discord.ComponentChannelSelect,
		CustomID:     customID,
		Placeholder:  placeholder,
		ChannelTypes: channelTypes,
	}
}

// Text input and modal components.

func NewTextInput(customID, placeholder string, style discord.TextInputStyle) discord.TextInput {
	return discord.TextInput{
		Type:        discord.ComponentTextInput,
		Style:       style,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func NewShortTextInput(customID, placeholder string) discord.TextInput {
	return NewTextInput(customID, placeholder, discord.TextInputShort)
}

func NewParagraphTextInput(customID, placeholder string) discord.TextInput {
	return NewTextInput(customID, placeholder, discord.TextInputParagraph)
}

func NewLabel(label string, input discord.TextInput) discord.Label {
	return discord.Label{
		Type:      discord.ComponentLabel,
		Label:     label,
		TextInput: &input,
	}
}

func NewLabelWithStringSelect(label string, selectMenu discord.StringSelect) discord.Label {
	return discord.Label{
		Type:         discord.ComponentLabel,
		Label:        label,
		StringSelect: &selectMenu,
	}
}

func NewLabelWithSelect(label string, selectMenu discord.SelectMenu) discord.Label {
	return discord.Label{
		Type:       discord.ComponentLabel,
		Label:      label,
		SelectMenu: &selectMenu,
	}
}

func NewLabelWithRadioGroup(label string, radio discord.RadioGroup) discord.Label {
	return discord.Label{
		Type:       discord.ComponentLabel,
		Label:      label,
		RadioGroup: &radio,
	}
}

func NewLabelWithCheckboxGroup(label string, chk discord.CheckboxGroup) discord.Label {
	return discord.Label{
		Type:          discord.ComponentLabel,
		Label:         label,
		CheckboxGroup: &chk,
	}
}

func NewLabelWithCheckbox(label string, chk discord.Checkbox) discord.Label {
	return discord.Label{
		Type:     discord.ComponentLabel,
		Label:    label,
		Checkbox: &chk,
	}
}

func NewLabelWithFileUpload(label string, up discord.FileUpload) discord.Label {
	return discord.Label{
		Type:       discord.ComponentLabel,
		Label:      label,
		FileUpload: &up,
	}
}

func NewRadioGroup(customID string, options ...discord.RadioGroupOption) discord.RadioGroup {
	return discord.RadioGroup{
		Type:     discord.ComponentRadioGroup,
		CustomID: customID,
		Options:  options,
	}
}

func NewRadioGroupOption(label, value string) discord.RadioGroupOption {
	return discord.RadioGroupOption{
		Value: value,
		Label: label,
	}
}

func NewCheckboxGroup(customID string, options ...discord.CheckboxGroupOption) discord.CheckboxGroup {
	return discord.CheckboxGroup{
		Type:     discord.ComponentCheckboxGroup,
		CustomID: customID,
		Options:  options,
	}
}

func NewCheckboxGroupOption(label, value string) discord.CheckboxGroupOption {
	return discord.CheckboxGroupOption{
		Value: value,
		Label: label,
	}
}

func NewCheckbox(customID string, defaultChecked bool) discord.Checkbox {
	return discord.Checkbox{
		Type:     discord.ComponentCheckbox,
		CustomID: customID,
		Default:  defaultChecked,
	}
}

func NewFileUpload(customID string, fileTypes ...string) discord.FileUpload {
	return discord.FileUpload{
		Type:      discord.ComponentFileUpload,
		CustomID:  customID,
		FileTypes: fileTypes,
	}
}
