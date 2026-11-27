package builder_test

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/builder"
	"github.com/amatsagu/tempest/discord"
)

func assertSameJSON(t *testing.T, got, want any) {
	t.Helper()

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal builder value: %v", err)
	}

	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal raw value: %v", err)
	}

	if !bytes.Equal(gotJSON, wantJSON) {
		t.Errorf("JSON mismatch:\nbuilder: %s\nraw:     %s", gotJSON, wantJSON)
	}
}

func TestButtonConstructorsMatchRaw(t *testing.T) {
	tests := []struct {
		got  any
		want any
		name string
	}{
		{
			name: "primary button",
			got:  builder.Button("btn_ok", "Confirm", discord.PrimaryButtonStyle),
			want: discord.ButtonComponent{
				Type:     discord.ButtonComponentType,
				Style:    discord.PrimaryButtonStyle,
				Label:    "Confirm",
				CustomID: "btn_ok",
			},
		},
		{
			name: "link button",
			got:  builder.LinkButton("https://discord.com/developers/docs", "Docs"),
			want: discord.ButtonComponent{
				Type:  discord.ButtonComponentType,
				Style: discord.LinkButtonStyle,
				Label: "Docs",
				URL:   "https://discord.com/developers/docs",
			},
		},
		{
			name: "premium button",
			got:  builder.PremiumButton(discord.SkuID{Snowflake: 123456789012345678}),
			want: discord.ButtonComponent{
				Type:  discord.ButtonComponentType,
				Style: discord.PremiumButtonStyle,
				SkuID: discord.SkuID{Snowflake: 123456789012345678},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertSameJSON(t, tc.got, tc.want)
		})
	}
}

func TestSelectConstructorsMatchRaw(t *testing.T) {
	tests := []struct {
		got  any
		want any
		name string
	}{
		{
			name: "string select",
			got: builder.StringSelect("menu_id", "Pick one",
				builder.SelectOption("Alpha", "a"),
				discord.SelectMenuOption{Label: "Beta", Value: "b", Description: "Second", Default: true},
			),
			want: discord.StringSelectComponent{
				Type:        discord.StringSelectComponentType,
				CustomID:    "menu_id",
				Placeholder: "Pick one",
				Options: []discord.SelectMenuOption{
					{Label: "Alpha", Value: "a"},
					{Label: "Beta", Value: "b", Description: "Second", Default: true},
				},
			},
		},
		{
			name: "user select",
			got:  builder.UserSelect("pick_user", "Choose users"),
			want: discord.SelectComponent{
				Type:        discord.UserSelectComponentType,
				CustomID:    "pick_user",
				Placeholder: "Choose users",
			},
		},
		{
			name: "role select",
			got:  builder.RoleSelect("pick_role", "Choose roles"),
			want: discord.SelectComponent{
				Type:        discord.RoleSelectComponentType,
				CustomID:    "pick_role",
				Placeholder: "Choose roles",
			},
		},
		{
			name: "mentionable select",
			got:  builder.MentionableSelect("pick_mention", "Choose target"),
			want: discord.SelectComponent{
				Type:        discord.MentionableSelectComponentType,
				CustomID:    "pick_mention",
				Placeholder: "Choose target",
			},
		},
		{
			name: "channel select",
			got: builder.ChannelSelect("pick_chan", "Choose channels",
				discord.GuildTextChannelType,
				discord.GuildAnnouncementChannelType,
			),
			want: discord.SelectComponent{
				Type:         discord.ChannelSelectComponentType,
				CustomID:     "pick_chan",
				Placeholder:  "Choose channels",
				ChannelTypes: []discord.ChannelType{discord.GuildTextChannelType, discord.GuildAnnouncementChannelType},
			},
		},
		{
			name: "channel select without filter",
			got:  builder.ChannelSelect("pick_any_chan", "Any channel"),
			want: discord.SelectComponent{
				Type:        discord.ChannelSelectComponentType,
				CustomID:    "pick_any_chan",
				Placeholder: "Any channel",
			},
		},
		{
			name: "default user",
			got:  builder.SelectDefaultUser(discord.UserID{Snowflake: 11}),
			want: discord.DefaultValueOption{Type: discord.UserDefaultValue, ID: 11},
		},
		{
			name: "default role",
			got:  builder.SelectDefaultRole(discord.RoleID{Snowflake: 22}),
			want: discord.DefaultValueOption{Type: discord.RoleDefaultValue, ID: 22},
		},
		{
			name: "default channel",
			got:  builder.SelectDefaultChannel(discord.ChannelID{Snowflake: 33}),
			want: discord.DefaultValueOption{Type: discord.ChannelDefaultValue, ID: 33},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertSameJSON(t, tc.got, tc.want)
		})
	}
}

