package util

import "github.com/amatsagu/tempest"

// ActionRow groups interactive components in a message.
func ActionRow(components ...tempest.ActionRowChildComponent) tempest.ActionRowComponent {
	return tempest.ActionRowComponent{
		Type:       tempest.ACTION_ROW_COMPONENT_TYPE,
		Components: components,
	}
}

// Button builds a custom-ID button.
func Button(customID, label string, style tempest.ButtonStyle) tempest.ButtonComponent {
	return tempest.ButtonComponent{
		Type:     tempest.BUTTON_COMPONENT_TYPE,
		Style:    style,
		Label:    label,
		CustomID: customID,
	}
}

// LinkButton builds a link button.
func LinkButton(url, label string) tempest.ButtonComponent {
	return tempest.ButtonComponent{
		Type:  tempest.BUTTON_COMPONENT_TYPE,
		Style: tempest.LINK_BUTTON_STYLE,
		Label: label,
		URL:   url,
	}
}

// StringSelect builds a string select menu.
func StringSelect(customID, placeholder string, options ...tempest.SelectMenuOption) tempest.StringSelectComponent {
	return tempest.StringSelectComponent{
		Type:        tempest.STRING_SELECT_COMPONENT_TYPE,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// SelectOption builds one string select option.
func SelectOption(label, value string) tempest.SelectMenuOption {
	return tempest.SelectMenuOption{Label: label, Value: value}
}

// TextDisplay builds a Components V2 text display.
func TextDisplay(content string) tempest.TextDisplayComponent {
	return tempest.TextDisplayComponent{
		Type:    tempest.TEXT_DISPLAY_COMPONENT_TYPE,
		Content: content,
	}
}

// Separator builds a Components V2 separator.
func Separator(divider bool) tempest.SeparatorComponent {
	return tempest.SeparatorComponent{
		Type:    tempest.SEPARATOR_COMPONENT_TYPE,
		Divider: divider,
	}
}

// Thumbnail builds a Components V2 thumbnail.
func Thumbnail(url, description string) tempest.ThumbnailComponent {
	return tempest.ThumbnailComponent{
		Type:        tempest.THUMBNAIL_COMPONENT_TYPE,
		Description: description,
		Media:       tempest.UnfurledMediaItem{URL: url},
	}
}

// SectionWithThumbnail builds a section whose accessory is a thumbnail.
func SectionWithThumbnail(thumbnail tempest.ThumbnailComponent, text ...tempest.TextDisplayComponent) tempest.SectionComponent {
	return tempest.SectionComponent{
		Type:       tempest.SECTION_COMPONENT_TYPE,
		Components: text,
		Accessory:  thumbnail,
	}
}

// MediaGalleryItem builds one gallery item.
func MediaGalleryItem(url string) tempest.MediaGalleryItem {
	return tempest.MediaGalleryItem{Media: tempest.UnfurledMediaItem{URL: url}}
}

// MediaGallery builds a Components V2 media gallery.
func MediaGallery(items ...tempest.MediaGalleryItem) tempest.MediaGalleryComponent {
	return tempest.MediaGalleryComponent{
		Type:  tempest.MEDIA_GALLERY_COMPONENT_TYPE,
		Items: items,
	}
}

// Container builds a Components V2 container.
func Container(accentColor uint32, components ...tempest.ContainerChildComponent) tempest.ContainerComponent {
	return tempest.ContainerComponent{
		Type:        tempest.CONTAINER_COMPONENT_TYPE,
		AccentColor: accentColor,
		Components:  components,
	}
}

// ModalResponse builds a modal response.
func ModalResponse(customID, title string, components ...tempest.ModalComponent) tempest.ResponseModalData {
	return tempest.ResponseModalData{
		CustomID:   customID,
		Title:      title,
		Components: components,
	}
}

// Label wraps a modal input with text.
func Label(label string, component tempest.LabelChildComponent) tempest.LabelComponent {
	return tempest.LabelComponent{
		Type:      tempest.LABEL_COMPONENT_TYPE,
		Label:     label,
		Component: component,
	}
}

// ShortTextInput builds a single-line text input.
func ShortTextInput(customID, placeholder string) tempest.TextInputComponent {
	return tempest.TextInputComponent{
		Type:        tempest.TEXT_INPUT_COMPONENT_TYPE,
		Style:       tempest.SHORT_TEXT_INPUT_STYLE,
		CustomID:    customID,
		Placeholder: placeholder,
		Required:    true,
	}
}

// ParagraphTextInput builds a multi-line text input.
func ParagraphTextInput(customID, placeholder string) tempest.TextInputComponent {
	return tempest.TextInputComponent{
		Type:        tempest.TEXT_INPUT_COMPONENT_TYPE,
		Style:       tempest.PARAGRAPH_TEXT_INPUT_STYLE,
		CustomID:    customID,
		Placeholder: placeholder,
		Required:    true,
	}
}

// FileUpload builds a modal file upload field.
func FileUpload(customID string, extensions ...string) tempest.FileUploadComponent {
	return tempest.FileUploadComponent{
		Type:      tempest.FILE_UPLOAD_COMPONENT_TYPE,
		CustomID:  customID,
		FileTypes: extensions,
		Required:  true,
	}
}
