package discord

import (
	json "encoding/json/v2"
	"strings"
	"testing"
	"time"
)

func TestChannelTypeMarshalJSON(t *testing.T) {
	types := []ChannelType{
		GuildTextChannelType,
		DMChannelType,
		GuildVoiceChannelType,
		GroupDMChannelType,
		GuildCategoryChannelType,
		GuildAnnouncementChannelType,
		GuildAnnouncementThreadChannelType,
		GuildPublicThreadChannelType,
		GuildPrivateThreadChannelType,
		GuildStageVoiceChannelType,
		GuildDirectoryChannelType,
		GuildForumChannelType,
		GuildMediaChannelType,
	}

	raw, err := json.Marshal(types)
	if err != nil {
		t.Fatalf("marshal []ChannelType failed: %v", err)
	}

	want := "[0,1,2,3,4,5,10,11,12,13,14,15,16]"
	if string(raw) != want {
		t.Fatalf("got %s, want %s", raw, want)
	}
}

func TestEmojiMention(t *testing.T) {
	unicode := Emoji{Name: "🔥"}
	if got := unicode.Mention(); got != "🔥" {
		t.Fatalf("unicode Mention = %q, want 🔥", got)
	}

	static := Emoji{ID: EmojiID{Snowflake: 12345}, Name: "pepe"}
	if got := static.Mention(); got != "<:pepe:12345>" {
		t.Fatalf("static Mention = %q, want <:pepe:12345>", got)
	}

	animated := Emoji{ID: EmojiID{Snowflake: 67890}, Name: "dance", Animated: true}
	if got := animated.Mention(); got != "<a:dance:67890>" {
		t.Fatalf("animated Mention = %q, want <a:dance:67890>", got)
	}
}

func TestNonceUnmarshalJSON(t *testing.T) {
	t.Run("valid_cases", func(t *testing.T) {
		tests := []struct {
			name string
			json string
			want Nonce
		}{
			{"string", `{"nonce":"abc-123"}`, "abc-123"},
			{"escaped_string", `{"nonce":"a\"b"}`, `a"b`},
			{"number", `{"nonce":987654321}`, "987654321"},
			{"null", `{"nonce":null}`, ""},
			{"empty_string", `{"nonce":""}`, ""},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				var msg Message
				if err := json.Unmarshal([]byte(tc.json), &msg); err != nil {
					t.Fatalf("unmarshal %s failed: %v", tc.json, err)
				}
				if msg.Nonce != tc.want {
					t.Fatalf("got %q, want %q", msg.Nonce, tc.want)
				}
			})
		}
	})

	t.Run("direct_call_and_errors", func(t *testing.T) {
		n := Nonce("preset")
		if err := n.UnmarshalJSON([]byte("   ")); err != nil || n != "" {
			t.Fatalf("empty bytes: got n=%q err=%v", n, err)
		}

		n = "preset"
		if err := n.UnmarshalJSON([]byte(`"simple-nonce"`)); err != nil || n != "simple-nonce" {
			t.Fatalf("simple quoted bytes: got n=%q err=%v", n, err)
		}

		n = "preset"
		if err := n.UnmarshalJSON([]byte(`"hello-🚀"`)); err != nil || n != "hello-🚀" {
			t.Fatalf("utf8 quoted bytes: got n=%q err=%v", n, err)
		}

		n = "preset"
		if err := n.UnmarshalJSON([]byte("\"bad\nline\"")); err == nil {
			t.Fatal("expected error for unescaped control character in JSON string")
		}

		n = "preset"
		if err := n.UnmarshalJSON([]byte(`"\xff"`)); err == nil {
			t.Fatal("expected error for invalid escape / JSON string")
		}

		n = "preset"
		if err := n.UnmarshalJSON([]byte(" null ")); err != nil || n != "" {
			t.Fatalf("null bytes: got n=%q err=%v", n, err)
		}

		n = "preset"
		if err := n.UnmarshalJSON([]byte("12345")); err != nil || n != "12345" {
			t.Fatalf("numeric bytes: got n=%q err=%v", n, err)
		}

		if err := n.UnmarshalJSON([]byte(`"unterminated`)); err == nil {
			t.Fatal("expected error for malformed quoted string")
		}
	})
}

