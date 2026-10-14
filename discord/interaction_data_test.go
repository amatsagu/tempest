package discord

import (
	"encoding/json/v2"
	"testing"
)

func TestCommandInteractionData_Helpers(t *testing.T) {
	data := CommandInteractionData{
		ID:   CommandID{Snowflake: 1234},
		Name: "ban",
		Type: CommandChatInput,
		Options: []CommandInteractionOption{
			{Name: "user", Type: OptionUser, Value: "1159846879854792704"},
			{Name: "reason", Type: OptionString, Value: "spamming"},
			{Name: "days", Type: OptionInteger, Value: float64(7)},
			{Name: "silent", Type: OptionBoolean, Value: true},
		},
	}

	user, ok := data.UserOption("user")
	if !ok || user.String() != "1159846879854792704" {
		t.Errorf("UserOption failed: %v, %v", user, ok)
	}

	reason, ok := data.StringOption("reason")
	if !ok || reason != "spamming" {
		t.Errorf("StringOption failed: %v, %v", reason, ok)
	}

	days, ok := data.IntOption("days")
	if !ok || days != 7 {
		t.Errorf("IntOption failed: %v, %v", days, ok)
	}

	silent, ok := data.BoolOption("silent")
	if !ok || !silent {
		t.Errorf("BoolOption failed: %v, %v", silent, ok)
	}

	_, missing := data.Option("not_found")
	if missing {
		t.Errorf("expected not_found to be missing")
	}
}

func TestModalInteractionData_Helpers(t *testing.T) {
	raw := []byte(`{
		"custom_id": "profile_modal",
		"components": [
			{
				"type": 18,
				"id": 1,
				"component": {
					"type": 4,
					"id": 2,
					"custom_id": "user_bio",
					"value": "Hello Gophers"
				}
			},
			{
				"type": 18,
				"id": 3,
				"component": {
					"type": 23,
					"id": 4,
					"custom_id": "agree_terms",
					"value": true
				}
			}
		]
	}`)

	var modal ModalInteractionData
	if err := json.Unmarshal(raw, &modal); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	bio, ok := modal.TextInput("user_bio")
	if !ok || bio != "Hello Gophers" {
		t.Errorf("modal.TextInput failed: %s, %v", bio, ok)
	}

	agreed := modal.Checkbox("agree_terms")
	if !agreed {
		t.Errorf("modal.Checkbox failed: got false, want true")
	}

	notAgreed := modal.Checkbox("random_box")
	if notAgreed {
		t.Errorf("expected random_box to return false")
	}
}

func BenchmarkModalInteractionData_Unmarshal(b *testing.B) {
	raw := []byte(`{
		"custom_id": "profile_modal",
		"components": [
			{
				"type": 18,
				"id": 1,
				"component": {
					"type": 4,
					"id": 2,
					"custom_id": "user_bio",
					"value": "Hello Gophers"
				}
			}
		]
	}`)

	b.ReportAllocs()
	for b.Loop() {
		var modal ModalInteractionData
		_ = json.Unmarshal(raw, &modal)
	}
}
