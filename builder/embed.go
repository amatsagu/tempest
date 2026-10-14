package builder

import "github.com/amatsagu/tempest/discord"

func NewEmbed(title, description string) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
	}
}

func NewEmbedFooter(text string) discord.EmbedFooter {
	return discord.EmbedFooter{Text: text}
}

func NewEmbedAuthor(name string) discord.EmbedAuthor {
	return discord.EmbedAuthor{Name: name}
}

func NewEmbedImage(url string) discord.EmbedImage {
	return discord.EmbedImage{URL: url}
}

func NewEmbedThumbnail(url string) discord.EmbedThumbnail {
	return discord.EmbedThumbnail{URL: url}
}

func NewEmbedVideo(url string) discord.EmbedVideo {
	return discord.EmbedVideo{URL: url}
}

func NewEmbedProvider(name, url string) discord.EmbedProvider {
	return discord.EmbedProvider{Name: name, URL: url}
}

func NewEmbedField(name, value string, inline bool) discord.EmbedField {
	return discord.EmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}