func TestMessageUnmarshalAndSerialization(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	expiry := now.Add(time.Hour)

	msg := Message{
		ID:        MessageID{Snowflake: 111},
		ChannelID: ChannelID{Snowflake: 222},
		GuildID:   GuildID{Snowflake: 333},
		Author: &User{
			ID:       UserID{Snowflake: 444},
			Username: "bot",
		},
		Content:   "Status update",
		Timestamp: &now,
		Nonce:     "req-1",
		Type:      DefaultMessageType,
		Flags:     IsComponentsV2MessageFlag | EphemeralMessageFlag,
		Embeds: []Embed{
			{
				Title:       "Metrics",
				Description: "All systems green",
				Color:       0x00FF00,
				Flags:       ContainsExplicitContentEmbedFlag,
				Author:      &EmbedAuthor{Name: "Monitor"},
				Footer:      &EmbedFooter{Text: "v2"},
				Thumbnail:   &EmbedThumbnail{URL: "https://example.com/t.png"},
				Image:       &EmbedImage{URL: "https://example.com/i.png"},
				Fields:      []EmbedField{{Name: "Latency", Value: "1ms", Inline: true}},
			},
		},
		Attachments: []Attachment{
			{ID: AttachmentID{Snowflake: 555}, FileName: "log.txt", Size: 64, URL: "https://example.com/log.txt", ProxyURL: "https://media.example.com/log.txt"},
		},
		Poll: &Poll{
			Question:    PollMedia{Text: "Ship it?"},
			Answers:     []PollAnswer{{AnswerID: 1, PollMedia: PollMedia{Text: "Yes", Emoji: &Emoji{Name: "🚀"}}}},
			Duration:    24,
			Expiry:      &expiry,
			LayoutType:  DefaultPollLayoutType,
			Multiselect: false,
		},
		Components: []MessageComponent{
			ActionRowComponent{
				Type: ActionRowComponentType,
				Components: []ActionRowChildComponent{
					ButtonComponent{Type: ButtonComponentType, Style: SuccessButtonStyle, Label: "Ack", CustomID: "ack"},
				},
			},
			TextDisplayComponent{Type: TextDisplayComponentType, Content: "Footnote"},
		},
	}

	raw, err := json.Marshal(&msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Message
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.ID != msg.ID || parsed.ChannelID != msg.ChannelID || parsed.Nonce != "req-1" {
		t.Fatalf("unexpected decoded scalar fields: %+v", parsed)
	}
	if len(parsed.Components) != 2 || len(parsed.Embeds) != 1 || len(parsed.Attachments) != 1 || parsed.Poll == nil {
		t.Fatalf("missing nested slices/structs in decoded message: %#v", parsed)
	}

	t.Run("omitzero_and_message_null_edge_cases", func(t *testing.T) {
		linear := Message{Content: "hello"}
		encoded := string(mustMarshalJSON(t, &linear))
		for _, omittedKey := range []string{`"embeds"`, `"attachments"`, `"mentions"`, `"mention_roles"`, `"timestamp"`, `"components"`} {
			if strings.Contains(encoded, omittedKey) {
				t.Fatalf("expected %s to be omitted in linear message, got %s", omittedKey, encoded)
			}
		}

		clearing := Message{Embeds: []Embed{}}
		clearingEncoded := string(mustMarshalJSON(t, &clearing))
		if !strings.Contains(clearingEncoded, `"embeds":[]`) {
			t.Fatalf("expected non-nil empty embeds slice to emit [], got %s", clearingEncoded)
		}

		var msg Message
		if err := json.Unmarshal([]byte(`null`), &msg); err != nil {
			t.Fatalf("unmarshal null message failed: %v", err)
		}
		if err := json.Unmarshal([]byte(`{"unknown_prop":{"nested":1},"flags":0,"position":1,"type":0,"mention_everyone":true,"tts":true,"pinned":true}`), &msg); err != nil {
			t.Fatalf("unmarshal unknown prop failed: %v", err)
		}
		if !msg.MentionEveryone || !msg.TTS || !msg.Pinned || msg.Position != 1 {
			t.Fatalf("unexpected decoded scalar flags: %+v", msg)
		}

		full := Message{
			ID:                  MessageID{Snowflake: 1},
			ChannelID:           ChannelID{Snowflake: 2},
			GuildID:             GuildID{Snowflake: 3},
			ApplicationID:       AppID{Snowflake: 4},
			WebhookID:           5,
			Timestamp:           &now,
			EditedTimestamp:     &now,
			Author:              &User{ID: UserID{Snowflake: 6}, Username: "a"},
			Member:              &Member{JoinedAt: now, Nickname: "n"},
			Resolved:            &InteractionDataResolved{Users: map[UserID]User{{Snowflake: 6}: {Username: "a"}}},
			MessageReference:    &MessageReference{MessageID: MessageID{Snowflake: 7}, FailIfNotExists: true},
			InteractionMetadata: &MessageInteractionMetadata{ID: 8, Type: ApplicationCommandInteractionType, User: User{Username: "u"}},
			ReferencedMessage:   &Message{ID: MessageID{Snowflake: 9}, Content: "ref"},
			Mentions:            []User{{ID: UserID{Snowflake: 10}, Username: "m"}},
			MentionRoles:        []RoleID{{Snowflake: 11}},
			MentionChannels:     []ChannelMention{{ID: ChannelID{Snowflake: 12}, GuildID: GuildID{Snowflake: 3}, Name: "general", Type: GuildTextChannelType}},
			StickerItems:        []StickerItem{{ID: 13, Name: "wave", FormatType: PNGStickerFormatType}},
			Reactions:           []Reaction{{Emoji: Emoji{Name: "👍"}, Count: 2, Me: true}},
			Content:             "full message",
		}
		fullRaw := mustMarshalJSON(t, &full)
		var decodedFull Message
		if err := decodedFull.UnmarshalJSON(fullRaw); err != nil {
			t.Fatalf("Message.UnmarshalJSON(full) failed: %v", err)
		}
		if decodedFull.WebhookID != 5 || decodedFull.ReferencedMessage == nil || len(decodedFull.Reactions) != 1 || len(decodedFull.StickerItems) != 1 {
			t.Fatalf("missing full message fields: %+v", decodedFull)
		}

		ctRaw, err := GuildForumChannelType.MarshalJSON()
		if err != nil || string(ctRaw) != "15" {
			t.Fatalf("ChannelType.MarshalJSON = %q, err=%v", ctRaw, err)
		}
	})

	t.Run("unmarshal_errors_and_skip_non_message_component", func(t *testing.T) {
		var bad Message
		if err := json.Unmarshal([]byte(`{`), &bad); err == nil {
			t.Fatal("expected JSON syntax error")
		}
		if err := json.Unmarshal([]byte(`{"components":[{"type":99}]}`), &bad); err == nil {
			t.Fatal("expected component decode error")
		}
		if err := json.Unmarshal([]byte(`{"components":[{"type":18,"label":"ModalOnly","component":{"type":23,"custom_id":"c"}}]}`), &bad); err != nil {
			t.Fatalf("unexpected error when non-MessageComponent present: %v", err)
		}
		if len(bad.Components) != 0 {
			t.Fatalf("expected non-MessageComponent to be skipped, got %#v", bad.Components)
		}
	})
}

func TestCommandEntitlementAndResponseJSON(t *testing.T) {
	minVal := 1.5
	maxVal := 9.5
	cmd := Command{
		Name:                     "deploy",
		Description:              "Deploy service",
		NameLocalizations:        map[Language]string{PolishLanguage: "wdroz"},
		DescriptionLocalizations: map[Language]string{GermanLanguage: "Bereitstellen"},
		Type:                     ChatInputCommandType,
		IntegrationTypes:         []ApplicationIntegrationType{GuildInstall, UserInstall},
		Contexts:                 []InteractionContextType{GuildContextType, BotDMContextType},
		RequiredPermissions:      AdministratorPermissionFlag | ManageGuildPermissionFlag,
		Options: []CommandOption{
			{
				Name:         "count",
				Description:  "Replicas",
				Type:         NumberOptionType,
				MinValue:     &minVal,
				MaxValue:     &maxVal,
				ChannelTypes: []ChannelType{GuildTextChannelType, GuildAnnouncementChannelType},
				Choices:      []CommandOptionChoice{{Name: "Default", Value: float64(2)}},
			},
		},
	}

	cmdRaw, err := json.Marshal(&cmd)
	if err != nil {
		t.Fatalf("marshal Command failed: %v", err)
	}
	var cmdOut Command
	if err := json.Unmarshal(cmdRaw, &cmdOut); err != nil {
		t.Fatalf("unmarshal Command failed: %v", err)
	}
	if cmdOut.Name != "deploy" || cmdOut.RequiredPermissions != cmd.RequiredPermissions || len(cmdOut.Contexts) != 2 {
		t.Fatalf("unexpected decoded command: %+v", cmdOut)
	}

	now := time.Unix(1700000000, 0).UTC()
	ent := Entitlement{
		ID:            EntitlementID{Snowflake: 1},
		SkuID:         SkuID{Snowflake: 2},
		ApplicationID: AppID{Snowflake: 3},
		UserID:        UserID{Snowflake: 4},
		Type:          ApplicationSubscriptionEntitlementType,
		StartsAt:      &now,
		EndsAt:        &now,
	}
	entRaw, err := json.Marshal(&ent)
	if err != nil {
		t.Fatalf("marshal Entitlement failed: %v", err)
	}
	var entOut Entitlement
	if err := json.Unmarshal(entRaw, &entOut); err != nil {
		t.Fatalf("unmarshal Entitlement failed: %v", err)
	}
	if entOut.ID != ent.ID || entOut.Type != ApplicationSubscriptionEntitlementType {
		t.Fatalf("unexpected entitlement: %+v", entOut)
	}

	resp := ResponseMessage{
		Type: ChannelMessageWithSourceResponseType,
		Data: &ResponseMessageData{
			Content: "ok",
			AllowedMentions: &AllowedMentions{
				Parse:       []AllowedMentionsType{AllowedUsersMentionType},
				RepliedUser: true,
			},
			Flags: EphemeralMessageFlag,
		},
	}
	respRaw, err := json.Marshal(&resp)
	if err != nil {
		t.Fatalf("marshal ResponseMessage failed: %v", err)
	}
	if !strings.Contains(string(respRaw), `"content":"ok"`) {
		t.Fatalf("unexpected response payload: %s", respRaw)
	}

	if !AllPermissionFlags.Has(AllChannelPermissionFlags, AllTextPermissionFlags, AllThreadPermissionFlags, AllVoicePermissionFlags) {
		t.Fatal("AllPermissionFlags missing expected subsets")
	}
}

func TestCommandOptionMinValuesAndEmbedReplySemantics(t *testing.T) {
	t.Run("command_option_min_values", func(t *testing.T) {
		tests := []struct {
			minValue *float64
			wantSub  string
			name     string
			wantOmit bool
		}{
			{name: "unset_nil", minValue: nil, wantOmit: true},
			{name: "zero", minValue: new(float64(0)), wantSub: `"min_value":0`},
			{name: "one", minValue: new(float64(1)), wantSub: `"min_value":1`},
			{name: "hundred", minValue: new(float64(100)), wantSub: `"min_value":100`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				opt := CommandOption{
					Name:        "amount",
					Description: "Amount",
					Type:        IntegerOptionType,
					MinValue:    tc.minValue,
				}
				raw := string(mustMarshalJSON(t, opt))
				if tc.wantOmit {
					if strings.Contains(raw, `"min_value"`) {
						t.Fatalf("expected min_value omitted when nil, got %s", raw)
					}
					return
				}
				if !strings.Contains(raw, tc.wantSub) {
					t.Fatalf("expected %s in %s", tc.wantSub, raw)
				}

				var decoded CommandOption
				if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
					t.Fatalf("unmarshal CommandOption failed: %v", err)
				}
				if decoded.MinValue == nil || *decoded.MinValue != *tc.minValue {
					t.Fatalf("decoded MinValue = %v, want %v", decoded.MinValue, *tc.minValue)
				}
			})
		}
	})

	t.Run("reply_embed_then_no_embeds_vanish_vs_clear_empty_array", func(t *testing.T) {
		// 1. Initial reply containing an embed.
		withEmbed := ResponseMessage{
			Type: ChannelMessageWithSourceResponseType,
			Data: &ResponseMessageData{
				Content: "Initial reply",
				Embeds:  []Embed{{Title: "Report", Description: "Details"}},
			},
		}
		rawWithEmbed := string(mustMarshalJSON(t, &withEmbed))
		if !strings.Contains(rawWithEmbed, `"embeds":[{`) {
			t.Fatalf("expected embeds array with embed object, got %s", rawWithEmbed)
		}

		// 2. Follow-up / reply with nil Embeds -> "embeds" field vanishes completely.
		noEmbeds := ResponseMessage{
			Type: UpdateMessageResponseType,
			Data: &ResponseMessageData{
				Content: "Text only",
			},
		}
		rawNoEmbeds := string(mustMarshalJSON(t, &noEmbeds))
		if strings.Contains(rawNoEmbeds, `"embeds"`) {
			t.Fatalf("expected embeds key to vanish when nil, got %s", rawNoEmbeds)
		}

		// 3. Edit reply with non-nil empty slice []Embed{} -> emits "embeds":[] to clear embeds.
		clearedEmbeds := ResponseMessage{
			Type: UpdateMessageResponseType,
			Data: &ResponseMessageData{
				Content: "Cleared",
				Embeds:  []Embed{},
			},
		}
		rawCleared := string(mustMarshalJSON(t, &clearedEmbeds))
		if !strings.Contains(rawCleared, `"embeds":[]`) {
			t.Fatalf("expected explicit empty array \"embeds\":[], got %s", rawCleared)
		}

		// Same lifecycle on Message directly.
		msgWithEmbed := string(mustMarshalJSON(t, &Message{Embeds: []Embed{{Title: "v1"}}}))
		if !strings.Contains(msgWithEmbed, `"embeds":[{`) {
			t.Fatalf("expected embed on Message, got %s", msgWithEmbed)
		}
		msgVanish := string(mustMarshalJSON(t, &Message{Content: "plain"}))
		if strings.Contains(msgVanish, `"embeds"`) || strings.Contains(msgVanish, `"channel_id"`) || strings.Contains(msgVanish, `"id"`) {
			t.Fatalf("expected nil/zero fields to vanish on plain Message, got %s", msgVanish)
		}
		msgClear := string(mustMarshalJSON(t, &Message{Content: "plain", Embeds: []Embed{}}))
		if !strings.Contains(msgClear, `"embeds":[]`) {
			t.Fatalf("expected \"embeds\":[] on cleared Message, got %s", msgClear)
		}

		// AllowedMentions parse=[] / roles=[] semantics.
		am := AllowedMentions{
			Parse: []AllowedMentionsType{},
			Roles: []RoleID{},
		}
		rawAM := string(mustMarshalJSON(t, am))
		if !strings.Contains(rawAM, `"parse":[]`) || !strings.Contains(rawAM, `"roles":[]`) {
			t.Fatalf("expected empty parse and roles arrays, got %s", rawAM)
		}

		// Context menu command omits empty description; button partial emoji omits read-only flags; nil Entitlement dates omit null.
		ctxCmdRaw := string(mustMarshalJSON(t, Command{Name: "Inspect", Type: UserCommandType}))
		if strings.Contains(ctxCmdRaw, `"description"`) {
			t.Fatalf("expected context menu command to omit empty description, got %s", ctxCmdRaw)
		}
		btnEmojiRaw := string(mustMarshalJSON(t, ButtonComponent{
			Type:     ButtonComponentType,
			Style:    SecondaryButtonStyle,
			CustomID: "react",
			Emoji:    &Emoji{ID: EmojiID{Snowflake: 42}, Name: "wave"},
		}))
		for _, unexpected := range []string{`"require_colons"`, `"managed"`, `"animated"`, `"available"`} {
			if strings.Contains(btnEmojiRaw, unexpected) {
				t.Fatalf("expected partial button emoji to omit %s, got %s", unexpected, btnEmojiRaw)
			}
		}
		entRaw := string(mustMarshalJSON(t, Entitlement{ID: EntitlementID{Snowflake: 1}, SkuID: SkuID{Snowflake: 2}, ApplicationID: AppID{Snowflake: 3}, Type: PurchaseEntitlementType}))
		if strings.Contains(entRaw, `"starts_at"`) || strings.Contains(entRaw, `"ends_at"`) {
			t.Fatalf("expected nil entitlement timestamps to be omitted, got %s", entRaw)
		}
	})
}