func TestLayoutAndContentConstructorsMatchRaw(t *testing.T) {
	tests := []struct {
		got  any
		want any
		name string
	}{
		{
			name: "action row with buttons",
			got: builder.ActionRow(
				builder.Button("yes", "Yes", discord.SuccessButtonStyle),
				builder.Button("no", "No", discord.DangerButtonStyle),
			),
			want: discord.ActionRowComponent{
				Type: discord.ActionRowComponentType,
				Components: []discord.ActionRowChildComponent{
					discord.ButtonComponent{Type: discord.ButtonComponentType, Style: discord.SuccessButtonStyle, Label: "Yes", CustomID: "yes"},
					discord.ButtonComponent{Type: discord.ButtonComponentType, Style: discord.DangerButtonStyle, Label: "No", CustomID: "no"},
				},
			},
		},
		{
			name: "action row with select",
			got: builder.ActionRow(
				builder.UserSelect("u", "Pick user"),
			),
			want: discord.ActionRowComponent{
				Type: discord.ActionRowComponentType,
				Components: []discord.ActionRowChildComponent{
					discord.SelectComponent{Type: discord.UserSelectComponentType, CustomID: "u", Placeholder: "Pick user"},
				},
			},
		},
		{
			name: "text display",
			got:  builder.TextDisplay("## Notice"),
			want: discord.TextDisplayComponent{Type: discord.TextDisplayComponentType, Content: "## Notice"},
		},
		{
			name: "thumbnail",
			got:  builder.Thumbnail("https://cdn.discordapp.com/thumb.png", "preview"),
			want: discord.ThumbnailComponent{
				Type:        discord.ThumbnailComponentType,
				Description: "preview",
				Media:       discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/thumb.png"},
			},
		},
		{
			name: "unfurled media item",
			got:  builder.UnfurledMediaItem("attachment://chart.png"),
			want: discord.UnfurledMediaItem{URL: "attachment://chart.png"},
		},
		{
			name: "media gallery",
			got: builder.MediaGallery(
				builder.MediaGalleryItem("https://cdn.discordapp.com/1.png"),
				discord.MediaGalleryItem{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/2.png"}, Description: "two", Spoiler: true},
			),
			want: discord.MediaGalleryComponent{
				Type: discord.MediaGalleryComponentType,
				Items: []discord.MediaGalleryItem{
					{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/1.png"}},
					{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/2.png"}, Description: "two", Spoiler: true},
				},
			},
		},
		{
			name: "file component",
			got:  builder.File("attachment://report.csv"),
			want: discord.FileComponent{
				Type: discord.FileComponentType,
				File: discord.UnfurledMediaItem{URL: "attachment://report.csv"},
			},
		},
		{
			name: "separator visible",
			got:  builder.Separator(true),
			want: discord.SeparatorComponent{Type: discord.SeparatorComponentType},
		},
		{
			name: "separator hidden divider",
			got:  builder.Separator(false),
			want: discord.SeparatorComponent{Type: discord.SeparatorComponentType, Divider: new(false)},
		},
		{
			name: "section with thumbnail",
			got: builder.Section(
				builder.Thumbnail("https://cdn.discordapp.com/avatar.png", "Avatar"),
				builder.TextDisplay("Line 1"),
				builder.TextDisplay("Line 2"),
			),
			want: discord.SectionComponent{
				Type: discord.SectionComponentType,
				Accessory: discord.ThumbnailComponent{
					Type:        discord.ThumbnailComponentType,
					Description: "Avatar",
					Media:       discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/avatar.png"},
				},
				Components: []discord.TextDisplayComponent{
					{Type: discord.TextDisplayComponentType, Content: "Line 1"},
					{Type: discord.TextDisplayComponentType, Content: "Line 2"},
				},
			},
		},
		{
			name: "section with button",
			got: builder.Section(
				builder.Button("act", "Action", discord.SecondaryButtonStyle),
				builder.TextDisplay("Prompt"),
			),
			want: discord.SectionComponent{
				Type: discord.SectionComponentType,
				Accessory: discord.ButtonComponent{
					Type:     discord.ButtonComponentType,
					Style:    discord.SecondaryButtonStyle,
					Label:    "Action",
					CustomID: "act",
				},
				Components: []discord.TextDisplayComponent{
					{Type: discord.TextDisplayComponentType, Content: "Prompt"},
				},
			},
		},
		{
			name: "container with accent",
			got: builder.Container(0xffd484,
				builder.TextDisplay("Inside"),
				builder.Separator(true),
			),
			want: discord.ContainerComponent{
				Type:        discord.ContainerComponentType,
				AccentColor: 0xffd484,
				Components: []discord.ContainerChildComponent{
					discord.TextDisplayComponent{Type: discord.TextDisplayComponentType, Content: "Inside"},
					discord.SeparatorComponent{Type: discord.SeparatorComponentType},
				},
			},
		},
		{
			name: "container without accent",
			got:  builder.ContainerNoAccent(builder.TextDisplay("No accent")),
			want: discord.ContainerComponent{
				Type: discord.ContainerComponentType,
				Components: []discord.ContainerChildComponent{
					discord.TextDisplayComponent{Type: discord.TextDisplayComponentType, Content: "No accent"},
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

func TestModalComponentConstructorsMatchRaw(t *testing.T) {
	optionalSelect := builder.StringSelect("opt_sel", "Optional")
	optionalSelect.MinValues = new(uint8(0))
	optionalSelect.Required = new(false)

	tests := []struct {
		got  any
		want any
		name string
	}{
		{
			name: "short text input",
			got:  builder.ShortTextInput("title", "Short title"),
			want: discord.TextInputComponent{
				Type:        discord.TextInputComponentType,
				Style:       discord.ShortTextInputStyle,
				CustomID:    "title",
				Placeholder: "Short title",
			},
		},
		{
			name: "paragraph text input",
			got:  builder.ParagraphTextInput("body", "Details"),
			want: discord.TextInputComponent{
				Type:        discord.TextInputComponentType,
				Style:       discord.ParagraphTextInputStyle,
				CustomID:    "body",
				Placeholder: "Details",
			},
		},
		{
			name: "text input explicit style",
			got:  builder.TextInput("custom", "Enter", discord.ShortTextInputStyle),
			want: discord.TextInputComponent{
				Type:        discord.TextInputComponentType,
				Style:       discord.ShortTextInputStyle,
				CustomID:    "custom",
				Placeholder: "Enter",
			},
		},
		{
			name: "label wrapping text input",
			got:  builder.Label("Title", builder.ShortTextInput("title", "Your title")),
			want: discord.LabelComponent{
				Type:  discord.LabelComponentType,
				Label: "Title",
				Component: discord.TextInputComponent{
					Type:        discord.TextInputComponentType,
					Style:       discord.ShortTextInputStyle,
					CustomID:    "title",
					Placeholder: "Your title",
				},
			},
		},
		{
			name: "label with description",
			got: builder.LabelWithDescription("Avatar", "Upload a PNG or WebP",
				builder.FileUpload("avatar_up", ".png", ".webp"),
			),
			want: discord.LabelComponent{
				Type:        discord.LabelComponentType,
				Label:       "Avatar",
				Description: "Upload a PNG or WebP",
				Component: discord.FileUploadComponent{
					Type:      discord.FileUploadComponentType,
					CustomID:  "avatar_up",
					FileTypes: []string{".png", ".webp"},
				},
			},
		},
		{
			name: "radio group",
			got: builder.RadioGroup("role",
				builder.RadioGroupOption("Tank", "tank"),
				discord.RadioGroupOption{Label: "Healer", Value: "heal", Description: "Support", Default: true},
			),
			want: discord.RadioGroupComponent{
				Type:     discord.RadioGroupComponentType,
				CustomID: "role",
				Options: []discord.RadioGroupOption{
					{Label: "Tank", Value: "tank"},
					{Label: "Healer", Value: "heal", Description: "Support", Default: true},
				},
			},
		},
		{
			name: "checkbox group",
			got: builder.CheckboxGroup("days",
				builder.CheckboxGroupOption("Monday", "mon"),
				builder.CheckboxGroupOption("Friday", "fri"),
			),
			want: discord.CheckboxGroupComponent{
				Type:     discord.CheckboxGroupComponentType,
				CustomID: "days",
				Options: []discord.CheckboxGroupOption{
					{Label: "Monday", Value: "mon"},
					{Label: "Friday", Value: "fri"},
				},
			},
		},
		{
			name: "checkbox",
			got:  builder.Checkbox("agree_tos", true),
			want: discord.CheckboxComponent{
				Type:     discord.CheckboxComponentType,
				CustomID: "agree_tos",
				Default:  true,
			},
		},
		{
			name: "string select with zero min values and explicit optional required",
			got:  builder.Label("Pick", optionalSelect),
			want: discord.LabelComponent{
				Type:  discord.LabelComponentType,
				Label: "Pick",
				Component: discord.StringSelectComponent{
					Type:        discord.StringSelectComponentType,
					CustomID:    "opt_sel",
					Placeholder: "Optional",
					MinValues:   new(uint8(0)),
					Required:    new(false),
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

func TestComponentBuilderSemantics(t *testing.T) {
	noAccent, err := json.Marshal(builder.ContainerNoAccent(builder.TextDisplay("x")))
	if err != nil {
		t.Fatalf("marshal ContainerNoAccent: %v", err)
	}

	if strings.Contains(string(noAccent), "accent_color") {
		t.Errorf("ContainerNoAccent must omit accent_color, got %s", noAccent)
	}

	visibleDivider, err := json.Marshal(builder.Separator(true))
	if err != nil {
		t.Fatalf("marshal Separator(true): %v", err)
	}

	if strings.Contains(string(visibleDivider), `"divider"`) {
		t.Errorf("Separator(true) must omit divider (Discord default true), got %s", visibleDivider)
	}

	hiddenDivider, err := json.Marshal(builder.Separator(false))
	if err != nil {
		t.Fatalf("marshal Separator(false): %v", err)
	}

	if !strings.Contains(string(hiddenDivider), `"divider":false`) {
		t.Errorf("Separator(false) must serialize divider:false, got %s", hiddenDivider)
	}

	msgSelect, err := json.Marshal(builder.StringSelect("pick", "Pick one", builder.SelectOption("A", "a")))
	if err != nil {
		t.Fatalf("marshal StringSelect: %v", err)
	}

	if strings.Contains(string(msgSelect), `"required"`) {
		t.Errorf("message StringSelect must omit modal-only required key, got %s", msgSelect)
	}

	// Verify round-trip unmarshaling of polymorphic Label + Container produced by builders.
	container := builder.Container(0x5865f2,
		builder.Section(
			builder.Thumbnail("https://example.com/a.png", "desc"),
			builder.TextDisplay("hello"),
		),
		builder.Separator(true),
		builder.ActionRow(builder.Button("b", "Go", discord.PrimaryButtonStyle)),
	)
	raw, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("marshal container: %v", err)
	}

	var decoded discord.ContainerComponent
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal container: %v", err)
	}

	if decoded.AccentColor != 0x5865f2 || len(decoded.Components) != 3 {
		t.Errorf("unexpected decoded container: %+v", decoded)
	}
}
