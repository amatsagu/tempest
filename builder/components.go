// Package builder provides ergonomic, low-overhead constructors for the discord data model.
// Each builder returns the corresponding discord struct directly by value so payloads
// encode identically to handwritten struct literals.
package builder

import "github.com/amatsagu/tempest/discord"

// Groups up to 5 buttons or a single select menu inside a message.
func ActionRow(components ...discord.ActionRowChildComponent) discord.ActionRowComponent {
	return discord.ActionRowComponent{
		Type:       discord.ActionRowComponentType,
		Components: components,
	}
}

// Builds an interactive button component that dispatches an interaction via customID.
func Button(customID, label string, style discord.ButtonStyle) discord.ButtonComponent {
	return discord.ButtonComponent{
		Type:     discord.ButtonComponentType,
		Style:    style,
		Label:    label,
		CustomID: customID,
	}
}

// Builds a grey navigation button that opens url when clicked.
func LinkButton(url, label string) discord.ButtonComponent {
	return discord.ButtonComponent{
		Type:  discord.ButtonComponentType,
		Style: discord.LinkButtonStyle,
		Label: label,
		URL:   url,
	}
}

// Builds a premium monetization button tied to skuID.
func PremiumButton(skuID discord.SkuID) discord.ButtonComponent {
	return discord.ButtonComponent{
		Type:  discord.ButtonComponentType,
		Style: discord.PremiumButtonStyle,
		SkuID: skuID,
	}
}

