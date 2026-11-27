package discord

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"strings"
	"testing"
)

func mustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal(%T) failed: %v", v, err)
	}

	return data
}

func TestUnmarshalComponentAllTypes(t *testing.T) {
	emoji := &Emoji{
		ID:       EmojiID{Snowflake: 555},
		Name:     "wave",
		Animated: true,
	}

	tests := []struct {
		component AnyComponent
		assert    func(t *testing.T, got AnyComponent)
		name      string
	}{
		{
			name: "action_row",
			component: ActionRowComponent{
				Type: ActionRowComponentType,
				ID:   1,
				Components: []ActionRowChildComponent{
					ButtonComponent{
						Type:     ButtonComponentType,
						Style:    PrimaryButtonStyle,
						Label:    "Click",
						CustomID: "btn_click",
					},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				row, ok := got.(ActionRowComponent)
				if !ok || row.ID != 1 || len(row.Components) != 1 {
					t.Fatalf("unexpected action row: %#v", got)
				}
			},
		},
		{
			name: "button",
			component: ButtonComponent{
				Type:     ButtonComponentType,
				ID:       2,
				Style:    SecondaryButtonStyle,
				Label:    "Press",
				CustomID: "btn_press",
				Emoji:    emoji,
				Disabled: true,
			},
			assert: func(t *testing.T, got AnyComponent) {
				btn, ok := got.(ButtonComponent)
				if !ok || btn.CustomID != "btn_press" || !btn.Disabled || btn.Emoji == nil {
					t.Fatalf("unexpected button: %#v", got)
				}
			},
		},
		{
			name: "string_select",
			component: StringSelectComponent{
				Type:        StringSelectComponentType,
				ID:          3,
				CustomID:    "str_sel",
				Placeholder: "Choose",
				MinValues:   new(uint8(1)),
				MaxValues:   2,
				Options: []SelectMenuOption{
					{Label: "One", Value: "1", Description: "First", Default: true, Emoji: emoji},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				sel, ok := got.(StringSelectComponent)
				if !ok || sel.CustomID != "str_sel" || len(sel.Options) != 1 || !sel.Options[0].Default {
					t.Fatalf("unexpected string select: %#v", got)
				}
			},
		},
		{
			name: "text_input",
			component: TextInputComponent{
				Type:        TextInputComponentType,
				ID:          4,
				CustomID:    "txt_in",
				Style:       ParagraphTextInputStyle,
				Label:       "Bio",
				Placeholder: "About you",
				Value:       "Hello",
				MinLength:   2,
				MaxLength:   200,
				Required:    new(true),
			},
			assert: func(t *testing.T, got AnyComponent) {
				ti, ok := got.(TextInputComponent)
				if !ok || ti.CustomID != "txt_in" || ti.Value != "Hello" || ti.Required == nil || !*ti.Required {
					t.Fatalf("unexpected text input: %#v", got)
				}
			},
		},
		{
			name: "user_select",
			component: SelectComponent{
				Type:      UserSelectComponentType,
				ID:        5,
				CustomID:  "usr_sel",
				MinValues: new(uint8(1)),
				MaxValues: 3,
				DefaultValues: []DefaultValueOption{
					{Type: UserDefaultValue, ID: 1001},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				sel, ok := got.(SelectComponent)
				if !ok || sel.Type != UserSelectComponentType || len(sel.DefaultValues) != 1 {
					t.Fatalf("unexpected user select: %#v", got)
				}
			},
		},
		{
			name: "role_select",
			component: SelectComponent{
				Type:     RoleSelectComponentType,
				ID:       6,
				CustomID: "role_sel",
			},
			assert: func(t *testing.T, got AnyComponent) {
				sel, ok := got.(SelectComponent)
				if !ok || sel.Type != RoleSelectComponentType || sel.CustomID != "role_sel" {
					t.Fatalf("unexpected role select: %#v", got)
				}
			},
		},
		{
			name: "mentionable_select",
			component: SelectComponent{
				Type:     MentionableSelectComponentType,
				ID:       7,
				CustomID: "ment_sel",
			},
			assert: func(t *testing.T, got AnyComponent) {
				sel, ok := got.(SelectComponent)
				if !ok || sel.Type != MentionableSelectComponentType || sel.CustomID != "ment_sel" {
					t.Fatalf("unexpected mentionable select: %#v", got)
				}
			},
		},
		{
			name: "channel_select",
			component: SelectComponent{
				Type:         ChannelSelectComponentType,
				ID:           8,
				CustomID:     "chan_sel",
				ChannelTypes: []ChannelType{GuildTextChannelType, GuildAnnouncementChannelType},
			},
			assert: func(t *testing.T, got AnyComponent) {
				sel, ok := got.(SelectComponent)
				if !ok || sel.Type != ChannelSelectComponentType || len(sel.ChannelTypes) != 2 {
					t.Fatalf("unexpected channel select: %#v", got)
				}
			},
		},
		{
			name: "section",
			component: SectionComponent{
				Type: SectionComponentType,
				ID:   9,
				Components: []TextDisplayComponent{
					{Type: TextDisplayComponentType, Content: "Section line"},
				},
				Accessory: ThumbnailComponent{
					Type:        ThumbnailComponentType,
					Description: "thumb",
					Media:       UnfurledMediaItem{URL: "https://example.com/a.png"},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				sec, ok := got.(SectionComponent)
				if !ok || len(sec.Components) != 1 || sec.Accessory == nil {
					t.Fatalf("unexpected section: %#v", got)
				}
			},
		},
		{
			name: "text_display",
			component: TextDisplayComponent{
				Type:    TextDisplayComponentType,
				ID:      10,
				Content: "Markdown **text**",
			},
			assert: func(t *testing.T, got AnyComponent) {
				td, ok := got.(TextDisplayComponent)
				if !ok || td.Content != "Markdown **text**" {
					t.Fatalf("unexpected text display: %#v", got)
				}
			},
		},
		{
			name: "thumbnail",
			component: ThumbnailComponent{
				Type:        ThumbnailComponentType,
				ID:          11,
				Description: "Alt",
				Spoiler:     true,
				Media:       UnfurledMediaItem{URL: "https://example.com/t.png", Width: 100, Height: 100},
			},
			assert: func(t *testing.T, got AnyComponent) {
				th, ok := got.(ThumbnailComponent)
				if !ok || !th.Spoiler || th.Media.Width != 100 {
					t.Fatalf("unexpected thumbnail: %#v", got)
				}
			},
		},
		{
			name: "media_gallery",
			component: MediaGalleryComponent{
				Type: MediaGalleryComponentType,
				ID:   12,
				Items: []MediaGalleryItem{
					{Description: "img", Spoiler: true, Media: UnfurledMediaItem{URL: "https://example.com/1.png"}},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				mg, ok := got.(MediaGalleryComponent)
				if !ok || len(mg.Items) != 1 || mg.Items[0].Description != "img" {
					t.Fatalf("unexpected media gallery: %#v", got)
				}
			},
		},
		{
			name: "file",
			component: FileComponent{
				Type:    FileComponentType,
				ID:      13,
				Name:    "report.pdf",
				Size:    4096,
				Spoiler: true,
				File:    UnfurledMediaItem{URL: "attachment://report.pdf", AttachmentID: AttachmentID{Snowflake: 777}},
			},
			assert: func(t *testing.T, got AnyComponent) {
				fc, ok := got.(FileComponent)
				if !ok || fc.Name != "report.pdf" || fc.Size != 4096 || fc.File.AttachmentID.Snowflake != 777 {
					t.Fatalf("unexpected file: %#v", got)
				}
			},
		},
		{
			name: "separator",
			component: SeparatorComponent{
				Type:    SeparatorComponentType,
				ID:      14,
				Divider: new(true),
				Spacing: LargeSeparatorSpacing,
			},
			assert: func(t *testing.T, got AnyComponent) {
				sep, ok := got.(SeparatorComponent)
				if !ok || sep.Divider == nil || !*sep.Divider || sep.Spacing != LargeSeparatorSpacing {
					t.Fatalf("unexpected separator: %#v", got)
				}
			},
		},
		{
			name: "container",
			component: ContainerComponent{
				Type:        ContainerComponentType,
				ID:          17,
				AccentColor: 0x5865F2,
				Spoiler:     true,
				Components: []ContainerChildComponent{
					TextDisplayComponent{Type: TextDisplayComponentType, Content: "Inside"},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				ct, ok := got.(ContainerComponent)
				if !ok || ct.AccentColor != 0x5865F2 || !ct.Spoiler || len(ct.Components) != 1 {
					t.Fatalf("unexpected container: %#v", got)
				}
			},
		},
		{
			name: "label",
			component: LabelComponent{
				Type:        LabelComponentType,
				ID:          18,
				Label:       "Field",
				Description: "Desc",
				Component: CheckboxComponent{
					Type:     CheckboxComponentType,
					CustomID: "cb",
					Value:    true,
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				lbl, ok := got.(LabelComponent)
				if !ok || lbl.Label != "Field" || lbl.Description != "Desc" || lbl.Component == nil {
					t.Fatalf("unexpected label: %#v", got)
				}
			},
		},
		{
			name: "file_upload",
			component: FileUploadComponent{
				Type:      FileUploadComponentType,
				ID:        19,
				CustomID:  "upload",
				FileTypes: []string{"image"},
				Values:    []AttachmentID{{Snowflake: 888}},
				MinValues: new(uint8(1)),
				MaxValues: 5,
				Required:  new(true),
			},
			assert: func(t *testing.T, got AnyComponent) {
				fu, ok := got.(FileUploadComponent)
				if !ok || fu.CustomID != "upload" || len(fu.Values) != 1 {
					t.Fatalf("unexpected file upload: %#v", got)
				}
			},
		},
		{
			name: "radio_group",
			component: RadioGroupComponent{
				Type:     RadioGroupComponentType,
				ID:       21,
				CustomID: "radio",
				Required: new(true),
				Options: []RadioGroupOption{
					{Label: "A", Value: "a", Description: "opt a", Default: true},
					{Label: "B", Value: "b"},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				rg, ok := got.(RadioGroupComponent)
				if !ok || rg.CustomID != "radio" || len(rg.Options) != 2 {
					t.Fatalf("unexpected radio group: %#v", got)
				}
			},
		},
		{
			name: "checkbox_group",
			component: CheckboxGroupComponent{
				Type:      CheckboxGroupComponentType,
				ID:        22,
				CustomID:  "cbg",
				MinValues: new(uint8(1)),
				MaxValues: 2,
				Required:  new(true),
				Values:    []string{"x"},
				Options: []CheckboxGroupOption{
					{Label: "X", Value: "x", Default: true},
				},
			},
			assert: func(t *testing.T, got AnyComponent) {
				cbg, ok := got.(CheckboxGroupComponent)
				if !ok || cbg.CustomID != "cbg" || len(cbg.Values) != 1 {
					t.Fatalf("unexpected checkbox group: %#v", got)
				}
			},
		},
		{
			name: "checkbox",
			component: CheckboxComponent{
				Type:     CheckboxComponentType,
				ID:       23,
				CustomID: "chk",
				Default:  true,
				Value:    true,
			},
			assert: func(t *testing.T, got AnyComponent) {
				chk, ok := got.(CheckboxComponent)
				if !ok || chk.CustomID != "chk" || !chk.Default || !chk.Value {
					t.Fatalf("unexpected checkbox: %#v", got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := mustMarshalJSON(t, tc.component)
			got, err := UnmarshalComponent(raw)
			if err != nil {
				t.Fatalf("UnmarshalComponent failed: %v\njson: %s", err, raw)
			}

			tc.assert(t, got)
		})
	}
}

func TestUnmarshalComponentErrors(t *testing.T) {
	invalidPayloads := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"invalid_json", "{"},
		{"non_object", `[]`},
		{"missing_type", `{"id":1}`},
		{"string_type", `{"type":"2"}`},
		{"unknown_type", `{"type":99}`},
		{"bad_action_row", `{"type":1,"components":"oops"}`},
		{"bad_action_row_child", `{"type":1,"components":[{"type":99}]}`},
		{"bad_button", `{"type":2,"style":"bad"}`},
		{"bad_string_select", `{"type":3,"options":"bad"}`},
		{"bad_text_input", `{"type":4,"style":"bad"}`},
		{"bad_select", `{"type":5,"values":"bad"}`},
		{"bad_section", `{"type":9,"components":"bad"}`},
		{"bad_section_accessory", `{"type":9,"accessory":{"type":99}}`},
		{"bad_text_display", `{"type":10,"content":123}`},
		{"bad_thumbnail", `{"type":11,"spoiler":"bad"}`},
		{"bad_media_gallery", `{"type":12,"items":"bad"}`},
		{"bad_file", `{"type":13,"size":"bad"}`},
		{"bad_separator", `{"type":14,"divider":"bad"}`},
		{"bad_container", `{"type":17,"components":"bad"}`},
		{"bad_container_child", `{"type":17,"components":[{"type":99}]}`},
		{"bad_label", `{"type":18,"label":123}`},
		{"bad_label_child", `{"type":18,"component":{"type":99}}`},
		{"bad_file_upload", `{"type":19,"required":"bad"}`},
		{"bad_radio_group", `{"type":21,"required":"bad"}`},
		{"bad_checkbox_group", `{"type":22,"required":"bad"}`},
		{"bad_checkbox", `{"type":23,"value":"bad"}`},
	}

	for _, tc := range invalidPayloads {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := UnmarshalComponent([]byte(tc.raw)); err == nil {
				t.Fatalf("expected error for payload %s, got nil", tc.raw)
			}
		})
	}
}

func TestNestedComponentUnmarshalAndSemantics(t *testing.T) {
	t.Run("container_with_all_children", func(t *testing.T) {
		container := ContainerComponent{
			Type:        ContainerComponentType,
			ID:          100,
			AccentColor: 0xFF00AA,
			Components: []ContainerChildComponent{
				TextDisplayComponent{Type: TextDisplayComponentType, Content: "Header"},
				SeparatorComponent{Type: SeparatorComponentType, Divider: new(false), Spacing: SmallSeparatorSpacing},
				SectionComponent{
					Type: SectionComponentType,
					Components: []TextDisplayComponent{
						{Type: TextDisplayComponentType, Content: "With button"},
					},
					Accessory: ButtonComponent{Type: ButtonComponentType, Style: LinkButtonStyle, Label: "Open", URL: "https://example.com"},
				},
				SectionComponent{
					Type: SectionComponentType,
					Components: []TextDisplayComponent{
						{Type: TextDisplayComponentType, Content: "No accessory"},
					},
				},
				ActionRowComponent{
					Type: ActionRowComponentType,
					Components: []ActionRowChildComponent{
						StringSelectComponent{
							Type:     StringSelectComponentType,
							CustomID: "sel",
							Options:  []SelectMenuOption{{Label: "A", Value: "a"}},
						},
					},
				},
				ActionRowComponent{
					Type: ActionRowComponentType,
					Components: []ActionRowChildComponent{
						SelectComponent{Type: UserSelectComponentType, CustomID: "u"},
					},
				},
				MediaGalleryComponent{
					Type: MediaGalleryComponentType,
					Items: []MediaGalleryItem{
						{Media: UnfurledMediaItem{URL: "https://example.com/1.png"}},
					},
				},
				FileComponent{
					Type: FileComponentType,
					File: UnfurledMediaItem{URL: "attachment://a.txt"},
				},
			},
		}

		raw := mustMarshalJSON(t, container)
		if !strings.Contains(string(raw), `"divider":false`) {
			t.Fatalf("expected explicit divider:false in serialized container, got %s", raw)
		}

		var decoded ContainerComponent
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("ContainerComponent unmarshal failed: %v", err)
		}

		if len(decoded.Components) != 8 {
			t.Fatalf("expected 8 children, got %d", len(decoded.Components))
		}
	})

	t.Run("section_without_accessory_or_with_button", func(t *testing.T) {
		var sec SectionComponent
		if err := json.Unmarshal([]byte(`{"type":9,"components":[{"type":10,"content":"hi"}]}`), &sec); err != nil {
			t.Fatalf("section unmarshal failed: %v", err)
		}

		if sec.Accessory != nil || len(sec.Components) != 1 {
			t.Fatalf("unexpected section: %#v", sec)
		}

		if err := json.Unmarshal([]byte(`{"type":9,"accessory":{"type":10,"content":"not accessory"}}`), &sec); err != nil {
			t.Fatalf("section unmarshal failed when non-accessory: %v", err)
		}

		if sec.Accessory != nil {
			t.Fatalf("expected non-accessory component to be ignored, got %#v", sec.Accessory)
		}

		if err := json.Unmarshal([]byte(`{"type":9,"accessory":{"type":99}}`), &sec); err == nil {
			t.Fatal("expected error for unknown section accessory component")
		}
	})

	t.Run("action_row_and_container_skip_non_member_component", func(t *testing.T) {
		var row ActionRowComponent
		if err := json.Unmarshal([]byte(`{"type":1,"components":[{"type":10,"content":"text_display"}]}`), &row); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(row.Components) != 0 {
			t.Fatalf("expected non-ActionRowChildComponent to be skipped, got %#v", row.Components)
		}

		var container ContainerComponent
		if err := json.Unmarshal([]byte(`{"type":17,"components":[{"type":2,"style":1,"label":"btn","custom_id":"b"}]}`), &container); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(container.Components) != 0 {
			t.Fatalf("expected non-ContainerChildComponent to be skipped, got %#v", container.Components)
		}
	})

	t.Run("label_without_component_or_with_non_label_child", func(t *testing.T) {
		var lbl LabelComponent
		if err := json.Unmarshal([]byte(`{"type":18,"label":"Empty"}`), &lbl); err != nil {
			t.Fatalf("unexpected error when label has no component: %v", err)
		}

		if lbl.Component != nil {
			t.Fatalf("expected nil component, got %#v", lbl.Component)
		}

		if err := json.Unmarshal([]byte(`{"type":18,"label":"BadChild","component":{"type":10,"content":"hi"}}`), &lbl); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if lbl.Component != nil {
			t.Fatalf("expected non-LabelChildComponent to be ignored, got %#v", lbl.Component)
		}
	})

	t.Run("non_null_omitzero_empty_slices", func(t *testing.T) {
		row := ActionRowComponent{Type: ActionRowComponentType, Components: []ActionRowChildComponent{}}
		raw := string(mustMarshalJSON(t, row))
		if !strings.Contains(raw, `"components":[]`) {
			t.Fatalf("expected non-nil empty slice to serialize as [], got %s", raw)
		}
	})

	t.Run("escaped_keys_null_fields_and_unknown_fields", func(t *testing.T) {
		got, err := UnmarshalComponent([]byte(`{"\u0074ype":10,"content":"escaped"}`))
		if err != nil {
			t.Fatalf("escaped type key unmarshal failed: %v", err)
		}

		td, ok := got.(TextDisplayComponent)
		if !ok || td.Content != "escaped" {
			t.Fatalf("unexpected escaped component: %#v", got)
		}

		var lbl LabelComponent
		if err := json.Unmarshal([]byte(`{"type":18,"label":"L","component":null,"unknown_field":{"a":1}}`), &lbl); err != nil {
			t.Fatalf("label with null component failed: %v", err)
		}

		if lbl.Component != nil || lbl.Label != "L" {
			t.Fatalf("unexpected label: %#v", lbl)
		}

		if err := json.Unmarshal([]byte(`null`), &lbl); err != nil || lbl.Label != "" {
			t.Fatalf("null label failed: %#v, err=%v", lbl, err)
		}

		var row ActionRowComponent
		if err := json.Unmarshal([]byte(`{"type":1,"id":null,"components":null,"extra":1}`), &row); err != nil || row.ID != 0 {
			t.Fatalf("null action row fields failed: %#v, err=%v", row, err)
		}

		if err := json.Unmarshal([]byte(`null`), &row); err != nil {
			t.Fatalf("null action row failed: %v", err)
		}

		var container ContainerComponent
		if err := json.Unmarshal([]byte(`{"type":17,"spoiler":null,"components":null,"extra":[1,2]}`), &container); err != nil {
			t.Fatalf("null container fields failed: %v", err)
		}

		if err := json.Unmarshal([]byte(`null`), &container); err != nil {
			t.Fatalf("null container failed: %v", err)
		}

		var sec SectionComponent
		if err := json.Unmarshal([]byte(`{"type":9,"components":null,"accessory":null,"extra":true}`), &sec); err != nil {
			t.Fatalf("null section fields failed: %v", err)
		}

		if err := json.Unmarshal([]byte(`null`), &sec); err != nil {
			t.Fatalf("null section failed: %v", err)
		}

		invalidInputs := []struct {
			dst any
			raw string
		}{
			{&lbl, `{"type":18,"id":"bad"}`},
			{&lbl, `{"type":999}`},
			{&lbl, `{"type":18,"label":123}`},
			{&lbl, `{"type":18,"id":true}`},
			{&row, `{"type":1,"id":4294967296}`},
			{&row, `{"type":1,"components":123}`},
			{&container, `{"type":17,"spoiler":1}`},
			{&container, `{"type":17,"spoiler":null}`},
			{&sec, `{"type":9,"components":123}`},
			{&sec, `{"type":9,"components":[{"type":99}]}`},
			{&lbl, `[]`},
			{&row, `[]`},
			{&container, `[]`},
			{&sec, `[]`},
		}

		for _, tc := range invalidInputs {
			if tc.raw == `{"type":17,"spoiler":null}` {
				if err := json.Unmarshal([]byte(tc.raw), tc.dst); err != nil {
					t.Fatalf("unexpected error for null bool: %v", err)
				}

				continue
			}

			if err := json.Unmarshal([]byte(tc.raw), tc.dst); err == nil {
				t.Fatalf("expected error for %T input %s", tc.dst, tc.raw)
			}
		}

		// Direct UnmarshalJSON method calls (std/v1 interface) + trailing token checks
		var directRow ActionRowComponent
		if err := directRow.UnmarshalJSON([]byte(`{"type":1,"id":7,"components":[{"type":2,"style":1,"label":"B","custom_id":"b"}]}`)); err != nil {
			t.Fatalf("ActionRowComponent.UnmarshalJSON failed: %v", err)
		}

		if directRow.ID != 7 || len(directRow.Components) != 1 {
			t.Fatalf("unexpected directRow: %#v", directRow)
		}

		if err := directRow.UnmarshalJSON([]byte(`{"type":1} trailing`)); err == nil {
			t.Fatal("expected trailing token error")
		}

		var directLbl LabelComponent
		if err := directLbl.UnmarshalJSON([]byte(`{"\u0074ype":18,"\u006cabel":"X\u0059","description":null,"component":{"type":23,"custom_id":"c"}}`)); err != nil {
			t.Fatalf("LabelComponent.UnmarshalJSON failed: %v", err)
		}

		if directLbl.Label != "XY" || directLbl.Description != "" || directLbl.Component == nil {
			t.Fatalf("unexpected directLbl: %#v", directLbl)
		}

		var directContainer ContainerComponent
		if err := directContainer.UnmarshalJSON([]byte(`{"type":17,"id":1,"components":[{"type":14,"divider":true}]}`)); err != nil || len(directContainer.Components) != 1 {
			t.Fatalf("ContainerComponent.UnmarshalJSON failed: err=%v, got=%#v", err, directContainer)
		}

		var directSec SectionComponent
		if err := directSec.UnmarshalJSON([]byte(`{"type":9,"id":2,"components":[{"type":10,"content":"hi"}]}`)); err != nil || len(directSec.Components) != 1 {
			t.Fatalf("SectionComponent.UnmarshalJSON failed: err=%v, got=%#v", err, directSec)
		}

		// Malformed peekComponentType cases (escapes, nested objects/strings, bad numbers)
		malformed := []string{
			`{,"type":2}`,
			`{"a":"v\"w","b":{"k":"v\"w","c":[1,2]},"type":2}`,
			`{"a":{"unclosed":1`,
			`{"a":"unterminated`,
			`{"a":123, "b":`,
			`{"type":01}`,
			`{"type":256}`,
			`{"type":1.5}`,
			`{"\u0074ype":"str"}`,
			`{"\u0074ype":300}`,
			`{"\u0074ype":true}`,
			`{"missing_colon" 2}`,
			`{123:2}`,
			`{"\u0061":1}`,
		}

		for i, payload := range malformed {
			_, err := UnmarshalComponent([]byte(payload))
			if i == 1 {
				if err == nil {
					// Extra check: valid skipped escapes/strings still decode if complete, wait payload 1 has {"a":..., "type":2} -> valid ButtonComponent!
					continue
				}

				t.Fatalf("expected valid component with skipped escapes, got err=%v", err)
			}

			if err == nil {
				t.Fatalf("expected error for malformed payload %s", payload)
			}
		}
	})
}

func TestComponentUnionInterfaces(t *testing.T) {
	actionRow := ActionRowComponent{}
	button := ButtonComponent{}
	stringSelect := StringSelectComponent{}
	textInput := TextInputComponent{}
	selectComp := SelectComponent{}
	section := SectionComponent{}
	textDisplay := TextDisplayComponent{}
	thumbnail := ThumbnailComponent{}
	mediaGallery := MediaGalleryComponent{}
	fileComp := FileComponent{}
	separator := SeparatorComponent{}
	container := ContainerComponent{}
	label := LabelComponent{}
	fileUpload := FileUploadComponent{}
	radioGroup := RadioGroupComponent{}
	checkboxGroup := CheckboxGroupComponent{}
	checkbox := CheckboxComponent{}

	layout := []LayoutComponent{actionRow, section, separator, container, label}
	for _, c := range layout {
		c._lcmp()
	}

	interactive := []InteractiveComponent{button, stringSelect, textInput, selectComp, fileUpload, radioGroup, checkboxGroup, checkbox}
	for _, c := range interactive {
		c._icmp()
	}

	content := []ContentComponent{textDisplay, thumbnail, mediaGallery, fileComp}
	for _, c := range content {
		c._ccmp()
	}

	accessories := []AccessoryComponent{button, thumbnail}
	for _, c := range accessories {
		c._acmp()
	}

	messageComps := []MessageComponent{actionRow, section, textDisplay, mediaGallery, fileComp, separator, container}
	for _, c := range messageComps {
		c._messagecmp()
	}

	modalComps := []ModalComponent{textDisplay, label}
	for _, c := range modalComps {
		c._modalcmp()
	}

	labelChildren := []LabelChildComponent{stringSelect, textInput, selectComp, fileUpload, radioGroup, checkboxGroup, checkbox}
	for _, c := range labelChildren {
		c._lblcmp()
	}

	containerChildren := []ContainerChildComponent{actionRow, section, textDisplay, mediaGallery, fileComp, separator}
	for _, c := range containerChildren {
		c._containercmp()
	}

	actionRowChildren := []ActionRowChildComponent{button, stringSelect, selectComp}
	for _, c := range actionRowChildren {
		c._arowcmp()
	}

	all := []AnyComponent{
		actionRow, button, stringSelect, textInput, selectComp,
		section, textDisplay, thumbnail, mediaGallery, fileComp,
		separator, container, label, fileUpload, radioGroup,
		checkboxGroup, checkbox,
	}

	for _, c := range all {
		c._kind()
	}
}

func TestMinValuesZeroOneUnsetSemantics(t *testing.T) {
	zero := uint8(0)
	one := uint8(1)

	tests := []struct {
		component AnyComponent
		wantSub   string
		name      string
		wantOmit  bool
	}{
		{
			name:      "string_select_unset",
			component: StringSelectComponent{Type: StringSelectComponentType, CustomID: "s"},
			wantOmit:  true,
		},
		{
			name:      "string_select_zero",
			component: StringSelectComponent{Type: StringSelectComponentType, CustomID: "s", MinValues: &zero, Required: new(false)},
			wantSub:   `"min_values":0`,
		},
		{
			name:      "string_select_one",
			component: StringSelectComponent{Type: StringSelectComponentType, CustomID: "s", MinValues: &one},
			wantSub:   `"min_values":1`,
		},
		{
			name:      "entity_select_zero",
			component: SelectComponent{Type: RoleSelectComponentType, CustomID: "r", MinValues: &zero, Required: new(false)},
			wantSub:   `"min_values":0`,
		},
		{
			name:      "file_upload_zero",
			component: FileUploadComponent{Type: FileUploadComponentType, CustomID: "f", MinValues: &zero, Required: new(false)},
			wantSub:   `"min_values":0`,
		},
		{
			name:      "checkbox_group_zero",
			component: CheckboxGroupComponent{Type: CheckboxGroupComponentType, CustomID: "c", MinValues: &zero, Required: new(false)},
			wantSub:   `"min_values":0`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := string(mustMarshalJSON(t, tc.component))
			if tc.wantOmit {
				if strings.Contains(raw, `"min_values"`) {
					t.Fatalf("expected min_values omitted when nil, got %s", raw)
				}

				return
			}

			if !strings.Contains(raw, tc.wantSub) {
				t.Fatalf("expected %s in %s", tc.wantSub, raw)
			}

			decoded, err := UnmarshalComponent([]byte(raw))
			if err != nil {
				t.Fatalf("round-trip UnmarshalComponent failed: %v", err)
			}

			switch v := decoded.(type) {
			case StringSelectComponent:
				if v.MinValues == nil {
					t.Fatal("expected non-nil MinValues after round-trip")
				}
			case SelectComponent:
				if v.MinValues == nil {
					t.Fatal("expected non-nil MinValues after round-trip")
				}
			case FileUploadComponent:
				if v.MinValues == nil {
					t.Fatal("expected non-nil MinValues after round-trip")
				}
			case CheckboxGroupComponent:
				if v.MinValues == nil {
					t.Fatal("expected non-nil MinValues after round-trip")
				}
			}
		})
	}
}

func TestEnumAndFlagValues(t *testing.T) {
	checkBit := func(name string, got BitSet, bit uint8) {
		t.Helper()
		if uint64(got) != uint64(1)<<bit {
			t.Errorf("%s = %d, want 1 << %d", name, got, bit)
		}
	}

	checkBit("SendVoiceMessagesPermissionFlag", SendVoiceMessagesPermissionFlag, 46)
	checkBit("SetVoiceChannelStatusPermissionFlag", SetVoiceChannelStatusPermissionFlag, 48)
	checkBit("SendPollsPermissionFlag", SendPollsPermissionFlag, 49)
	checkBit("UseExternalAppsPermissionFlag", UseExternalAppsPermissionFlag, 50)
	checkBit("PinMessagesPermissionFlag", PinMessagesPermissionFlag, 51)
	checkBit("BypassSlowmodePermissionFlag", BypassSlowmodePermissionFlag, 52)
	checkBit("SuppressNotificationsMessageFlag", SuppressNotificationsMessageFlag, 12)
	checkBit("IsComponentsV2MessageFlag", IsComponentsV2MessageFlag, 15)
	checkBit("DidRejoinMemberFlag", DidRejoinMemberFlag, 0)
	checkBit("AutomodQuarantinedGuildTagMemberFlag", AutomodQuarantinedGuildTagMemberFlag, 10)
	checkBit("IsSpoilerAttachmentFlag", IsSpoilerAttachmentFlag, 3)

	if uint8(ComponentType(23)) != uint8(CheckboxComponentType) ||
		uint8(ContainerComponentType) != 17 ||
		uint8(LabelComponentType) != 18 ||
		uint8(FileUploadComponentType) != 19 ||
		uint8(RadioGroupComponentType) != 21 ||
		uint8(CheckboxGroupComponentType) != 22 {
		t.Fatal("unexpected component type constant value")
	}

	if uint8(PollResultMessageType) != 46 || uint8(PurchaseNotificationMessageType) != 44 {
		t.Fatal("unexpected message type constant value")
	}

	if uint8(GuildForumChannelType) != 15 || uint8(GuildMediaChannelType) != 16 {
		t.Fatal("unexpected channel type constant value")
	}
}

func benchmarkSampleContainer() ContainerComponent {
	return ContainerComponent{
		Type:        ContainerComponentType,
		ID:          1,
		AccentColor: 0x5865F2,
		Components: []ContainerChildComponent{
			TextDisplayComponent{Type: TextDisplayComponentType, ID: 2, Content: "# Release Notes\nEverything is fast and clean."},
			SeparatorComponent{Type: SeparatorComponentType, ID: 3, Divider: new(true), Spacing: SmallSeparatorSpacing},
			SectionComponent{
				Type: SectionComponentType,
				ID:   4,
				Components: []TextDisplayComponent{
					{Type: TextDisplayComponentType, Content: "Click to inspect deployment metrics"},
					{Type: TextDisplayComponentType, Content: "Updated just now"},
				},
				Accessory: ButtonComponent{
					Type:     ButtonComponentType,
					Style:    PrimaryButtonStyle,
					Label:    "Details",
					CustomID: "view_metrics",
				},
			},
			ActionRowComponent{
				Type: ActionRowComponentType,
				ID:   5,
				Components: []ActionRowChildComponent{
					StringSelectComponent{
						Type:        StringSelectComponentType,
						CustomID:    "env_select",
						Placeholder: "Select region",
						MinValues:   new(uint8(1)),
						MaxValues:   1,
						Options: []SelectMenuOption{
							{Label: "US East", Value: "us-east", Default: true},
							{Label: "EU Central", Value: "eu-central"},
							{Label: "AP South", Value: "ap-south"},
						},
					},
				},
			},
		},
	}
}

func BenchmarkComponent_MarshalContainer(b *testing.B) {
	c := benchmarkSampleContainer()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(&c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalComponent_Button(b *testing.B) {
	raw := []byte(`{"type":2,"id":10,"style":1,"label":"Confirm","custom_id":"confirm_action","emoji":{"id":"555","name":"check"}}`)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := UnmarshalComponent(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContainerComponent_UnmarshalJSON(b *testing.B) {
	c := benchmarkSampleContainer()
	raw, err := json.Marshal(&c)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var dst ContainerComponent
		if err := json.Unmarshal(raw, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkActionRowComponent_UnmarshalJSON(b *testing.B) {
	row := ActionRowComponent{
		Type: ActionRowComponentType,
		ID:   1,
		Components: []ActionRowChildComponent{
			ButtonComponent{Type: ButtonComponentType, Style: PrimaryButtonStyle, Label: "One", CustomID: "1"},
			ButtonComponent{Type: ButtonComponentType, Style: SecondaryButtonStyle, Label: "Two", CustomID: "2"},
			ButtonComponent{Type: ButtonComponentType, Style: SuccessButtonStyle, Label: "Three", CustomID: "3"},
		},
	}

	raw, err := json.Marshal(&row)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var dst ActionRowComponent
		if err := json.Unmarshal(raw, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSectionComponent_UnmarshalJSON(b *testing.B) {
	sec := SectionComponent{
		Type: SectionComponentType,
		ID:   9,
		Components: []TextDisplayComponent{
			{Type: TextDisplayComponentType, Content: "Section header"},
			{Type: TextDisplayComponentType, Content: "Section body"},
		},
		Accessory: ThumbnailComponent{
			Type:        ThumbnailComponentType,
			Description: "Preview",
			Media:       UnfurledMediaItem{URL: "https://example.com/preview.png", Width: 512, Height: 512},
		},
	}

	raw, err := json.Marshal(&sec)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var dst SectionComponent
		if err := json.Unmarshal(raw, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLabelComponent_UnmarshalJSON(b *testing.B) {
	lbl := LabelComponent{
		Type:        LabelComponentType,
		ID:          18,
		Label:       "Username",
		Description: "Enter handle",
		Component: TextInputComponent{
			Type:        TextInputComponentType,
			CustomID:    "username",
			Style:       ShortTextInputStyle,
			Placeholder: "@user",
			Required:    new(true),
		},
	}

	raw, err := json.Marshal(&lbl)
	if err != nil {
		b.Fatal(err)
	}

	var rawValue jsontext.Value = raw
	b.ReportAllocs()
	for b.Loop() {
		var dst LabelComponent
		if err := json.Unmarshal(rawValue, &dst); err != nil {
			b.Fatal(err)
		}
	}
}
