package builder_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

func buildEmbedWithBuilder() discord.Embed {
	emb := builder.Embed("Release", "v2.0.0")
	emb.URL = "https://example.com"
	emb.Color = 0xffd484
	emb.Footer = new(builder.EmbedFooter("Tempest"))
	emb.Author = new(builder.EmbedAuthor("Bot"))
	emb.Image = new(builder.EmbedImage("https://cdn.discordapp.com/image.png"))
	emb.Thumbnail = new(builder.EmbedThumbnail("https://cdn.discordapp.com/thumb.png"))
	emb.Provider = new(builder.EmbedProvider("Tempest", "https://example.com"))
	emb.Video = new(builder.EmbedVideo("https://cdn.discordapp.com/video.mp4"))
	emb.Fields = []discord.EmbedField{
		builder.EmbedField("Users", "42", true),
		builder.EmbedField("Channels", "12", true),
		builder.EmbedField("Status", "Online", false),
	}

	return emb
}

func buildEmbedRaw() discord.Embed {
	return discord.Embed{
		Title:       "Release",
		Description: "v2.0.0",
		URL:         "https://example.com",
		Color:       0xffd484,
		Footer:      &discord.EmbedFooter{Text: "Tempest"},
		Author:      &discord.EmbedAuthor{Name: "Bot"},
		Image:       &discord.EmbedImage{URL: "https://cdn.discordapp.com/image.png"},
		Thumbnail:   &discord.EmbedThumbnail{URL: "https://cdn.discordapp.com/thumb.png"},
		Provider:    &discord.EmbedProvider{Name: "Tempest", URL: "https://example.com"},
		Video:       &discord.EmbedVideo{URL: "https://cdn.discordapp.com/video.mp4"},
		Fields: []discord.EmbedField{
			{Name: "Users", Value: "42", Inline: true},
			{Name: "Channels", Value: "12", Inline: true},
			{Name: "Status", Value: "Online", Inline: false},
		},
	}
}

func TestEmbedBuildersMatchRaw(t *testing.T) {
	assertSameJSON(t, buildEmbedWithBuilder(), buildEmbedRaw())
	assertSameJSON(t, builder.EmbedFooter("f"), discord.EmbedFooter{Text: "f"})
	assertSameJSON(t, builder.EmbedAuthor("a"), discord.EmbedAuthor{Name: "a"})
	assertSameJSON(t, builder.EmbedImage("u"), discord.EmbedImage{URL: "u"})
	assertSameJSON(t, builder.EmbedThumbnail("u"), discord.EmbedThumbnail{URL: "u"})
	assertSameJSON(t, builder.EmbedVideo("u"), discord.EmbedVideo{URL: "u"})
	assertSameJSON(t, builder.EmbedProvider("n", "u"), discord.EmbedProvider{Name: "n", URL: "u"})
	assertSameJSON(t, builder.EmbedField("k", "v", true), discord.EmbedField{Name: "k", Value: "v", Inline: true})
}

func buildPollWithBuilder() discord.Poll {
	return builder.Poll("Favorite language?", 24,
		builder.PollAnswer("Go"),
		builder.PollAnswer("Rust"),
	)
}

func buildPollRaw() discord.Poll {
	return discord.Poll{
		Question: discord.PollMedia{Text: "Favorite language?"},
		Answers: []discord.PollAnswer{
			{PollMedia: discord.PollMedia{Text: "Go"}},
			{PollMedia: discord.PollMedia{Text: "Rust"}},
		},
		Duration:   24,
		LayoutType: discord.DefaultPollLayoutType,
	}
}

func TestPollBuildersMatchRaw(t *testing.T) {
	assertSameJSON(t, buildPollWithBuilder(), buildPollRaw())
	assertSameJSON(t, builder.PollMedia("q"), discord.PollMedia{Text: "q"})
	assertSameJSON(t, builder.PollAnswer("ans"), discord.PollAnswer{PollMedia: discord.PollMedia{Text: "ans"}})
	assertSameJSON(t, builder.PollAnswerCounts(1, 5), discord.PollAnswerCounts{ID: 1, Count: 5})
	assertSameJSON(t,
		builder.PollResult(builder.PollAnswerCounts(1, 5), builder.PollAnswerCounts(2, 3)),
		discord.PollResult{
			AnswerCounts: []discord.PollAnswerCounts{
				{ID: 1, Count: 5},
				{ID: 2, Count: 3},
			},
		},
	)
}