func BenchmarkMessage_JSON(b *testing.B) {
	now := time.Now().UTC().Truncate(time.Second)
	msg := Message{
		ID:        MessageID{Snowflake: 8888},
		ChannelID: ChannelID{Snowflake: 9999},
		Author: &User{
			ID:       UserID{Snowflake: 1001},
			Username: "bot",
		},
		Content:   "Hello world",
		Timestamp: &now,
		Nonce:     "nonce-42",
		Type:      DefaultMessageType,
		Flags:     EphemeralMessageFlag,
	}
	data, err := json.Marshal(&msg)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed Message
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&msg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageWithComponents_JSON(b *testing.B) {
	now := time.Now().UTC().Truncate(time.Second)
	msg := Message{
		ID:        MessageID{Snowflake: 8888},
		ChannelID: ChannelID{Snowflake: 9999},
		GuildID:   GuildID{Snowflake: 7777},
		Author: &User{
			ID:       UserID{Snowflake: 1001},
			Username: "bot",
		},
		Timestamp: &now,
		Type:      DefaultMessageType,
		Flags:     IsComponentsV2MessageFlag,
		Embeds: []Embed{
			{
				Title:       "Deployment",
				Description: "Succeeded",
				Fields:      []EmbedField{{Name: "Region", Value: "eu-central", Inline: true}},
			},
		},
		Components: []MessageComponent{
			benchmarkSampleContainer(),
		},
	}
	data, err := json.Marshal(&msg)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed Message
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&msg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNonce_UnmarshalJSON(b *testing.B) {
	strPayload := []byte(`"abc-123-nonce"`)
	numPayload := []byte(`123456789012345678`)

	b.Run("string", func(b *testing.B) {
		var n Nonce
		b.ReportAllocs()
		for b.Loop() {
			if err := n.UnmarshalJSON(strPayload); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("number", func(b *testing.B) {
		var n Nonce
		b.ReportAllocs()
		for b.Loop() {
			if err := n.UnmarshalJSON(numPayload); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkChannelType_MarshalJSON(b *testing.B) {
	types := []ChannelType{
		GuildTextChannelType,
		GuildVoiceChannelType,
		GuildAnnouncementChannelType,
		GuildForumChannelType,
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(types); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCommand_JSON(b *testing.B) {
	cmd := Command{
		Name:                "ban",
		Description:         "Ban a member",
		Type:                ChatInputCommandType,
		RequiredPermissions: BanMembersPermissionFlag,
		Contexts:            []InteractionContextType{GuildContextType},
		Options: []CommandOption{
			{Name: "user", Description: "Target", Type: UserOptionType, Required: true},
			{Name: "reason", Description: "Reason", Type: StringOptionType},
		},
	}
	data, err := json.Marshal(&cmd)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed Command
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&cmd); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResponseMessage_JSON(b *testing.B) {
	resp := ResponseMessage{
		Type: ChannelMessageWithSourceResponseType,
		Data: &ResponseMessageData{
			Content: "Processed request",
			Flags:   EphemeralMessageFlag,
			Components: []MessageComponent{
				ActionRowComponent{
					Type: ActionRowComponentType,
					Components: []ActionRowChildComponent{
						ButtonComponent{Type: ButtonComponentType, Style: PrimaryButtonStyle, Label: "Done", CustomID: "done"},
					},
				},
			},
		},
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(&resp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBitSet_Operations(b *testing.B) {
	perms := AllTextPermissionFlags

	b.ReportAllocs()
	for b.Loop() {
		updated := perms.Add(AdministratorPermissionFlag).Remove(SendTTSMessagesPermissionFlag)
		if !updated.Has(ViewChannelPermissionFlag, SendMessagesPermissionFlag) || updated.Missing(AdministratorPermissionFlag) {
			b.Fatal("unexpected bitset state")
		}
	}
}
