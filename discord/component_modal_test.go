package discord

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestComponentModal(t *testing.T) {
	txt := NewShortTextInput("name_input", "Your name")
	label := NewLabel("Name", txt)

	if label.Type != ComponentLabel {
		t.Errorf("got %d, want %d", label.Type, ComponentLabel)
	}

	data, err := json.Marshal(label)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"label":"Name"`) || !strings.Contains(str, `"type":18`) || !strings.Contains(str, `"custom_id":"name_input"`) {
		t.Errorf("got %s, missing expected fields", str)
	}

	var parsedLabel Label
	if err := json.Unmarshal(data, &parsedLabel); err != nil {
		t.Fatalf("Unmarshal Label failed: %v", err)
	}
	if parsedLabel.TextInput == nil || parsedLabel.TextInput.CustomID != "name_input" {
		t.Errorf("parsedLabel = %+v, want text input", parsedLabel)
	}

	chk := NewCheckbox("agree_tos", true)
	if chk.Type != ComponentCheckbox {
		t.Errorf("got %d, want %d", chk.Type, ComponentCheckbox)
	}

	chkGroup := NewCheckboxGroup("days",
		CheckboxGroupOption{Label: "Monday", Value: "mon"},
		CheckboxGroupOption{Label: "Friday", Value: "fri"},
	)
	if chkGroup.Type != ComponentCheckboxGroup {
		t.Errorf("got %d, want %d", chkGroup.Type, ComponentCheckboxGroup)
	}

	radio := NewRadioGroup("class",
		RadioGroupOption{Label: "Warrior", Value: "warrior"},
		RadioGroupOption{Label: "Mage", Value: "mage"},
	)
	if radio.Type != ComponentRadioGroup {
		t.Errorf("got %d, want %d", radio.Type, ComponentRadioGroup)
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
