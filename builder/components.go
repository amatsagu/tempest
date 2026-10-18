// Package builder provides ergonomic constructors for the discord data model. Each
// builder fills the corresponding struct and encodes exactly like a literal.
package builder

import "github.com/amatsagu/tempest/discord"

func Button(customID, label string, style discord.ButtonStyle) discord.Button {
	return discord.Button{
		Type:     discord.ComponentButton,
		Style:    style,
		Label:    label,
		CustomID: customID,
	}
}

func LinkButton(url, label string) discord.Button {
	return discord.Button{
		Type:  discord.ComponentButton,
		Style: discord.ButtonLink,
		Label: label,
		URL:   url,
	}
}

func PremiumButton(skuID discord.SkuID) discord.Button {
	return discord.Button{
		Type:  discord.ComponentButton,
		Style: discord.ButtonPremium,
		SKUID: skuID,
	}
}

func TextDisplay(content string) discord.TextDisplay {
	return discord.TextDisplay{
		Type:    discord.ComponentTextDisplay,
		Content: content,
	}
}

func Thumbnail(url, description string) discord.Thumbnail {
	return discord.Thumbnail{
		Type: discord.ComponentThumbnail,
		Media: discord.UnfurledMediaItem{
			URL: url,
		},
		Description: description,
	}
}

func MediaGallery(items ...discord.MediaGalleryItem) discord.MediaGallery {
	return discord.MediaGallery{
		Type:  discord.ComponentMediaGallery,
		Items: items,
	}
}

func MediaGalleryItem(url string) discord.MediaGalleryItem {
	return discord.MediaGalleryItem{
		Media: discord.UnfurledMediaItem{URL: url},
	}
}

func UnfurledMediaItem(url string) discord.UnfurledMediaItem {
	return discord.UnfurledMediaItem{URL: url}
}

func FileComponent(url string) discord.FileComponent {
	return discord.FileComponent{
		Type: discord.ComponentFile,
		File: discord.UnfurledMediaItem{
			URL: url,
		},
	}
}

func ActionRow(buttons ...discord.Button) discord.ActionRow {
	return discord.ActionRow{
		Type:    discord.ComponentActionRow,
		Buttons: buttons,
	}
}

func ActionRowWithStringSelect(selectMenu discord.StringSelect) discord.ActionRow {
	return discord.ActionRow{
		Type:         discord.ComponentActionRow,
		StringSelect: &selectMenu,
	}
}

func ActionRowWithSelect(selectMenu discord.SelectMenu) discord.ActionRow {
	return discord.ActionRow{
		Type:       discord.ComponentActionRow,
		SelectMenu: &selectMenu,
	}
}

func Section(components ...discord.TextDisplay) discord.Section {
	return discord.Section{
		Type:       discord.ComponentSection,
		Components: components,
	}
}

func SectionWithButton(button discord.Button, components ...discord.TextDisplay) discord.Section {
	return discord.Section{
		Type:       discord.ComponentSection,
		Button:     &button,
		Components: components,
	}
}

func SectionWithThumbnail(thumbnail discord.Thumbnail, components ...discord.TextDisplay) discord.Section {
	return discord.Section{
		Type:       discord.ComponentSection,
		Thumbnail:  &thumbnail,
		Components: components,
	}
}

func Container(accentColor uint32, children ...discord.Component) discord.Container {
	return discord.Container{
		Type:        discord.ComponentContainer,
		AccentColor: accentColor,
		Components:  discord.Components(children),
	}
}

// ContainerNoAccent is Container(0, children...) because accent_color is omitted when zero.
func ContainerNoAccent(children ...discord.Component) discord.Container {
	return Container(0, children...)
}

func Separator(divider bool) discord.Separator {
	return discord.Separator{
		Type:    discord.ComponentSeparator,
		Divider: divider,
	}
}

