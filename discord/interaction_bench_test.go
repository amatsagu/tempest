package discord

import (
	json "encoding/json/v2"
	"testing"
)

func TestInteractionEdgeCases(t *testing.T) {
	t.Run("command_option_and_focused_edge_cases", func(t *testing.T) {
		itx := CommandInteraction{
			Data: CommandInteractionData{
				Options: []CommandInteractionOption{
					{Name: "sub", Type: SubCommandOptionType},
				},
			},
		}

		if val, ok := itx.GetOptionValue("missing"); ok || val != nil {
			t.Fatalf("expected missing option to return nil, false; got %v, %v", val, ok)
		}

		empty := CommandInteraction{}
		if val, ok := empty.GetOptionValue("any"); ok || val != nil {
			t.Fatalf("expected empty options to return nil, false; got %v, %v", val, ok)
		}

		assertPanics(t, "GetFocusedValue with no focused option", func() {
			_, _ = empty.GetFocusedValue()
		})
	})

	t.Run("resolved_empty_and_missing_member_user", func(t *testing.T) {
		var nilResolved *InteractionDataResolved
		if nilResolved.ResolveUser(UserID{Snowflake: 1}).ID.Snowflake != 0 {
			t.Fatal("expected empty User from nil resolved")
		}

		if len(nilResolved.ResolveMember(UserID{Snowflake: 1}).Nickname) != 0 {
			t.Fatal("expected empty Member from nil resolved")
		}

		if nilResolved.ResolveChannel(ChannelID{Snowflake: 1}).ID.Snowflake != 0 {
			t.Fatal("expected empty PartialChannel from nil resolved")
		}

		if nilResolved.ResolveMessage(MessageID{Snowflake: 1}).ID.Snowflake != 0 {
			t.Fatal("expected empty Message from nil resolved")
		}

		if nilResolved.ResolveAttachment(AttachmentID{Snowflake: 1}).ID.Snowflake != 0 {
			t.Fatal("expected empty Attachment from nil resolved")
		}

		resolved := &InteractionDataResolved{
			Users:       map[UserID]User{},
			Members:     map[UserID]Member{{Snowflake: 1}: {Nickname: "solo"}},
			Roles:       map[RoleID]Role{},
			Channels:    map[ChannelID]PartialChannel{},
			Messages:    map[MessageID]Message{},
			Attachments: map[AttachmentID]Attachment{},
		}

		member := resolved.ResolveMember(UserID{Snowflake: 1})
		if member.Nickname != "solo" || member.User != nil {
			t.Fatalf("unexpected member without matching user: %+v", member)
		}

		if resolved.ResolveUser(UserID{Snowflake: 99}).ID.Snowflake != 0 {
			t.Fatal("expected missing user")
		}

		if resolved.ResolveChannel(ChannelID{Snowflake: 99}).ID.Snowflake != 0 {
			t.Fatal("expected missing channel")
		}

		if resolved.ResolveMessage(MessageID{Snowflake: 99}).ID.Snowflake != 0 {
			t.Fatal("expected missing message")
		}

		if resolved.ResolveAttachment(AttachmentID{Snowflake: 99}).ID.Snowflake != 0 {
			t.Fatal("expected missing attachment")
		}
	})

	t.Run("modal_missing_and_unmarshal_errors", func(t *testing.T) {
		modal := &ModalInteraction{
			Data: ModalInteractionData{
				CustomID: "m",
				Resolved: &InteractionDataResolved{
					Attachments: map[AttachmentID]Attachment{},
				},
				Components: []ModalComponent{
					TextDisplayComponent{Type: TextDisplayComponentType, Content: "note"},
					LabelComponent{Type: LabelComponentType, Label: "upload", Component: FileUploadComponent{
						Type:     FileUploadComponentType,
						CustomID: "files",
						Values:   []AttachmentID{{Snowflake: 404}},
					}},
				},
			},
		}

		if modal.GetInputValue("x") != "" {
			t.Fatal("expected empty input value")
		}

		if modal.GetFileUploadValues("x") != nil {
			t.Fatal("expected nil file upload values")
		}

		if got := modal.GetFileUploadAttachments("files"); len(got) != 0 {
			t.Fatalf("expected empty file upload attachments when unresolved, got %#v", got)
		}

		if v, ok := modal.GetRadioGroupValue("x"); ok || v != "" {
			t.Fatalf("expected empty radio group value, got %q, %v", v, ok)
		}

		if modal.GetCheckboxGroupValues("x") != nil {
			t.Fatal("expected nil checkbox group values")
		}

		if v, ok := modal.GetCheckboxValue("x"); ok || v {
			t.Fatalf("expected false, false checkbox value, got %v, %v", v, ok)
		}

		if modal.GetStringSelectValues("x") != nil {
			t.Fatal("expected nil string select values")
		}

		if modal.GetSelectValues("x") != nil {
			t.Fatal("expected nil select values")
		}

		var direct ModalInteractionData
		if err := direct.UnmarshalJSON([]byte(`null`)); err != nil || direct.CustomID != "" {
			t.Fatalf("null ModalInteractionData failed: %+v, err=%v", direct, err)
		}

		if err := direct.UnmarshalJSON([]byte(`{"custom_id":"ok","resolved":null,"extra":123}`)); err != nil || direct.CustomID != "ok" {
			t.Fatalf("ModalInteractionData with null resolved failed: %+v, err=%v", direct, err)
		}

		if err := direct.UnmarshalJSON([]byte(`{"resolved":"bad"}`)); err == nil {
			t.Fatal("expected error for invalid resolved object")
		}

		var bad ModalInteractionData
		if err := json.Unmarshal([]byte(`{`), &bad); err == nil {
			t.Fatal("expected error for malformed JSON")
		}

		if err := json.Unmarshal([]byte(`{"components":[{"type":99}]}`), &bad); err == nil {
			t.Fatal("expected error for unknown modal component")
		}

		if err := json.Unmarshal([]byte(`{"components":[{"type":14,"divider":true}]}`), &bad); err != nil {
			t.Fatalf("unexpected error when non-ModalComponent present: %v", err)
		}

		if len(bad.Components) != 0 {
			t.Fatalf("expected non-ModalComponent to be skipped, got %#v", bad.Components)
		}
	})
}

