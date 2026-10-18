package builder

import "github.com/amatsagu/tempest/discord"

func Embed(title, description string) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
	}
}

func EmbedFooter(text string) discord.EmbedFooter {
	return discord.EmbedFooter{Text: text}
}

func EmbedAuthor(name string) discord.EmbedAuthor {
	return discord.EmbedAuthor{Name: name}
}

func EmbedImage(url string) discord.EmbedImage {
	return discord.EmbedImage{URL: url}
}

func EmbedThumbnail(url string) discord.EmbedThumbnail {
	return discord.EmbedThumbnail{URL: url}
}

func EmbedVideo(url string) discord.EmbedVideo {
	return discord.EmbedVideo{URL: url}
}

func EmbedProvider(name, url string) discord.EmbedProvider {
	return discord.EmbedProvider{Name: name, URL: url}
}

func EmbedField(name, value string, inline bool) discord.EmbedField {
	return discord.EmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}
