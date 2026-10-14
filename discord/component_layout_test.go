package discord

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestComponentLayout(t *testing.T) {
	btn := NewButton("b1", "Button 1", ButtonPrimary)
	row := NewActionRow(btn)

	if row.Type != ComponentActionRow {
		t.Errorf("got %d, want %d", row.Type, ComponentActionRow)
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `"type":1`) || !strings.Contains(str, `"custom_id":"b1"`) {
		t.Errorf("got %s, missing expected fields", str)
	}

	var parsedRow ActionRow
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

	var parsedSec Section
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
	btn := NewButton("b1", "Button 1", ButtonPrimary)
	row := NewActionRow(btn)
	container := NewContainer(0xff0000, row)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(&container)
	}
}