func BenchmarkCommandInteraction_JSON(b *testing.B) {
	raw := []byte(`{
		"id": "123456789012345678",
		"application_id": "223456789012345678",
		"guild_id": "323456789012345678",
		"channel_id": "423456789012345678",
		"type": 2,
		"token": "interaction-token",
		"app_permissions": "2147483648",
		"locale": "en-US",
		"guild_locale": "en-US",
		"member": {
			"joined_at": "2024-01-01T00:00:00Z",
			"roles": ["555", "666"],
			"permissions": "8",
			"flags": 0,
			"user": {
				"id": "777",
				"username": "amatsagu",
				"global_name": "Amatsagu"
			}
		},
		"data": {
			"id": "888",
			"name": "ban",
			"type": 1,
			"options": [
				{"name": "user", "type": 6, "value": "999"},
				{"name": "reason", "type": 3, "value": "spam"}
			],
			"resolved": {
				"users": {
					"999": {"id": "999", "username": "target"}
				}
			}
		}
	}`)

	b.ReportAllocs()
	for b.Loop() {
		var itx CommandInteraction
		if err := json.Unmarshal(raw, &itx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkModalInteraction_Unmarshal(b *testing.B) {
	raw := []byte(`{
		"id": "123456789012345678",
		"application_id": "223456789012345678",
		"type": 5,
		"token": "modal-token",
		"app_permissions": "2048",
		"user": {"id": "777", "username": "amatsagu"},
		"data": {
			"custom_id": "feedback_modal",
			"components": [
				{
					"type": 18,
					"id": 1,
					"label": "Summary",
					"component": {"type": 4, "id": 2, "custom_id": "summary", "value": "Fast and clean"}
				},
				{
					"type": 18,
					"id": 3,
					"label": "Topics",
					"component": {"type": 3, "id": 4, "custom_id": "topics", "values": ["perf", "api"]}
				},
				{
					"type": 18,
					"id": 5,
					"label": "Confirm",
					"component": {"type": 23, "id": 6, "custom_id": "confirm", "value": true}
				}
			]
		}
	}`)

	b.ReportAllocs()
	for b.Loop() {
		var modal ModalInteraction
		if err := json.Unmarshal(raw, &modal); err != nil {
			b.Fatal(err)
		}
	}
}
