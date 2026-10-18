package builder

import (
	"encoding/json/v2"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func ptr[T any](v T) *T { return &v }

func buildEmbedViaHelpers() discord.Embed {
	emb := Embed("Release", "v1.0.0")
	emb.URL = "https://rinellebot.com"
	emb.Color = 0xffd484

	emb.Footer = ptr(EmbedFooter("Atelier"))
	emb.Author = ptr(EmbedAuthor("Rinelle"))
	emb.Image = ptr(EmbedImage("https://cdn.discordapp.com/c.png"))
	emb.Thumbnail = ptr(EmbedThumbnail("https://cdn.discordapp.com/d.png"))
	emb.Provider = ptr(EmbedProvider("Rinelle", "https://rinellebot.com"))
	emb.Video = ptr(EmbedVideo("https://cdn.discordapp.com/e.mp4"))

	emb.Fields = []discord.EmbedField{
		EmbedField("Color roles", "45", true),
		EmbedField("Badges", "30", true),
		EmbedField("Uptime", "99.9%", false),
	}

	return emb
}

func buildEmbedDirect() discord.Embed {
	return discord.Embed{
		Title:       "Release",
		Description: "v1.0.0",
		URL:         "https://rinellebot.com",
		Color:       0xffd484,
		Footer:      &discord.EmbedFooter{Text: "Atelier"},
		Author:      &discord.EmbedAuthor{Name: "Rinelle"},
		Image:       &discord.EmbedImage{URL: "https://cdn.discordapp.com/c.png"},
		Thumbnail:   &discord.EmbedThumbnail{URL: "https://cdn.discordapp.com/d.png"},
		Provider:    &discord.EmbedProvider{Name: "Rinelle", URL: "https://rinellebot.com"},
		Video:       &discord.EmbedVideo{URL: "https://cdn.discordapp.com/e.mp4"},
		Fields: []discord.EmbedField{
			{Name: "Color roles", Value: "45", Inline: true},
			{Name: "Badges", Value: "30", Inline: true},
			{Name: "Uptime", Value: "99.9%", Inline: false},
		},
	}
}

func TestEmbedBuildersMatchDirect(t *testing.T) {
	assertSameJSON(t, buildEmbedViaHelpers(), buildEmbedDirect())

	assertSameJSON(t, EmbedFooter("t"), discord.EmbedFooter{Text: "t"})
	assertSameJSON(t, EmbedAuthor("a"), discord.EmbedAuthor{Name: "a"})
	assertSameJSON(t, EmbedImage("u"), discord.EmbedImage{URL: "u"})
	assertSameJSON(t, EmbedThumbnail("u"), discord.EmbedThumbnail{URL: "u"})
	assertSameJSON(t, EmbedVideo("u"), discord.EmbedVideo{URL: "u"})
	assertSameJSON(t, EmbedProvider("n", "u"), discord.EmbedProvider{Name: "n", URL: "u"})
	assertSameJSON(t, EmbedField("n", "v", true), discord.EmbedField{Name: "n", Value: "v", Inline: true})
}

func TestEmbedRoundTrip(t *testing.T) {
	raw, err := json.Marshal(ptr(buildEmbedViaHelpers()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed discord.Embed
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.Title != "Release" || len(parsed.Fields) != 3 || parsed.Footer == nil || parsed.Author == nil || parsed.Provider == nil {
		t.Errorf("round-trip mismatch: %+v", parsed)
	}
}

func BenchmarkEmbed_JSON(b *testing.B) {
	emb := buildEmbedViaHelpers()
	data, _ := json.Marshal(&emb)

	var parsed discord.Embed
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&emb)
	}
}

func BenchmarkBuildEmbed_Helpers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkEmbed = buildEmbedViaHelpers()
	}
}

func BenchmarkBuildEmbed_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkEmbed = buildEmbedDirect()
	}
}

var sinkEmbed discord.Embed