func TestResponseBuildersMatchRaw(t *testing.T) {
	tests := []struct {
		got  any
		want any
		name string
	}{
		{
			name: "pong",
			got:  builder.PongResponse(),
			want: discord.ResponseMessage{Type: discord.PongResponseType},
		},
		{
			name: "deferred non-ephemeral",
			got:  builder.DeferredResponse(false),
			want: discord.ResponseMessage{Type: discord.DeferredChannelMessageWithSourceResponseType},
		},
		{
			name: "deferred ephemeral",
			got:  builder.DeferredResponse(true),
			want: discord.ResponseMessage{
				Type: discord.DeferredChannelMessageWithSourceResponseType,
				Data: &discord.ResponseMessageData{Flags: discord.EphemeralMessageFlag},
			},
		},
		{
			name: "deferred update",
			got:  builder.DeferredUpdateResponse(),
			want: discord.ResponseMessage{Type: discord.DeferredUpdateMessageResponseType},
		},
		{
			name: "message data",
			got: builder.MessageData("Hello", discord.EphemeralMessageFlag,
				builder.ActionRow(builder.Button("ok", "OK", discord.PrimaryButtonStyle)),
			),
			want: discord.ResponseMessageData{
				Content: "Hello",
				Flags:   discord.EphemeralMessageFlag,
				Components: []discord.MessageComponent{
					discord.ActionRowComponent{
						Type: discord.ActionRowComponentType,
						Components: []discord.ActionRowChildComponent{
							discord.ButtonComponent{
								Type:     discord.ButtonComponentType,
								Style:    discord.PrimaryButtonStyle,
								Label:    "OK",
								CustomID: "ok",
							},
						},
					},
				},
			},
		},
		{
			name: "message response",
			got: builder.MessageResponse("Welcome", discord.SuppressEmbedsMessageFlag,
				builder.ActionRow(builder.LinkButton("https://discord.com", "Discord")),
			),
			want: discord.ResponseMessage{
				Type: discord.ChannelMessageWithSourceResponseType,
				Data: &discord.ResponseMessageData{
					Content: "Welcome",
					Flags:   discord.SuppressEmbedsMessageFlag,
					Components: []discord.MessageComponent{
						discord.ActionRowComponent{
							Type: discord.ActionRowComponentType,
							Components: []discord.ActionRowChildComponent{
								discord.ButtonComponent{
									Type:  discord.ButtonComponentType,
									Style: discord.LinkButtonStyle,
									Label: "Discord",
									URL:   "https://discord.com",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "ephemeral message",
			got:  builder.EphemeralMessage("Only you can see this."),
			want: discord.ResponseMessage{
				Type: discord.ChannelMessageWithSourceResponseType,
				Data: &discord.ResponseMessageData{
					Content: "Only you can see this.",
					Flags:   discord.EphemeralMessageFlag,
				},
			},
		},
		{
			name: "update message response",
			got:  builder.UpdateMessageResponse("Updated", 0),
			want: discord.ResponseMessage{
				Type: discord.UpdateMessageResponseType,
				Data: &discord.ResponseMessageData{Content: "Updated"},
			},
		},
		{
			name: "modal data",
			got: builder.ModalData("feedback", "Feedback",
				builder.Label("Summary", builder.ShortTextInput("summary", "One line")),
				builder.TextDisplay("Optional extra instructions."),
			),
			want: discord.ResponseModalData{
				CustomID: "feedback",
				Title:    "Feedback",
				Components: []discord.ModalComponent{
					discord.LabelComponent{
						Type:  discord.LabelComponentType,
						Label: "Summary",
						Component: discord.TextInputComponent{
							Type:        discord.TextInputComponentType,
							Style:       discord.ShortTextInputStyle,
							CustomID:    "summary",
							Placeholder: "One line",
						},
					},
					discord.TextDisplayComponent{
						Type:    discord.TextDisplayComponentType,
						Content: "Optional extra instructions.",
					},
				},
			},
		},
		{
			name: "modal response",
			got: builder.ModalResponse("profile", "Edit Profile",
				builder.Label("Name", builder.ShortTextInput("name", "Your name")),
				builder.Label("Bio", builder.ParagraphTextInput("bio", "About you")),
				builder.Label("Topics", builder.CheckboxGroup("topics", builder.CheckboxGroupOption("Go", "go"))),
				builder.Label("Role", builder.RadioGroup("role", builder.RadioGroupOption("Dev", "dev"))),
				builder.Label("Notify", builder.Checkbox("notify", true)),
				builder.Label("Select", builder.StringSelect("sel", "Pick", builder.SelectOption("One", "1"))),
				builder.Label("File", builder.FileUpload("file", "image")),
			),
			want: discord.ResponseModal{
				Type: discord.ModalResponseType,
				Data: &discord.ResponseModalData{
					CustomID: "profile",
					Title:    "Edit Profile",
					Components: []discord.ModalComponent{
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "Name",
							Component: discord.TextInputComponent{
								Type: discord.TextInputComponentType, Style: discord.ShortTextInputStyle,
								CustomID: "name", Placeholder: "Your name",
							},
						},
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "Bio",
							Component: discord.TextInputComponent{
								Type: discord.TextInputComponentType, Style: discord.ParagraphTextInputStyle,
								CustomID: "bio", Placeholder: "About you",
							},
						},
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "Topics",
							Component: discord.CheckboxGroupComponent{
								Type: discord.CheckboxGroupComponentType, CustomID: "topics",
								Options: []discord.CheckboxGroupOption{{Label: "Go", Value: "go"}},
							},
						},
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "Role",
							Component: discord.RadioGroupComponent{
								Type: discord.RadioGroupComponentType, CustomID: "role",
								Options: []discord.RadioGroupOption{{Label: "Dev", Value: "dev"}},
							},
						},
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "Notify",
							Component: discord.CheckboxComponent{
								Type: discord.CheckboxComponentType, CustomID: "notify", Default: true,
							},
						},
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "Select",
							Component: discord.StringSelectComponent{
								Type: discord.StringSelectComponentType, CustomID: "sel", Placeholder: "Pick",
								Options: []discord.SelectMenuOption{{Label: "One", Value: "1"}},
							},
						},
						discord.LabelComponent{
							Type:  discord.LabelComponentType,
							Label: "File",
							Component: discord.FileUploadComponent{
								Type: discord.FileUploadComponentType, CustomID: "file",
								FileTypes: []string{"image"},
							},
						},
					},
				},
			},
		},
		{
			name: "autocomplete data",
			got: builder.AutocompleteData(
				discord.CommandOptionChoice{Name: "Alpha", Value: "alpha"},
				discord.CommandOptionChoice{Name: "Beta", Value: "beta"},
			),
			want: discord.ResponseAutoCompleteData{
				Choices: []discord.CommandOptionChoice{
					{Name: "Alpha", Value: "alpha"},
					{Name: "Beta", Value: "beta"},
				},
			},
		},
		{
			name: "autocomplete response",
			got: builder.AutocompleteResponse(
				discord.CommandOptionChoice{Name: "One", Value: "1"},
			),
			want: discord.ResponseAutoComplete{
				Type: discord.AutocompleteResponseType,
				Data: &discord.ResponseAutoCompleteData{
					Choices: []discord.CommandOptionChoice{{Name: "One", Value: "1"}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertSameJSON(t, tc.got, tc.want)
		})
	}
}

func TestResponseEmbedsClearingPreserved(t *testing.T) {
	// Verify empty slice vs nil semantics work cleanly through builder structs.
	editKeep := builder.UpdateMessageResponse("still here", 0)
	rawKeep, err := json.Marshal(editKeep)
	if err != nil {
		t.Fatalf("marshal keep: %v", err)
	}

	if strings.Contains(string(rawKeep), `"embeds"`) {
		t.Errorf("expected nil embeds to be omitted, got %s", rawKeep)
	}

	editClear := builder.UpdateMessageResponse("cleared", 0)
	editClear.Data.Embeds = []discord.Embed{}
	rawClear, err := json.Marshal(editClear)
	if err != nil {
		t.Fatalf("marshal clear: %v", err)
	}

	if !strings.Contains(string(rawClear), `"embeds":[]`) {
		t.Errorf("expected explicit empty embeds array [], got %s", rawClear)
	}
}
