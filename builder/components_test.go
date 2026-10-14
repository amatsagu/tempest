package builder

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func TestButton(t *testing.T) {
	btn := NewButton("btn_click", "Click Me", discord.ButtonPrimary)
	if btn.Type != discord.ComponentButton {
		t.Errorf("got %d, want %d", btn.Type, discord.ComponentButton)
	}

	data, err := json.Marshal(btn)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed discord.Button
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.CustomID != btn.CustomID || parsed.Label != btn.Label || parsed.Style != btn.Style {
		t.Errorf("parsed = %+v, want %+v", parsed, btn)
	}
}

func BenchmarkButton_JSON(b *testing.B) {
	btn := NewButton("btn_click", "Click Me", discord.ButtonPrimary)
	data, _ := json.Marshal(&btn)

	var parsed discord.Button
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&btn)
	}
}

func TestComponentContent(t *testing.T) {
	txt := NewTextDisplay("**Bold Notice**")
	if txt.Type != discord.ComponentTextDisplay {
		t.Errorf("got %d, want %d", txt.Type, discord.ComponentTextDisplay)
	}

	data, err := json.Marshal(txt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed discord.TextDisplay
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Content != txt.Content {
		t.Errorf("parsed.Content = %s, want %s", parsed.Content, txt.Content)
	}

	thumb := NewThumbnail("https://example.com/thumb.png", "thumbnail")
	if thumb.Type != discord.ComponentThumbnail {
		t.Errorf("thumb.Type = %d, want %d", thumb.Type, discord.ComponentThumbnail)
	}
}

func BenchmarkComponentContent_JSON(b *testing.B) {
	txt := NewTextDisplay("**Bold Notice**")
	data, _ := json.Marshal(&txt)

	var parsed discord.TextDisplay
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&txt)
	}
}

func TestComponentLayout(t *testing.T) {
	btn := NewButton("b1", "Button 1", discord.ButtonPrimary)
	row := NewActionRow(btn)

	if row.Type != discord.ComponentActionRow {
		t.Errorf("got %d, want %d", row.Type, discord.ComponentActionRow)
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"type":1`) || !strings.Contains(str, `"custom_id":"b1"`) {
		t.Errorf("got %s, missing expected fields", str)
	}

	var parsedRow discord.ActionRow
	if err := json.Unmarshal(data, &parsedRow); err != nil {
		t.Fatalf("Unmarshal ActionRow failed: %v", err)
	}
	if len(parsedRow.Buttons) != 1 || parsedRow.Buttons[0].CustomID != "b1" {
		t.Errorf("parsedRow = %+v, want 1 button", parsedRow)
	}

	sec := NewSectionWithButton(btn, NewTextDisplay("Section text"))
	secData, err := json.Marshal(sec)
	if err != nil {
		t.Fatalf("Marshal Section failed: %v", err)
	}

	var parsedSec discord.Section
	if err := json.Unmarshal(secData, &parsedSec); err != nil {
		t.Fatalf("Unmarshal Section failed: %v", err)
	}
	if parsedSec.Button == nil || parsedSec.Button.CustomID != "b1" || len(parsedSec.Components) != 1 {
		t.Errorf("parsedSec = %+v, want button and 1 text display", parsedSec)
	}

	container := NewContainer(0xff0000, row)
	cData, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("Container marshal failed: %v", err)
	}

	if len(cData) == 0 {
		t.Errorf("expected non-empty container json")
	}
}

func BenchmarkComponentLayout_JSON(b *testing.B) {
	btn := NewButton("b1", "Button 1", discord.ButtonPrimary)
	row := NewActionRow(btn)
	container := NewContainer(0xff0000, row)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&container)
	}
}

func TestComponentModal(t *testing.T) {
	txt := NewShortTextInput("name_input", "Your name")
	label := NewLabel("Name", txt)

	if label.Type != discord.ComponentLabel {
		t.Errorf("got %d, want %d", label.Type, discord.ComponentLabel)
	}

	data, err := json.Marshal(label)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"label":"Name"`) || !strings.Contains(str, `"type":18`) || !strings.Contains(str, `"custom_id":"name_input"`) {
		t.Errorf("got %s, missing expected fields", str)
	}

	var parsedLabel discord.Label
	if err := json.Unmarshal(data, &parsedLabel); err != nil {
		t.Fatalf("Unmarshal Label failed: %v", err)
	}
	if parsedLabel.TextInput == nil || parsedLabel.TextInput.CustomID != "name_input" {
		t.Errorf("parsedLabel = %+v, want text input", parsedLabel)
	}

	chk := NewCheckbox("agree_tos", true)
	if chk.Type != discord.ComponentCheckbox {
		t.Errorf("got %d, want %d", chk.Type, discord.ComponentCheckbox)
	}

	chkGroup := NewCheckboxGroup("days",
		discord.CheckboxGroupOption{Label: "Monday", Value: "mon"},
		discord.CheckboxGroupOption{Label: "Friday", Value: "fri"},
	)
	if chkGroup.Type != discord.ComponentCheckboxGroup {
		t.Errorf("got %d, want %d", chkGroup.Type, discord.ComponentCheckboxGroup)
	}

	radio := NewRadioGroup("class",
		discord.RadioGroupOption{Label: "Warrior", Value: "warrior"},
		discord.RadioGroupOption{Label: "Mage", Value: "mage"},
	)
	if radio.Type != discord.ComponentRadioGroup {
		t.Errorf("got %d, want %d", radio.Type, discord.ComponentRadioGroup)
	}
}

