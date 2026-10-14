package builder

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

// The NewXXX constructors exist purely to make building payloads shorter and
// harder to get wrong. They must produce exactly the same structs (and thus the
// same JSON) as writing the literals by hand, and must not cost anything
// meaningful at runtime. The tests below assert identical output; the
// benchmarks measure the construction + marshal cost of both styles.

func buildComponentsV2ViaHelpers() discord.Response {
	return NewMessageResponse("", discord.MessageFlagIsComponentsV2,
		NewContainer(0xffd484,
			NewTextDisplay("## Welcome"),
			NewSeparator(true),
			NewSectionWithButton(
				NewButton("refresh", "Refresh", discord.ButtonSecondary),
				NewTextDisplay("Server info"),
				NewTextDisplay("-# updated just now"),
			),
			NewActionRow(
				NewButton("yes", "Yes", discord.ButtonSuccess),
				NewButton("no", "No", discord.ButtonDanger),
				NewLinkButton("https://discord.com", "Docs"),
			),
			NewActionRowWithStringSelect(
				NewStringSelect("pick", "Pick one",
					NewSelectOption("One", "1"),
					discord.SelectOption{Label: "Two", Value: "2", Description: "second", Default: true},
				),
			),
			NewMediaGallery(
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
	return NewModalResponse("profile", "Edit Profile",
		NewLabel("Name", NewShortTextInput("name", "Your name")),
		NewLabelWithStringSelect("Favorite", NewStringSelect("fav", "Choose",
			NewSelectOption("A", "a"),
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
	return NewAutocompleteResponse(
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
			NewDeferredResponse(true),
			discord.Response{Type: discord.ResponseDeferredChannelMessageWithSource, Data: discord.ResponseMessageData{Flags: discord.MessageFlagEphemeral}},
		)
	})
	t.Run("ephemeral message", func(t *testing.T) {
		assertSameJSON(t,
			NewEphemeralMessage("hi"),
			discord.Response{Type: discord.ResponseChannelMessageWithSource, Data: discord.ResponseMessageData{Content: "hi", Flags: discord.MessageFlagEphemeral}},
		)
	})
}

func TestOptionBuildersMatchDirect(t *testing.T) {
	assertSameJSON(t,
		NewStringSelect("s", "p", NewSelectOption("One", "1"), NewSelectOption("Two", "2")),
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
		NewMediaGallery(NewMediaGalleryItem("https://cdn.discordapp.com/a.png")),
		discord.MediaGallery{
			Type:  discord.ComponentMediaGallery,
			Items: []discord.MediaGalleryItem{{Media: discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"}}},
		},
	)

	assertSameJSON(t,
		NewUnfurledMediaItem("https://cdn.discordapp.com/a.png"),
		discord.UnfurledMediaItem{URL: "https://cdn.discordapp.com/a.png"},
	)

	assertSameJSON(t,
		NewRadioGroup("r", NewRadioGroupOption("A", "a")),
		discord.RadioGroup{
			Type:     discord.ComponentRadioGroup,
			CustomID: "r",
			Options:  []discord.RadioGroupOption{{Label: "A", Value: "a"}},
		},
	)

	assertSameJSON(t,
		NewCheckboxGroup("c", NewCheckboxGroupOption("A", "a")),
		discord.CheckboxGroup{
			Type:     discord.ComponentCheckboxGroup,
			CustomID: "c",
			Options:  []discord.CheckboxGroupOption{{Label: "A", Value: "a"}},
		},
	)

	assertSameJSON(t,
		[]discord.EmbedField{NewEmbedField("Name", "Value", true)},
		[]discord.EmbedField{{Name: "Name", Value: "Value", Inline: true}},
	)
}

// A container's accent color is optional in the API; passing zero (or using
// NewContainerNoAccent) must omit it rather than force a raw struct literal.
func TestContainerAccentOptional(t *testing.T) {
	noAccent, err := json.Marshal(NewContainer(0, NewTextDisplay("x")))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(noAccent), "accent_color") {
		t.Errorf("zero accent should be omitted, got %s", noAccent)
	}

	viaBuilder, _ := json.Marshal(NewContainerNoAccent(NewTextDisplay("x")))
	if string(viaBuilder) != string(noAccent) {
		t.Errorf("NewContainerNoAccent != NewContainer(0,...): %s vs %s", viaBuilder, noAccent)
	}

	withAccent, _ := json.Marshal(NewContainer(0xffd484, NewTextDisplay("x")))
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
