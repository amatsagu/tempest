package util

import "github.com/amatsagu/tempest/discord"

// ActionRow groups interactive components in a message.
func ActionRow(components ...discord.ActionRowChildComponent) discord.ActionRowComponent {
	return discord.ActionRowComponent{
		Type:       discord.ActionRowComponentType,
		Components: components,
	}
}

// Button builds a custom-ID button.
func Button(customID, label string, style discord.ButtonStyle) discord.ButtonComponent {
	return discord.ButtonComponent{
		Type:     discord.ButtonComponentType,
		Style:    style,
		Label:    label,
		CustomID: customID,
	}
}

// LinkButton builds a link button.
func LinkButton(url, label string) discord.ButtonComponent {
	return discord.ButtonComponent{
		Type:  discord.ButtonComponentType,
		Style: discord.LinkButtonStyle,
		Label: label,
		URL:   url,
	}
}

// StringSelect builds a string select menu.
func StringSelect(customID, placeholder string, options ...discord.SelectMenuOption) discord.StringSelectComponent {
	return discord.StringSelectComponent{
		Type:        discord.StringSelectComponentType,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// SelectOption builds one string select option.
func SelectOption(label, value string) discord.SelectMenuOption {
	return discord.SelectMenuOption{Label: label, Value: value}
}

// TextDisplay builds a Components V2 text display.
func TextDisplay(content string) discord.TextDisplayComponent {
	return discord.TextDisplayComponent{
		Type:    discord.TextDisplayComponentType,
		Content: content,
	}
}

// Separator builds a Components V2 separator. Passing false hides the divider line.
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

// Thumbnail builds a Components V2 thumbnail.
func Thumbnail(url, description string) discord.ThumbnailComponent {
	return discord.ThumbnailComponent{
		Type:        discord.ThumbnailComponentType,
		Description: description,
		Media:       discord.UnfurledMediaItem{URL: url},
	}
}

// SectionWithThumbnail builds a section whose accessory is a thumbnail.
func SectionWithThumbnail(thumbnail discord.ThumbnailComponent, text ...discord.TextDisplayComponent) discord.SectionComponent {
	return discord.SectionComponent{
		Type:       discord.SectionComponentType,
		Components: text,
		Accessory:  thumbnail,
	}
}

// MediaGalleryItem builds one gallery item.
func MediaGalleryItem(url string) discord.MediaGalleryItem {
	return discord.MediaGalleryItem{Media: discord.UnfurledMediaItem{URL: url}}
}

// MediaGallery builds a Components V2 media gallery.
func MediaGallery(items ...discord.MediaGalleryItem) discord.MediaGalleryComponent {
	return discord.MediaGalleryComponent{
		Type:  discord.MediaGalleryComponentType,
		Items: items,
	}
}

// Container builds a Components V2 container.
func Container(accentColor uint32, components ...discord.ContainerChildComponent) discord.ContainerComponent {
	return discord.ContainerComponent{
		Type:        discord.ContainerComponentType,
		AccentColor: accentColor,
		Components:  components,
	}
}

// ModalResponse builds a modal response.
func ModalResponse(customID, title string, components ...discord.ModalComponent) discord.ResponseModalData {
	return discord.ResponseModalData{
		CustomID:   customID,
		Title:      title,
		Components: components,
	}
}

// Label wraps a modal input with text.
func Label(label string, component discord.LabelChildComponent) discord.LabelComponent {
	return discord.LabelComponent{
		Type:      discord.LabelComponentType,
		Label:     label,
		Component: component,
	}
}

// ShortTextInput builds a single-line text input (required by default on Discord).
func ShortTextInput(customID, placeholder string) discord.TextInputComponent {
	return discord.TextInputComponent{
		Type:        discord.TextInputComponentType,
		Style:       discord.ShortTextInputStyle,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

// ParagraphTextInput builds a multi-line text input (required by default on Discord).
func ParagraphTextInput(customID, placeholder string) discord.TextInputComponent {
	return discord.TextInputComponent{
		Type:        discord.TextInputComponentType,
		Style:       discord.ParagraphTextInputStyle,
		CustomID:    customID,
		Placeholder: placeholder,
	}
}

// FileUpload builds a modal file upload field (required by default on Discord).
func FileUpload(customID string, extensions ...string) discord.FileUploadComponent {
	return discord.FileUploadComponent{
		Type:      discord.FileUploadComponentType,
		CustomID:  customID,
		FileTypes: extensions,
	}
}