func BenchmarkComponentModal_JSON(b *testing.B) {
	txt := NewShortTextInput("name_input", "Your name")
	label := NewLabel("Name", txt)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&label)
	}
}

func TestSelectComponents(t *testing.T) {
	sel := NewStringSelect("my_select", "Choose an option",
		discord.SelectOption{Label: "Opt 1", Value: "val_1"},
		discord.SelectOption{Label: "Opt 2", Value: "val_2", Default: true},
	)

	if sel.Type != discord.ComponentStringSelect {
		t.Errorf("got %d, want %d", sel.Type, discord.ComponentStringSelect)
	}

	data, err := json.Marshal(sel)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed discord.StringSelect
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.CustomID != sel.CustomID || len(parsed.Options) != 2 {
		t.Errorf("parsed = %+v, want %+v", parsed, sel)
	}

	userSel := NewUserSelect("user_pick", "Select user")
	if userSel.Type != discord.ComponentUserSelect {
		t.Errorf("userSel.Type = %d, want %d", userSel.Type, discord.ComponentUserSelect)
	}
}

func BenchmarkSelect_JSON(b *testing.B) {
	sel := NewStringSelect("my_select", "Choose an option",
		discord.SelectOption{Label: "Opt 1", Value: "val_1"},
		discord.SelectOption{Label: "Opt 2", Value: "val_2", Default: true},
	)
	data, _ := json.Marshal(&sel)

	var parsed discord.StringSelect
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&sel)
	}
}

func TestTextInput(t *testing.T) {
	input := NewShortTextInput("feedback", "Type feedback here")
	if input.Type != discord.ComponentTextInput {
		t.Errorf("got %d, want %d", input.Type, discord.ComponentTextInput)
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed discord.TextInput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.CustomID != input.CustomID || parsed.Style != input.Style {
		t.Errorf("parsed = %+v, want %+v", parsed, input)
	}
}

func BenchmarkTextInput_JSON(b *testing.B) {
	input := NewShortTextInput("feedback", "Type feedback here")
	data, _ := json.Marshal(&input)

	var parsed discord.TextInput
	b.ReportAllocs()
	for b.Loop() {
		_ = json.Unmarshal(data, &parsed)
		_, _ = json.Marshal(&input)
	}
}
