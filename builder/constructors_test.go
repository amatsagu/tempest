package builder

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

// The constructors must produce the same structs, and thus JSON, as handwritten
// literals, so the tests assert identical output and the benchmarks compare cost.

func buildComponentsV2ViaHelpers() discord.Response {
	return MessageResponse("", discord.MessageFlagIsComponentsV2,
		Container(0xffd484,
			TextDisplay("## Welcome"),
			Separator(true),
			SectionWithButton(
				Button("refresh", "Refresh", discord.ButtonSecondary),
				TextDisplay("Server info"),
				TextDisplay("-# updated just now"),
			),
			ActionRow(
				Button("yes", "Yes", discord.ButtonSuccess),
				Button("no", "No", discord.ButtonDanger),
				LinkButton("https://discord.com", "Docs"),
			),
			ActionRowWithStringSelect(
				StringSelect("pick", "Pick one",
					SelectOption("One", "1"),
					discord.SelectOption{Label: "Two", Value: "2", Description: "second", Default: true},
				),
			),
			MediaGallery(
				discord.MediaGalleryItem{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"}, Description: "a"},
			),
		),
	)
}

func buildComponentsV2Direct() discord.Response {
	return discord.Response{
		Type: discord.ResponseChannelMessageWithSource,
		Data: discord.ResponseMessageData{
			Flags: discord.MessageFlagIsComponentsV2,
			Components: discord.Components{
				discord.Container{
					Type:        discord.ComponentContainer,
					AccentColor: 0xffd484,
					Components: []discord.Component{
						discord.TextDisplay{
							Type:    discord.ComponentTextDisplay,
							Content: "## Welcome",
						},
						discord.Separator{
							Type:    discord.ComponentSeparator,
							Divider: true,
						},
						discord.Section{
							Type: discord.ComponentSection,
							Components: []discord.TextDisplay{
								{Type: discord.ComponentTextDisplay, Content: "Server info"},
								{Type: discord.ComponentTextDisplay, Content: "-# updated just now"},
							},
							Button: &discord.Button{
								Type:     discord.ComponentButton,
								Style:    discord.ButtonSecondary,
								Label:    "Refresh",
								CustomID: "refresh",
							},
						},
						discord.ActionRow{
							Type: discord.ComponentActionRow,
							Buttons: []discord.Button{
								{Type: discord.ComponentButton, Style: discord.ButtonSuccess, Label: "Yes", CustomID: "yes"},
								{Type: discord.ComponentButton, Style: discord.ButtonDanger, Label: "No", CustomID: "no"},
								{Type: discord.ComponentButton, Style: discord.ButtonLink, Label: "Docs", URL: "https://discord.com"},
							},
						},
						discord.ActionRow{
							Type: discord.ComponentActionRow,
							StringSelect: &discord.StringSelect{
								Type:        discord.ComponentStringSelect,
								CustomID:    "pick",
								Placeholder: "Pick one",
								Options: []discord.SelectOption{
									{Label: "One", Value: "1"},
									{Label: "Two", Value: "2", Description: "second", Default: true},
								},
							},
						},
						discord.MediaGallery{
							Type: discord.ComponentMediaGallery,
							Items: []discord.MediaGalleryItem{
								{
									Media:       discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"},
									Description: "a",
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildModalViaHelpers() discord.Response {
	return ModalResponse("profile", "Edit Profile",
		Label("Name", ShortTextInput("name", "Your name")),
		LabelWithStringSelect("Favorite", StringSelect("fav", "Choose",
			SelectOption("A", "a"),
		)),
	)
}

func buildModalDirect() discord.Response {
	return discord.Response{
		Type: discord.ResponseModal,
		Data: discord.ResponseModalData{
			CustomID: "profile",
			Title:    "Edit Profile",
			Components: discord.Components{
				discord.Label{
					Type:  discord.ComponentLabel,
					Label: "Name",
					TextInput: &discord.TextInput{
						Type:        discord.ComponentTextInput,
						Style:       discord.TextInputShort,
						CustomID:    "name",
						Placeholder: "Your name",
					},
				},
				discord.Label{
					Type:  discord.ComponentLabel,
					Label: "Favorite",
					StringSelect: &discord.StringSelect{
						Type:        discord.ComponentStringSelect,
						CustomID:    "fav",
						Placeholder: "Choose",
						Options:     []discord.SelectOption{{Label: "A", Value: "a"}},
					},
				},
			},
		},
	}
}

func buildAutocompleteViaHelpers() discord.Response {
	return AutocompleteResponse(
		discord.CommandOptionChoice{Name: "Option A", Value: "a"},
		discord.CommandOptionChoice{Name: "Option B", Value: "b"},
	)
}

func buildAutocompleteDirect() discord.Response {
	return discord.Response{
		Type: discord.ResponseApplicationCommandAutocomplete,
		Data: discord.ResponseAutocompleteData{
			Choices: []discord.CommandOptionChoice{
				{Name: "Option A", Value: "a"},
				{Name: "Option B", Value: "b"},
			},
		},
	}
}

func assertSameJSON(t *testing.T, helpers, direct any) {
	t.Helper()

	helperJSON, err := json.Marshal(helpers)
	if err != nil {
		t.Fatalf("marshal helpers: %v", err)
	}

	directJSON, err := json.Marshal(direct)
	if err != nil {
		t.Fatalf("marshal direct: %v", err)
	}

	if string(helperJSON) != string(directJSON) {
		t.Errorf("helpers and direct differ:\nhelpers: %s\ndirect:  %s", helperJSON, directJSON)
	}
}

func TestConstructorsMatchDirect(t *testing.T) {
	t.Run("components v2", func(t *testing.T) {
		assertSameJSON(t, buildComponentsV2ViaHelpers(), buildComponentsV2Direct())
	})
	t.Run("modal", func(t *testing.T) {
		assertSameJSON(t, buildModalViaHelpers(), buildModalDirect())
	})
	t.Run("autocomplete", func(t *testing.T) {
		assertSameJSON(t, buildAutocompleteViaHelpers(), buildAutocompleteDirect())
	})
	t.Run("deferred", func(t *testing.T) {
		assertSameJSON(t,
			DeferredResponse(true),
			discord.Response{Type: discord.ResponseDeferredChannelMessageWithSource, Data: discord.ResponseMessageData{Flags: discord.MessageFlagEphemeral}},
		)
	})
	t.Run("ephemeral message", func(t *testing.T) {
		assertSameJSON(t,
			EphemeralMessage("hi"),
			discord.Response{Type: discord.ResponseChannelMessageWithSource, Data: discord.ResponseMessageData{Content: "hi", Flags: discord.MessageFlagEphemeral}},
		)
	})
}

func TestOptionBuildersMatchDirect(t *testing.T) {
	assertSameJSON(t,
		StringSelect("s", "p", SelectOption("One", "1"), SelectOption("Two", "2")),
		discord.StringSelect{
			Type:        discord.ComponentStringSelect,
			CustomID:    "s",
			Placeholder: "p",
			Options: []discord.SelectOption{
				{Label: "One", Value: "1"},
				{Label: "Two", Value: "2"},
			},
		},
	)

	assertSameJSON(t,
		MediaGallery(MediaGalleryItem("https://cdn.discordapp.com/a.png")),
		discord.MediaGallery{
			Type:  discord.ComponentMediaGallery,
			Items: []discord.MediaGalleryItem{{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"}}},
		},
	)

	assertSameJSON(t,
		UnfurledMediaItem("https://cdn.discordapp.com/a.png"),
		discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"},
	)

	assertSameJSON(t,
		RadioGroup("r", RadioGroupOption("A", "a")),
		discord.RadioGroup{
			Type:     discord.ComponentRadioGroup,
			CustomID: "r",
			Options:  []discord.RadioGroupOption{{Label: "A", Value: "a"}},
		},
	)

	assertSameJSON(t,
		CheckboxGroup("c", CheckboxGroupOption("A", "a")),
		discord.CheckboxGroup{
			Type:     discord.ComponentCheckboxGroup,
			CustomID: "c",
			Options:  []discord.CheckboxGroupOption{{Label: "A", Value: "a"}},
		},
	)

	assertSameJSON(t,
		[]discord.EmbedField{EmbedField("Name", "Value", true)},
		[]discord.EmbedField{{Name: "Name", Value: "Value", Inline: true}},
	)
}

// A zero accent color must be omitted (ContainerNoAccent is Container(0, ...)).
func TestContainerAccentOptional(t *testing.T) {
	noAccent, err := json.Marshal(Container(0, TextDisplay("x")))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if strings.Contains(string(noAccent), "accent_color") {
		t.Errorf("zero accent should be omitted, got %s", noAccent)
	}

	viaBuilder, _ := json.Marshal(ContainerNoAccent(TextDisplay("x")))
	if string(viaBuilder) != string(noAccent) {
		t.Errorf("ContainerNoAccent != Container(0,...): %s vs %s", viaBuilder, noAccent)
	}

	withAccent, _ := json.Marshal(Container(0xffd484, TextDisplay("x")))
	if !strings.Contains(string(withAccent), `"accent_color":16766084`) {
		t.Errorf("non-zero accent should be present, got %s", withAccent)
	}
}

var sinkResponse discord.Response

func BenchmarkBuildComponentsV2_Helpers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResponse = buildComponentsV2ViaHelpers()
	}
}

func BenchmarkBuildComponentsV2_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResponse = buildComponentsV2Direct()
	}
}

func BenchmarkBuildComponentsV2_Marshal_Helpers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildComponentsV2ViaHelpers()
		sinkJSON, _ = json.Marshal(&resp)
	}
}

func BenchmarkBuildComponentsV2_Marshal_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		resp := buildComponentsV2Direct()
		sinkJSON, _ = json.Marshal(&resp)
	}
}

func BenchmarkBuildModal_Helpers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResponse = buildModalViaHelpers()
	}
}

func BenchmarkBuildModal_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResponse = buildModalDirect()
	}
}

func BenchmarkBuildAutocomplete_Helpers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResponse = buildAutocompleteViaHelpers()
	}
}

func BenchmarkBuildAutocomplete_Direct(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkResponse = buildAutocompleteDirect()
	}
}

var sinkJSON []byte

// Guard against accidental whitespace changes drifting the two encodings.
func TestComponentJSONHasComponentsKey(t *testing.T) {
	raw, _ := json.Marshal(buildComponentsV2ViaHelpers())
	if !strings.Contains(string(raw), `"components":[`) {
		t.Fatalf("expected components array in %s", raw)
	}
}
