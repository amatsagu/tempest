package builder

import "github.com/amatsagu/tempest/discord"

// Builds a rich message embed with title and description.
func Embed(title, description string) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
	}
}

// Builds an embed footer block.
func EmbedFooter(text string) discord.EmbedFooter {
	return discord.EmbedFooter{Text: text}
}

// Builds an embed author block.
func EmbedAuthor(name string) discord.EmbedAuthor {
	return discord.EmbedAuthor{Name: name}
}

// Builds an embed image attachment/URL block.
func EmbedImage(url string) discord.EmbedImage {
	return discord.EmbedImage{URL: url}
}

// Builds an embed thumbnail image block.
func EmbedThumbnail(url string) discord.EmbedThumbnail {
	return discord.EmbedThumbnail{URL: url}
}

// Builds an embed video media block.
func EmbedVideo(url string) discord.EmbedVideo {
	return discord.EmbedVideo{URL: url}
}

// Builds an embed provider metadata block.
func EmbedProvider(name, url string) discord.EmbedProvider {
	return discord.EmbedProvider{Name: name, URL: url}
}

// Builds a single name/value field entry for an embed.
func EmbedField(name, value string, inline bool) discord.EmbedField {
	return discord.EmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}
