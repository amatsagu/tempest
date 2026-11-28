package discord

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"strings"
	"testing"
)

func TestJSONUtilHelpersEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("consumeJSONObjectStart", func(t *testing.T) {
		if _, err := consumeJSONObjectStart(jsontext.NewDecoder(strings.NewReader(""))); err == nil {
			t.Fatal("expected error on empty input")
		}

		isNull, err := consumeJSONObjectStart(jsontext.NewDecoder(strings.NewReader("null")))
		if err != nil || !isNull {
			t.Fatalf("got (%t, %v), want (true, nil)", isNull, err)
		}
	})

	t.Run("peekComponentType_and_fallback_errors", func(t *testing.T) {
		badPayloads := []string{
			`{"a":1 "type":2}`,
			`{"a":1,`,
			`{"a":1,"type`,
			`{"\u0074ype":1}`,
			`{"\a`,
			`{"\u0074ype":`,
			`{"\u0074ype":-1}`,
			`{"\u0074ype":1.5}`,
			`{"\u0061":`,
		}

		for _, payload := range badPayloads {
			if payload == `{"\u0074ype":1}` {
				ct, err := peekComponentType([]byte(payload))
				if err != nil || ct != ActionRowComponentType {
					t.Fatalf("escaped type = (%d, %v), want (1, nil)", ct, err)
				}

				continue
			}

			if _, err := peekComponentType([]byte(payload)); err == nil {
				t.Fatalf("expected error for %q", payload)
			}
		}

		if _, err := peekComponentTypeFallback([]byte("123")); err == nil {
			t.Fatal("expected error for non-object in peekComponentTypeFallback")
		}
	})

	t.Run("readJSONKeyBytes", func(t *testing.T) {
		if _, err := readJSONKeyBytes(jsontext.NewDecoder(strings.NewReader(""))); err == nil {
			t.Fatal("expected error on empty input")
		}

		if _, err := readJSONKeyBytes(jsontext.NewDecoder(strings.NewReader("123"))); err == nil {
			t.Fatal("expected error on non-string key")
		}

		if _, err := readJSONKeyBytes(jsontext.NewDecoder(strings.NewReader(`"\x"`))); err == nil {
			t.Fatal("expected error on invalid escaped key")
		}
	})

	t.Run("readJSONString", func(t *testing.T) {
		if _, err := readJSONString(jsontext.NewDecoder(strings.NewReader(""))); err == nil {
			t.Fatal("expected error on empty input")
		}

		if _, err := readJSONString(jsontext.NewDecoder(strings.NewReader(`"\x"`))); err == nil {
			t.Fatal("expected error on invalid escape")
		}
	})

	t.Run("readJSONBool", func(t *testing.T) {
		got, err := readJSONBool(jsontext.NewDecoder(strings.NewReader("false")))
		if err != nil || got {
			t.Fatalf("got (%t, %v), want (false, nil)", got, err)
		}

		if _, err := readJSONBool(jsontext.NewDecoder(strings.NewReader(""))); err == nil {
			t.Fatal("expected error on empty input")
		}

		if _, err := readJSONBool(jsontext.NewDecoder(strings.NewReader("123"))); err == nil {
			t.Fatal("expected error on non-bool token")
		}
	})

	t.Run("readJSONUint64", func(t *testing.T) {
		if _, err := readJSONUint64(jsontext.NewDecoder(strings.NewReader("")), 10, "overflow"); err == nil {
			t.Fatal("expected error on empty input")
		}
	})
}

func TestComponentJSONMalformedStreams(t *testing.T) {
	t.Parallel()

	t.Run("componentSlice", func(t *testing.T) {
		var slice componentSlice[ActionRowChildComponent]
		if err := slice.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(""))); err == nil {
			t.Fatal("expected error on empty slice input")
		}

		if err := slice.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(`[{"type":2`))); err == nil {
			t.Fatal("expected error on truncated element")
		}
	})

	t.Run("LabelComponent", func(t *testing.T) {
		for _, bad := range []string{
			`{"component`,
			`{"component":null`,
			`{"component":{`,
			`{"description":123}`,
		} {
			var lbl LabelComponent
			if err := lbl.UnmarshalJSON([]byte(bad)); err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		}
	})

	t.Run("ActionRowComponent", func(t *testing.T) {
		var row ActionRowComponent
		if err := row.UnmarshalJSON([]byte(`{"components`)); err == nil {
			t.Fatal("expected error on truncated action row key")
		}
	})

	t.Run("ContainerComponent", func(t *testing.T) {
		var c ContainerComponent
		if err := c.UnmarshalJSON([]byte(`{"type":17,"spoiler":false}`)); err != nil || c.Spoiler {
			t.Fatalf("got (%+v, %v), want Spoiler=false", c, err)
		}

		if err := c.UnmarshalJSON([]byte(`{"id`)); err == nil {
			t.Fatal("expected error on truncated container key")
		}
	})

	t.Run("SectionComponent", func(t *testing.T) {
		var sec SectionComponent
		if err := sec.UnmarshalJSON([]byte(`{"type":9,"components":null,"accessory":null}`)); err != nil {
			t.Fatalf("unexpected error for null fields: %v", err)
		}

		for _, bad := range []string{
			`{"id`,
			`{"components":123}`,
			`{"components":[`,
			`{"components":[{`,
			`{"components":[{}]}`,
			`{"components":[{"type":10,"content":123}]}`,
			`{"accessory":null`,
			`{"accessory":{`,
			`{"id":true}`,
		} {
			var s SectionComponent
			if err := s.UnmarshalJSON([]byte(bad)); err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		}
	})
}