// Builds a string dropdown menu for use inside an ActionRow or modal Label.
func StringSelect(customID, placeholder string, options ...discord.SelectMenuOption) discord.StringSelectComponent {
	return discord.StringSelectComponent{
		Type:        discord.StringSelectComponentType,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// Builds an option for a StringSelectComponent.
func SelectOption(label, value string) discord.SelectMenuOption {
	return discord.SelectMenuOption{
		Label: label,
		Value: value,
	}
}

func entitySelect(kind discord.ComponentType, customID, placeholder string, channelTypes []discord.ChannelType) discord.SelectComponent {
	return discord.SelectComponent{
		Type:         kind,
		CustomID:     customID,
		Placeholder:  placeholder,
		ChannelTypes: channelTypes,
	}
}

// Builds an auto-populated user select menu.
func UserSelect(customID, placeholder string) discord.SelectComponent {
	return entitySelect(discord.UserSelectComponentType, customID, placeholder, nil)
}

// Builds an auto-populated role select menu.
func RoleSelect(customID, placeholder string) discord.SelectComponent {
	return entitySelect(discord.RoleSelectComponentType, customID, placeholder, nil)
}

// Builds an auto-populated user and role select menu.
func MentionableSelect(customID, placeholder string) discord.SelectComponent {
	return entitySelect(discord.MentionableSelectComponentType, customID, placeholder, nil)
}

// Builds an auto-populated channel select menu filtered to channelTypes when provided.
func ChannelSelect(customID, placeholder string, channelTypes ...discord.ChannelType) discord.SelectComponent {
	return entitySelect(discord.ChannelSelectComponentType, customID, placeholder, channelTypes)
}

// Builds a default user selection entry for UserSelect or MentionableSelect.
func SelectDefaultUser(id discord.UserID) discord.DefaultValueOption {
	return discord.DefaultValueOption{
		Type: discord.UserDefaultValue,
		ID:   id.Snowflake,
	}
}

// Builds a default role selection entry for RoleSelect or MentionableSelect.
func SelectDefaultRole(id discord.RoleID) discord.DefaultValueOption {
	return discord.DefaultValueOption{
		Type: discord.RoleDefaultValue,
		ID:   id.Snowflake,
	}
}

// Builds a default channel selection entry for ChannelSelect.
func SelectDefaultChannel(id discord.ChannelID) discord.DefaultValueOption {
	return discord.DefaultValueOption{
		Type: discord.ChannelDefaultValue,
		ID:   id.Snowflake,
	}
}

// Builds a Components V2 section pairing text blocks with an accessory button or thumbnail.
func Section(accessory discord.AccessoryComponent, components ...discord.TextDisplayComponent) discord.SectionComponent {
	return discord.SectionComponent{
		Type:       discord.SectionComponentType,
		Accessory:  accessory,
		Components: components,
	}
}

// Builds a Components V2 markdown text component.
func TextDisplay(content string) discord.TextDisplayComponent {
	return discord.TextDisplayComponent{
		Type:    discord.TextDisplayComponentType,
		Content: content,
	}
}

// Builds a thumbnail accessory component for use inside a Section.
func Thumbnail(url, description string) discord.ThumbnailComponent {
	return discord.ThumbnailComponent{
		Type:        discord.ThumbnailComponentType,
		Description: description,
		Media:       discord.UnfurledMediaItem{URL: url},
	}
}

// Builds an unfurled media reference pointing to url or an attachment:// reference.
func UnfurledMediaItem(url string) discord.UnfurledMediaItem {
	return discord.UnfurledMediaItem{URL: url}
}

// Builds a Components V2 media gallery component.
func MediaGallery(items ...discord.MediaGalleryItem) discord.MediaGalleryComponent {
	return discord.MediaGalleryComponent{
		Type:  discord.MediaGalleryComponentType,
		Items: items,
	}
}

// Builds a single item for a MediaGalleryComponent.
func MediaGalleryItem(url string) discord.MediaGalleryItem {
	return discord.MediaGalleryItem{
		Media: discord.UnfurledMediaItem{URL: url},
	}
}

// Builds a Components V2 file component referencing url (typically attachment://<filename>).
func File(url string) discord.FileComponent {
	return discord.FileComponent{
		Type: discord.FileComponentType,
		File: discord.UnfurledMediaItem{URL: url},
	}
}

// Builds a Components V2 separator. Passing false hides the horizontal divider line;
// passing true omits "divider" so Discord applies its visible default without extra payload/allocs.
func Separator(divider bool) discord.SeparatorComponent {
	var d *bool
	if !divider {
		d = new(false)
	}
	return discord.SeparatorComponent{
		Type:    discord.SeparatorComponentType,
		Divider: d,
	}
}

// Builds a Components V2 container with an RGB accent color (0x000001-0xFFFFFF).
func Container(accentColor uint32, components ...discord.ContainerChildComponent) discord.ContainerComponent {
	return discord.ContainerComponent{
		Type:        discord.ContainerComponentType,
		AccentColor: accentColor,
		Components:  components,
	}
}

// Builds a Components V2 container without an accent bar.
func ContainerNoAccent(components ...discord.ContainerChildComponent) discord.ContainerComponent {
	return Container(0, components...)
}

// Builds a modal text input (required by default on Discord when Required is nil).
func TextInput(customID, placeholder string, style discord.TextInputStyle) discord.TextInputComponent {
	return discord.TextInputComponent{
		Type:        discord.TextInputComponentType,
		Style:       style,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

// Builds a single-line modal text input (required by default on Discord).
func ShortTextInput(customID, placeholder string) discord.TextInputComponent {
	return TextInput(customID, placeholder, discord.ShortTextInputStyle)
}

// Builds a multi-line modal text input (required by default on Discord).
func ParagraphTextInput(customID, placeholder string) discord.TextInputComponent {
	return TextInput(customID, placeholder, discord.ParagraphTextInputStyle)
}

// Wraps a modal child component with a header label.
func Label(label string, component discord.LabelChildComponent) discord.LabelComponent {
	return discord.LabelComponent{
		Type:      discord.LabelComponentType,
		Label:     label,
		Component: component,
	}
}

// Wraps a modal child component with a header label and helper description.
func LabelWithDescription(label, description string, component discord.LabelChildComponent) discord.LabelComponent {
	return discord.LabelComponent{
		Type:        discord.LabelComponentType,
		Label:       label,
		Description: description,
		Component:   component,
	}
}

// Builds a modal file upload component (required by default on Discord). fileTypes accepts Discord
// category keywords ("image", "video", "audio") or dot-prefixed extensions (".png"); empty accepts any file.
func FileUpload(customID string, fileTypes ...string) discord.FileUploadComponent {
	return discord.FileUploadComponent{
		Type:      discord.FileUploadComponentType,
		CustomID:  customID,
		FileTypes: fileTypes,
	}
}

// Builds a modal radio group component (required by default on Discord).
func RadioGroup(customID string, options ...discord.RadioGroupOption) discord.RadioGroupComponent {
	return discord.RadioGroupComponent{
		Type:     discord.RadioGroupComponentType,
		CustomID: customID,
		Options:  options,
	}
}

// Builds one option inside a RadioGroupComponent.
func RadioGroupOption(label, value string) discord.RadioGroupOption {
	return discord.RadioGroupOption{
		Value: value,
		Label: label,
	}
}

// Builds a modal checkbox group component (required by default on Discord).
func CheckboxGroup(customID string, options ...discord.CheckboxGroupOption) discord.CheckboxGroupComponent {
	return discord.CheckboxGroupComponent{
		Type:     discord.CheckboxGroupComponentType,
		CustomID: customID,
		Options:  options,
	}
}

// Builds one option inside a CheckboxGroupComponent.
func CheckboxGroupOption(label, value string) discord.CheckboxGroupOption {
	return discord.CheckboxGroupOption{
		Value: value,
		Label: label,
	}
}

// Builds a single boolean checkbox component for use inside a modal Label.
func Checkbox(customID string, defaultChecked bool) discord.CheckboxComponent {
	return discord.CheckboxComponent{
		Type:     discord.CheckboxComponentType,
		CustomID: customID,
		Default:  defaultChecked,
	}
}
