package discord

import (
	"encoding/json/v2"
	"strings"
	"testing"
)

// These tests pin down json/v2 "omitzero" semantics: a nil slice/map is omitted,
// a non-nil empty one is emitted as []/{}. On an edit that difference matters.

func mustMarshal(t *testing.T, v any) string {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return string(raw)
}

func assertHas(t *testing.T, payload, want string) {
	t.Helper()

	if !strings.Contains(payload, want) {
		t.Errorf("expected payload to contain %s, got %s", want, payload)
	}
}

func assertMissingKey(t *testing.T, payload, key string) {
	t.Helper()

	if strings.Contains(payload, `"`+key+`"`) {
		t.Errorf("expected payload to omit %q, got %s", key, payload)
	}
}

func TestResponseMessageOmitsNilCollections(t *testing.T) {
	payload := mustMarshal(t, ResponseMessageData{})

	for _, key := range []string{"embeds", "components", "attachments", "sticker_ids"} {
		assertMissingKey(t, payload, key)
	}
}

func TestResponseMessageEmitsEmptyCollections(t *testing.T) {
	payload := mustMarshal(t, ResponseMessageData{
		Embeds:      []Embed{},
		Components:  Components{},
		Attachments: []Attachment{},
		StickerIDs:  []Snowflake{},
	})

	assertHas(t, payload, `"embeds":[]`)
	assertHas(t, payload, `"components":[]`)
	assertHas(t, payload, `"attachments":[]`)
	assertHas(t, payload, `"sticker_ids":[]`)
}

func TestEditResponseClearsEmbeds(t *testing.T) {
	original := Response{
		Type: ResponseChannelMessageWithSource,
		Data: ResponseMessageData{
			Content: "hello",
			Embeds:  []Embed{{Title: "first"}},
		},
	}

	originalPayload := mustMarshal(t, original)
	assertHas(t, originalPayload, `"embeds":[`)

	// An explicit empty array is how an edit removes the embeds.
	clear := Response{
		Type: ResponseUpdateMessage,
		Data: ResponseMessageData{Embeds: []Embed{}},
	}

	clearPayload := mustMarshal(t, clear)
	assertHas(t, clearPayload, `"embeds":[]`)

	// Omitting them does not clear anything on Discord; guard against regression.
	omit := Response{
		Type: ResponseUpdateMessage,
		Data: ResponseMessageData{},
	}

	omitPayload := mustMarshal(t, omit)
	assertMissingKey(t, omitPayload, "embeds")
}

func TestEditResponseClearsComponentsAttachmentsStickers(t *testing.T) {
	edit := Response{
		Type: ResponseUpdateMessage,
		Data: ResponseMessageData{
			Components:  Components{},
			Attachments: []Attachment{},
			StickerIDs:  []Snowflake{},
		},
	}

	payload := mustMarshal(t, edit)
	assertHas(t, payload, `"components":[]`)
	assertHas(t, payload, `"attachments":[]`)
	assertHas(t, payload, `"sticker_ids":[]`)
}

func TestEmptyEmbedFieldsEmitted(t *testing.T) {
	payload := mustMarshal(t, Embed{Fields: []EmbedField{}})
	assertHas(t, payload, `"fields":[]`)

	omitted := mustMarshal(t, Embed{})
	assertMissingKey(t, omitted, "fields")
}

func TestAllowedMentionsParseNone(t *testing.T) {
	// An empty parse list is how you disable all mention parsing.
	payload := mustMarshal(t, AllowedMentions{Parse: []AllowedMentionType{}})
	assertHas(t, payload, `"parse":[]`)

	omitted := mustMarshal(t, AllowedMentions{})
	assertMissingKey(t, omitted, "parse")
}

func TestEmptyMapEmitsObject(t *testing.T) {
	payload := mustMarshal(t, Command{NameLocalizations: map[Locale]string{}})
	assertHas(t, payload, `"name_localizations":{}`)

	omitted := mustMarshal(t, Command{})
	assertMissingKey(t, omitted, "name_localizations")
}

func TestResolvedEmptyMapEmitsObject(t *testing.T) {
	payload := mustMarshal(t, InteractionDataResolved{Users: map[UserID]User{}})
	assertHas(t, payload, `"users":{}`)

	omitted := mustMarshal(t, InteractionDataResolved{})
	assertMissingKey(t, omitted, "users")
}

func TestPointerToZeroIsEmitted(t *testing.T) {
	// Nullable numeric fields use pointers so 0 is serialized, not treated as unset.
	zero := uint8(0)
	payload := mustMarshal(t, StringSelect{MinValues: &zero})
	assertHas(t, payload, `"min_values":0`)
}

func TestRequiredSliceWithoutOmitzeroIsAlwaysArray(t *testing.T) {
	// Fields like StringSelect.Options have no omitzero, so nil encodes as [].
	payload := mustMarshal(t, StringSelect{})
	assertHas(t, payload, `"options":[]`)
}

func TestMessageEmptyCollections(t *testing.T) {
	payload := mustMarshal(t, Message{
		Components:   Components{},
		StickerItems: []StickerItem{},
		Reactions:    []Reaction{},
	})

	assertHas(t, payload, `"components":[]`)
	assertHas(t, payload, `"sticker_items":[]`)
	assertHas(t, payload, `"reactions":[]`)
}