func TestInteractionMessageNonceSnowflakeEdgeBranches(t *testing.T) {
	t.Parallel()

	t.Run("interaction resolved and modal edge paths", func(t *testing.T) {
		resolved := &InteractionDataResolved{
			Members:     map[UserID]Member{},
			Attachments: map[AttachmentID]Attachment{{Snowflake: 1}: {FileName: "other.png"}},
		}

		if m := resolved.ResolveMember(UserID{Snowflake: 999}); m.Nickname != "" || m.User != nil {
			t.Fatalf("expected empty member, got %+v", m)
		}

		modal := ModalInteraction{
			Data: ModalInteractionData{
				Resolved: resolved,
				Components: []ModalComponent{
					LabelComponent{
						Type:  LabelComponentType,
						Label: "Upload",
						Component: FileUploadComponent{
							Type:     FileUploadComponentType,
							CustomID: "up",
							Values:   []AttachmentID{{Snowflake: 42}},
						},
					},
				},
			},
		}

		if atts := modal.GetFileUploadAttachments("up"); atts != nil {
			t.Fatalf("expected nil attachments when resolved IDs do not match, got %+v", atts)
		}

		var msd ModalInteractionData
		if err := msd.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(`{"resolved":null`))); err == nil {
			t.Fatal("expected error on truncated modal after null resolved")
		}

		if err := msd.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(`{"custom_id":123}`))); err == nil {
			t.Fatal("expected error on non-string custom_id")
		}
	})

	t.Run("message unmarshal bad scalar", func(t *testing.T) {
		var msg Message
		if err := msg.UnmarshalJSON([]byte(`{"tts":"nope"}`)); err == nil {
			t.Fatal("expected error on non-bool tts")
		}
	})

	t.Run("nonce decoder and escaped branches", func(t *testing.T) {
		var n Nonce
		if err := n.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(`"unclosed`))); err == nil {
			t.Fatal("expected error on truncated string in UnmarshalJSONFrom")
		}

		if err := n.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(`bad`))); err == nil {
			t.Fatal("expected error on invalid token in UnmarshalJSONFrom")
		}

		if err := n.UnmarshalJSON([]byte(`"abc\ndef"`)); err != nil || n != "abc\ndef" {
			t.Fatalf("escaped nonce = (%q, %v), want abc\\ndef", n, err)
		}

		if isSimpleUnescapedJSONString([]byte{0xff}) {
			t.Fatal("expected invalid UTF-8 to fail simple string check")
		}
	})

	t.Run("snowflake text and streaming decoder", func(t *testing.T) {
		var s Snowflake
		if err := s.UnmarshalText([]byte("1234567890")); err != nil || s != 1234567890 {
			t.Fatalf("UnmarshalText = (%d, %v), want 1234567890", s, err)
		}

		if err := s.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(`"unclosed`))); err == nil {
			t.Fatal("expected error on truncated snowflake string")
		}

		var holder struct {
			QuotedEmpty Snowflake `json:"quoted_empty"`
			Unquoted    Snowflake `json:"unquoted"`
		}
		payload := `{"quoted_empty":"","unquoted":987654321}`

		if err := json.Unmarshal([]byte(payload), &holder); err != nil {
			t.Fatalf("unmarshal holder: %v", err)
		}

		if holder.QuotedEmpty != 0 || holder.Unquoted != 987654321 {
			t.Fatalf("unexpected holder: %+v", holder)
		}

		for _, bad := range []string{`"notdigits"`, `"12\u0033"`, `notdigits`, `true`, `-1`} {
			var sf Snowflake
			if err := sf.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(bad))); err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		}
	})
}
