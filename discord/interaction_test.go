package discord_test

import (
	json "encoding/json/v2"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func TestInteractionBaseUserAndResolved(t *testing.T) {
	t.Parallel()

	memberUser := &discord.User{ID: discord.UserID{10}, Username: "memberuser", GlobalName: "Member User"}
	directUser := &discord.User{ID: discord.UserID{20}, Username: "directuser"}

	inGuild := discord.Interaction{
		Member: &discord.Member{User: memberUser},
		User:   directUser,
	}
	if got := inGuild.BaseUser(); got == nil || got.ID.Snowflake != 10 || got.DisplayName() != "Member User" {
		t.Fatalf("BaseUser() in guild = %+v, want member user 10", got)
	}

	inDM := discord.Interaction{User: directUser}
	if got := inDM.BaseUser(); got == nil || got.ID.Snowflake != 20 || got.DisplayName() != "directuser" {
		t.Fatalf("BaseUser() in DM = %+v, want direct user 20", got)
	}

	var nilResolved *discord.InteractionDataResolved
	if u := nilResolved.ResolveUser(discord.UserID{1}); u.ID.Snowflake != 0 {
		t.Fatalf("nilResolved.ResolveUser(1) = %+v, want zero User", u)
	}
	if _, ok := nilResolved.ResolveRole(discord.RoleID{1}); ok {
		t.Fatal("nilResolved.ResolveRole(1) returned ok = true, want false")
	}

	resolved := &discord.InteractionDataResolved{
		Users: map[discord.UserID]discord.User{
			{10}: *memberUser,
		},
		Members: map[discord.UserID]discord.Member{
			{10}: {Nickname: "Nick"},
		},
		Roles: map[discord.RoleID]discord.Role{
			{30}: {ID: discord.RoleID{30}, Name: "Mod"},
		},
		Channels: map[discord.ChannelID]discord.PartialChannel{
			{40}: {ID: discord.ChannelID{40}, Name: "general"},
		},
		Messages: map[discord.MessageID]discord.Message{
			{50}: {ID: discord.MessageID{50}, Content: "hello"},
		},
		Attachments: map[discord.AttachmentID]discord.Attachment{
			{60}: {ID: discord.AttachmentID{60}, FileName: "test.png"},
		},
	}

	m := resolved.ResolveMember(discord.UserID{10})
	if m.Nickname != "Nick" || m.User == nil || m.User.ID.Snowflake != 10 {
		t.Fatalf("ResolveMember(10) = %+v, want Nick with bound User", m)
	}
	if r, ok := resolved.ResolveRole(discord.RoleID{30}); !ok || r.Name != "Mod" {
		t.Fatalf("ResolveRole(30) = (%+v, %t), want Mod", r, ok)
	}
	if ch := resolved.ResolveChannel(discord.ChannelID{40}); ch.Name != "general" {
		t.Fatalf("ResolveChannel(40) = %+v, want general", ch)
	}
	if msg := resolved.ResolveMessage(discord.MessageID{50}); msg.Content != "hello" {
		t.Fatalf("ResolveMessage(50) = %+v, want hello", msg)
	}
	if att := resolved.ResolveAttachment(discord.AttachmentID{60}); att.FileName != "test.png" {
		t.Fatalf("ResolveAttachment(60) = %+v, want test.png", att)
	}
}

func TestCommandAndModalInteractionHelpers(t *testing.T) {
	t.Parallel()

	cmdItx := discord.CommandInteraction{
		Data: discord.CommandInteractionData{
			Options: []discord.CommandInteractionOption{
				{Name: "query", Type: discord.StringOptionType, Value: "tempest", Focused: true},
				{Name: "count", Type: discord.IntegerOptionType, Value: float64(5)},
			},
		},
	}

	if val, ok := cmdItx.GetOptionValue("count"); !ok || val != float64(5) {
		t.Fatalf("GetOptionValue(count) = (%v, %t), want (5, true)", val, ok)
	}
	if _, ok := cmdItx.GetOptionValue("missing"); ok {
		t.Fatal("GetOptionValue(missing) returned true, want false")
	}
	if name, val := cmdItx.GetFocusedValue(); name != "query" || val != "tempest" {
		t.Fatalf("GetFocusedValue() = (%q, %v), want (\"query\", \"tempest\")", name, val)
	}

	rawModal := `{
		"id": "123",
		"application_id": "456",
		"type": 5,
		"token": "tok",
		"app_permissions": "0",
		"data": {
			"custom_id": "modal:1",
			"resolved": {
				"attachments": {
					"999": {"id": "999", "filename": "img.png", "url": "https://cdn.example/img.png", "proxy_url": "https://media.example/img.png", "size": 42}
				}
			},
			"components": [
				{"type": 18, "label": "Title", "component": {"type": 4, "style": 1, "custom_id": "in:title", "value": "Hello"}},
				{"type": 18, "label": "File", "component": {"type": 19, "custom_id": "in:file", "values": ["999"]}},
				{"type": 18, "label": "Radio", "component": {"type": 21, "custom_id": "in:radio", "value": "opt_a"}},
				{"type": 18, "label": "Checks", "component": {"type": 22, "custom_id": "in:checks", "values": ["c1", "c2"]}},
				{"type": 18, "label": "Agree", "component": {"type": 23, "custom_id": "in:agree", "value": true}},
				{"type": 18, "label": "Select", "component": {"type": 3, "custom_id": "in:str", "values": ["s1"]}},
				{"type": 18, "label": "Users", "component": {"type": 5, "custom_id": "in:usr", "values": ["777"]}}
			]
		}
	}`

	var modal discord.ModalInteraction
	if err := json.Unmarshal([]byte(rawModal), &modal); err != nil {
		t.Fatalf("Unmarshal modal interaction: %v", err)
	}

	if got := modal.GetInputValue("in:title"); got != "Hello" {
		t.Fatalf("GetInputValue = %q, want Hello", got)
	}
	if got := modal.GetInputValue(""); got != "" {
		t.Fatalf("GetInputValue(\"\") = %q, want empty", got)
	}
	if got := modal.GetFileUploadValues("in:file"); len(got) != 1 || got[0].Snowflake != 999 {
		t.Fatalf("GetFileUploadValues = %v, want [999]", got)
	}
	if atts := modal.GetFileUploadAttachments("in:file"); len(atts) != 1 || atts[0].FileName != "img.png" {
		t.Fatalf("GetFileUploadAttachments = %+v, want [img.png]", atts)
	}
	if val, ok := modal.GetRadioGroupValue("in:radio"); !ok || val != "opt_a" {
		t.Fatalf("GetRadioGroupValue = (%q, %t), want (opt_a, true)", val, ok)
	}
	if vals := modal.GetCheckboxGroupValues("in:checks"); len(vals) != 2 || vals[0] != "c1" || vals[1] != "c2" {
		t.Fatalf("GetCheckboxGroupValues = %v, want [c1 c2]", vals)
	}
	if checked, ok := modal.GetCheckboxValue("in:agree"); !ok || !checked {
		t.Fatalf("GetCheckboxValue = (%t, %t), want (true, true)", checked, ok)
	}
	if vals := modal.GetStringSelectValues("in:str"); len(vals) != 1 || vals[0] != "s1" {
		t.Fatalf("GetStringSelectValues = %v, want [s1]", vals)
	}
	if vals := modal.GetSelectValues("in:usr"); len(vals) != 1 || vals[0] != 777 {
		t.Fatalf("GetSelectValues = %v, want [777]", vals)
	}
}

func TestBitSetNonceAndMessageUnmarshal(t *testing.T) {
	t.Parallel()

	flags := discord.EphemeralMessageFlag.Add(discord.SuppressEmbedsMessageFlag)
	if !flags.Has(discord.EphemeralMessageFlag, discord.SuppressEmbedsMessageFlag) {
		t.Fatal("BitSet.Has returned false")
	}
	flags = flags.Remove(discord.SuppressEmbedsMessageFlag)
	if !flags.Missing(discord.SuppressEmbedsMessageFlag) {
		t.Fatal("BitSet.Missing returned false after Remove")
	}

	var n discord.Nonce
	if err := json.Unmarshal([]byte(`12345`), &n); err != nil || n != "12345" {
		t.Fatalf("Nonce numeric unmarshal = (%q, %v), want (\"12345\", nil)", n, err)
	}

	rawMsg := `{"id":"1","channel_id":"2","type":0,"flags":32768,"components":[{"type":10,"content":"hi"}]}`
	var msg discord.Message
	if err := json.Unmarshal([]byte(rawMsg), &msg); err != nil {
		t.Fatalf("Message.UnmarshalJSON error: %v", err)
	}
	if len(msg.Components) != 1 {
		t.Fatalf("len(msg.Components) = %d, want 1", len(msg.Components))
	}
	if td, ok := msg.Components[0].(discord.TextDisplayComponent); !ok || td.Content != "hi" {
		t.Fatalf("msg.Components[0] = %#v, want TextDisplayComponent(\"hi\")", msg.Components[0])
	}
}