func StringSelect(customID, placeholder string, options ...discord.SelectOption) discord.StringSelect {
	return discord.StringSelect{
		Type:        discord.ComponentStringSelect,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

func SelectOption(label, value string) discord.SelectOption {
	return discord.SelectOption{
		Label: label,
		Value: value,
	}
}

func UserSelect(customID, placeholder string) discord.SelectMenu {
	return discord.SelectMenu{
		Type:        discord.ComponentUserSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func RoleSelect(customID, placeholder string) discord.SelectMenu {
	return discord.SelectMenu{
		Type:        discord.ComponentRoleSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func MentionableSelect(customID, placeholder string) discord.SelectMenu {
	return discord.SelectMenu{
		Type:        discord.ComponentMentionableSelect,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func ChannelSelect(customID, placeholder string, channelTypes ...discord.ChannelType) discord.SelectMenu {
	return discord.SelectMenu{
		Type:         discord.ComponentChannelSelect,
		CustomID:     customID,
		Placeholder:  placeholder,
		ChannelTypes: channelTypes,
	}
}

func TextInput(customID, placeholder string, style discord.TextInputStyle) discord.TextInput {
	return discord.TextInput{
		Type:        discord.ComponentTextInput,
		Style:       style,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

func ShortTextInput(customID, placeholder string) discord.TextInput {
	return TextInput(customID, placeholder, discord.TextInputShort)
}

func ParagraphTextInput(customID, placeholder string) discord.TextInput {
	return TextInput(customID, placeholder, discord.TextInputParagraph)
}

func Label(label string, input discord.TextInput) discord.Label {
	return discord.Label{
		Type:      discord.ComponentLabel,
		Label:     label,
		TextInput: &input,
	}
}

func LabelWithStringSelect(label string, selectMenu discord.StringSelect) discord.Label {
	return discord.Label{
		Type:         discord.ComponentLabel,
		Label:        label,
		StringSelect: &selectMenu,
	}
}

func LabelWithSelect(label string, selectMenu discord.SelectMenu) discord.Label {
	return discord.Label{
		Type:       discord.ComponentLabel,
		Label:      label,
		SelectMenu: &selectMenu,
	}
}

func LabelWithRadioGroup(label string, radio discord.RadioGroup) discord.Label {
	return discord.Label{
		Type:       discord.ComponentLabel,
		Label:      label,
		RadioGroup: &radio,
	}
}

func LabelWithCheckboxGroup(label string, chk discord.CheckboxGroup) discord.Label {
	return discord.Label{
		Type:          discord.ComponentLabel,
		Label:         label,
		CheckboxGroup: &chk,
	}
}

func LabelWithCheckbox(label string, chk discord.Checkbox) discord.Label {
	return discord.Label{
		Type:     discord.ComponentLabel,
		Label:    label,
		Checkbox: &chk,
	}
}

func LabelWithFileUpload(label string, up discord.FileUpload) discord.Label {
	return discord.Label{
		Type:       discord.ComponentLabel,
		Label:      label,
		FileUpload: &up,
	}
}

func RadioGroup(customID string, options ...discord.RadioGroupOption) discord.RadioGroup {
	return discord.RadioGroup{
		Type:     discord.ComponentRadioGroup,
		CustomID: customID,
		Options:  options,
	}
}

func RadioGroupOption(label, value string) discord.RadioGroupOption {
	return discord.RadioGroupOption{
		Value: value,
		Label: label,
	}
}

func CheckboxGroup(customID string, options ...discord.CheckboxGroupOption) discord.CheckboxGroup {
	return discord.CheckboxGroup{
		Type:     discord.ComponentCheckboxGroup,
		CustomID: customID,
		Options:  options,
	}
}

func CheckboxGroupOption(label, value string) discord.CheckboxGroupOption {
	return discord.CheckboxGroupOption{
		Value: value,
		Label: label,
	}
}

func Checkbox(customID string, defaultChecked bool) discord.Checkbox {
	return discord.Checkbox{
		Type:     discord.ComponentCheckbox,
		CustomID: customID,
		Default:  defaultChecked,
	}
}

// FileUpload builds a modal file upload. fileTypes are category keywords ("image")
// or dot-prefixed extensions (".png"), never MIME types; empty accepts any file.
func FileUpload(customID string, fileTypes ...string) discord.FileUpload {
	return discord.FileUpload{
		Type:      discord.ComponentFileUpload,
		CustomID:  customID,
		FileTypes: fileTypes,
	}
}
