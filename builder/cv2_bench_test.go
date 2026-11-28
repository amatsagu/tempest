package builder_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

func buildSimpleCV2Builder() discord.ResponseMessage {
	return builder.MessageResponse("", discord.IsComponentsV2MessageFlag,
		builder.Container(0x5865f2,
			builder.TextDisplay("## Notice\nYour session is ready."),
			builder.Separator(true),
			builder.ActionRow(
				builder.Button("continue", "Continue", discord.PrimaryButtonStyle),
				builder.Button("cancel", "Cancel", discord.SecondaryButtonStyle),
			),
		),
	)
}

func buildSimpleCV2Raw() discord.ResponseMessage {
	return discord.ResponseMessage{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{
			Flags: discord.IsComponentsV2MessageFlag,
			Components: []discord.MessageComponent{
				discord.ContainerComponent{
					Type:        discord.ContainerComponentType,
					AccentColor: 0x5865f2,
					Components: []discord.ContainerChildComponent{
						discord.TextDisplayComponent{
							Type:    discord.TextDisplayComponentType,
							Content: "## Notice\nYour session is ready.",
						},
						discord.SeparatorComponent{
							Type: discord.SeparatorComponentType,
						},
						discord.ActionRowComponent{
							Type: discord.ActionRowComponentType,
							Components: []discord.ActionRowChildComponent{
								discord.ButtonComponent{
									Type:     discord.ButtonComponentType,
									Style:    discord.PrimaryButtonStyle,
									Label:    "Continue",
									CustomID: "continue",
								},
								discord.ButtonComponent{
									Type:     discord.ButtonComponentType,
									Style:    discord.SecondaryButtonStyle,
									Label:    "Cancel",
									CustomID: "cancel",
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildComplexCV2Builder() discord.ResponseMessage {
	sepLarge := builder.Separator(false)
	sepLarge.Spacing = discord.LargeSeparatorSpacing

	channelPick := builder.ChannelSelect("target_channel", "Route alerts to",
		discord.GuildTextChannelType,
		discord.GuildAnnouncementChannelType,
	)
	channelPick.DefaultValues = []discord.DefaultValueOption{
		builder.SelectDefaultChannel(discord.ChannelID{Snowflake: 123456789012345678}),
	}

	return builder.MessageResponse("", discord.IsComponentsV2MessageFlag|discord.EphemeralMessageFlag,
		builder.Container(0xffd484,
			builder.Section(
				builder.Thumbnail("https://cdn.discordapp.com/embed/avatars/0.png", "Service status"),
				builder.TextDisplay("## Cluster Dashboard"),
				builder.TextDisplay("Monitoring primary and backup shards in real time."),
				builder.TextDisplay("-# Updated seconds ago"),
			),
			builder.Separator(true),
			builder.Section(
				builder.Button("refresh", "Refresh", discord.SecondaryButtonStyle),
				builder.TextDisplay("**Shard Latency**\nEurope West: `18ms`\nUS East: `41ms`"),
			),
			sepLarge,
			builder.MediaGallery(
				builder.MediaGalleryItem("https://cdn.discordapp.com/a.png"),
				builder.MediaGalleryItem("https://cdn.discordapp.com/b.png"),
				discord.MediaGalleryItem{
					Media:       builder.UnfurledMediaItem("https://cdn.discordapp.com/c.png"),
					Description: "Throughput chart",
					Spoiler:     true,
				},
			),
			builder.File("attachment://diagnostics.json"),
			builder.ActionRow(
				builder.Button("deploy", "Deploy", discord.SuccessButtonStyle),
				builder.Button("rollback", "Rollback", discord.DangerButtonStyle),
				builder.LinkButton("https://discord.com/developers/docs", "Docs"),
				builder.PremiumButton(discord.SkuID{Snowflake: 987654321098765432}),
			),
			builder.ActionRow(
				builder.StringSelect("region", "Switch region",
					builder.SelectOption("Frankfurt", "eu-central"),
					builder.SelectOption("Ashburn", "us-east"),
					discord.SelectMenuOption{Label: "Tokyo", Value: "ap-northeast", Description: "Low ping APAC", Default: true},
				),
			),
		),
		builder.ContainerNoAccent(
			builder.TextDisplay("### Routing"),
			builder.ActionRow(channelPick),
		),
	)
}

func buildComplexCV2Raw() discord.ResponseMessage {
	return discord.ResponseMessage{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{
			Flags: discord.IsComponentsV2MessageFlag | discord.EphemeralMessageFlag,
			Components: []discord.MessageComponent{
				discord.ContainerComponent{
					Type:        discord.ContainerComponentType,
					AccentColor: 0xffd484,
					Components: []discord.ContainerChildComponent{
						discord.SectionComponent{
							Type: discord.SectionComponentType,
							Accessory: discord.ThumbnailComponent{
								Type:        discord.ThumbnailComponentType,
								Description: "Service status",
								Media:       discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/embed/avatars/0.png"},
							},
							Components: []discord.TextDisplayComponent{
								{Type: discord.TextDisplayComponentType, Content: "## Cluster Dashboard"},
								{Type: discord.TextDisplayComponentType, Content: "Monitoring primary and backup shards in real time."},
								{Type: discord.TextDisplayComponentType, Content: "-# Updated seconds ago"},
							},
						},
						discord.SeparatorComponent{Type: discord.SeparatorComponentType},
						discord.SectionComponent{
							Type: discord.SectionComponentType,
							Accessory: discord.ButtonComponent{
								Type:     discord.ButtonComponentType,
								Style:    discord.SecondaryButtonStyle,
								Label:    "Refresh",
								CustomID: "refresh",
							},
							Components: []discord.TextDisplayComponent{
								{Type: discord.TextDisplayComponentType, Content: "**Shard Latency**\nEurope West: `18ms`\nUS East: `41ms`"},
							},
						},
						discord.SeparatorComponent{Type: discord.SeparatorComponentType, Divider: new(false), Spacing: discord.LargeSeparatorSpacing},
						discord.MediaGalleryComponent{
							Type: discord.MediaGalleryComponentType,
							Items: []discord.MediaGalleryItem{
								{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"}},
								{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/b.png"}},
								{
									Media:       discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/c.png"},
									Description: "Throughput chart",
									Spoiler:     true,
								},
							},
						},
						discord.FileComponent{
							Type: discord.FileComponentType,
							File: discord.UnfurledMediaItem{URL: "attachment://diagnostics.json"},
						},
						discord.ActionRowComponent{
							Type: discord.ActionRowComponentType,
							Components: []discord.ActionRowChildComponent{
								discord.ButtonComponent{Type: discord.ButtonComponentType, Style: discord.SuccessButtonStyle, Label: "Deploy", CustomID: "deploy"},
								discord.ButtonComponent{Type: discord.ButtonComponentType, Style: discord.DangerButtonStyle, Label: "Rollback", CustomID: "rollback"},
								discord.ButtonComponent{Type: discord.ButtonComponentType, Style: discord.LinkButtonStyle, Label: "Docs", URL: "https://discord.com/developers/docs"},
								discord.ButtonComponent{Type: discord.ButtonComponentType, Style: discord.PremiumButtonStyle, SkuID: discord.SkuID{Snowflake: 987654321098765432}},
							},
						},
						discord.ActionRowComponent{
							Type: discord.ActionRowComponentType,
							Components: []discord.ActionRowChildComponent{
								discord.StringSelectComponent{
									Type:        discord.StringSelectComponentType,
									CustomID:    "region",
									Placeholder: "Switch region",
									Options: []discord.SelectMenuOption{
										{Label: "Frankfurt", Value: "eu-central"},
										{Label: "Ashburn", Value: "us-east"},
										{Label: "Tokyo", Value: "ap-northeast", Description: "Low ping APAC", Default: true},
									},
								},
							},
						},
					},
				},
				discord.ContainerComponent{
					Type: discord.ContainerComponentType,
					Components: []discord.ContainerChildComponent{
						discord.TextDisplayComponent{Type: discord.TextDisplayComponentType, Content: "### Routing"},
						discord.ActionRowComponent{
							Type: discord.ActionRowComponentType,
							Components: []discord.ActionRowChildComponent{
								discord.SelectComponent{
									Type:         discord.ChannelSelectComponentType,
									CustomID:     "target_channel",
									Placeholder:  "Route alerts to",
									ChannelTypes: []discord.ChannelType{discord.GuildTextChannelType, discord.GuildAnnouncementChannelType},
									DefaultValues: []discord.DefaultValueOption{
										{Type: discord.ChannelDefaultValue, ID: 123456789012345678},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildDynamicCV2Builder(title, summary, actionID, regionPlaceholder string, accent uint32) discord.ResponseMessage {
	return builder.MessageResponse("", discord.IsComponentsV2MessageFlag,
		builder.Container(accent,
			builder.Section(
				builder.Button(actionID, "Refresh", discord.SecondaryButtonStyle),
				builder.TextDisplay(title),
				builder.TextDisplay(summary),
			),
			builder.Separator(true),
			builder.ActionRow(
				builder.StringSelect(actionID+":region", regionPlaceholder,
					builder.SelectOption("Europe", "eu"),
					builder.SelectOption("North America", "na"),
				),
			),
		),
	)
}

func buildDynamicCV2Raw(title, summary, actionID, regionPlaceholder string, accent uint32) discord.ResponseMessage {
	return discord.ResponseMessage{
		Type: discord.ChannelMessageWithSourceResponseType,
		Data: &discord.ResponseMessageData{
			Flags: discord.IsComponentsV2MessageFlag,
			Components: []discord.MessageComponent{
				discord.ContainerComponent{
					Type:        discord.ContainerComponentType,
					AccentColor: accent,
					Components: []discord.ContainerChildComponent{
						discord.SectionComponent{
							Type: discord.SectionComponentType,
							Accessory: discord.ButtonComponent{
								Type:     discord.ButtonComponentType,
								Style:    discord.SecondaryButtonStyle,
								Label:    "Refresh",
								CustomID: actionID,
							},
							Components: []discord.TextDisplayComponent{
								{Type: discord.TextDisplayComponentType, Content: title},
								{Type: discord.TextDisplayComponentType, Content: summary},
							},
						},
						discord.SeparatorComponent{Type: discord.SeparatorComponentType},
						discord.ActionRowComponent{
							Type: discord.ActionRowComponentType,
							Components: []discord.ActionRowChildComponent{
								discord.StringSelectComponent{
									Type:        discord.StringSelectComponentType,
									CustomID:    actionID + ":region",
									Placeholder: regionPlaceholder,
									Options: []discord.SelectMenuOption{
										{Label: "Europe", Value: "eu"},
										{Label: "North America", Value: "na"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestCV2PanelsMatchRaw(t *testing.T) {
	t.Run("simple cv2 panel", func(t *testing.T) {
		assertSameJSON(t, buildSimpleCV2Builder(), buildSimpleCV2Raw())
	})
	t.Run("complex cv2 panel", func(t *testing.T) {
		assertSameJSON(t, buildComplexCV2Builder(), buildComplexCV2Raw())
	})
	t.Run("dynamic cv2 panel", func(t *testing.T) {
		assertSameJSON(t,
			buildDynamicCV2Builder("## Node", "Latency: 14ms", "node_1", "Select region", 0x5865f2),
			buildDynamicCV2Raw("## Node", "Latency: 14ms", "node_1", "Select region", 0x5865f2),
		)
	})
}

var (
	sinkResp    discord.ResponseMessage
	sinkPoll    discord.Poll
	sinkJSON    []byte
	benchTitle  = "## Node"
	benchSub    = "Latency: 14ms"
	benchID     = "node_1"
	benchHolder = "Select region"
	benchAccent = uint32(0x5865f2)
)

func BenchmarkSimpleCV2_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResp = buildSimpleCV2Builder()
	}
}

func BenchmarkSimpleCV2_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResp = buildSimpleCV2Raw()
	}
}

func BenchmarkSimpleCV2_Marshal_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildSimpleCV2Builder()
		var err error
		sinkJSON, err = json.Marshal(&resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimpleCV2_Marshal_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildSimpleCV2Raw()
		var err error
		sinkJSON, err = json.Marshal(&resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComplexCV2_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResp = buildComplexCV2Builder()
	}
}

func BenchmarkComplexCV2_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResp = buildComplexCV2Raw()
	}
}

func BenchmarkComplexCV2_Marshal_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildComplexCV2Builder()
		var err error
		sinkJSON, err = json.Marshal(&resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComplexCV2_Marshal_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildComplexCV2Raw()
		var err error
		sinkJSON, err = json.Marshal(&resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDynamicCV2_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResp = buildDynamicCV2Builder(benchTitle, benchSub, benchID, benchHolder, benchAccent)
	}
}

func BenchmarkDynamicCV2_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResp = buildDynamicCV2Raw(benchTitle, benchSub, benchID, benchHolder, benchAccent)
	}
}

func BenchmarkDynamicCV2_Marshal_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildDynamicCV2Builder(benchTitle, benchSub, benchID, benchHolder, benchAccent)
		var err error
		sinkJSON, err = json.Marshal(&resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDynamicCV2_Marshal_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildDynamicCV2Raw(benchTitle, benchSub, benchID, benchHolder, benchAccent)
		var err error
		sinkJSON, err = json.Marshal(&resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPoll_Builder(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkPoll = buildPollWithBuilder()
	}
}

func BenchmarkPoll_Raw(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkPoll = buildPollRaw()
	}
}
