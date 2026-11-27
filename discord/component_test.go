package discord_test

import (
	json "encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func TestComponentRequiredSerialization(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name: "SelectComponent with Required false",
			input: discord.SelectComponent{
				Type:     discord.RoleSelectComponentType,
				Required: new(false),
			},
			expected: `"required":false`,
		},
		{
			name: "StringSelectComponent with Required false",
			input: discord.StringSelectComponent{
				Type:     discord.StringSelectComponentType,
				Required: new(false),
			},
			expected: `"required":false`,
		},
		{
			name: "TextInputComponent with Required false",
			input: discord.TextInputComponent{
				Type:     discord.TextInputComponentType,
				Required: new(false),
			},
			expected: `"required":false`,
		},
		{
			name: "SeparatorComponent with Divider false",
			input: discord.SeparatorComponent{
				Type:    discord.SeparatorComponentType,
				Divider: new(false),
			},
			expected: `"divider":false`,
		},
		{
			name: "CommandOption with MinValue 0.0",
			input: discord.CommandOption{
				Type:     discord.NumberOptionType,
				Name:     "amount",
				MinValue: new(0.0),
			},
			expected: `"min_value":0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			if !strings.Contains(string(raw), tt.expected) {
				t.Errorf("expected JSON to contain %q, but got: %s", tt.expected, string(raw))
			}
		})
	}

	// Omitting default-true *bool fields must omit the JSON key (safe in messages, uses Discord true default in modals).
	for _, tc := range []struct {
		input any
		name  string
		key   string
	}{
		{name: "SelectComponent message omits required", input: discord.SelectComponent{Type: discord.UserSelectComponentType}, key: `"required"`},
		{name: "StringSelectComponent message omits required", input: discord.StringSelectComponent{Type: discord.StringSelectComponentType}, key: `"required"`},
		{name: "TextInputComponent omits required when nil", input: discord.TextInputComponent{Type: discord.TextInputComponentType}, key: `"required"`},
		{name: "FileUploadComponent omits required when nil", input: discord.FileUploadComponent{Type: discord.FileUploadComponentType}, key: `"required"`},
		{name: "RadioGroupComponent omits required when nil", input: discord.RadioGroupComponent{Type: discord.RadioGroupComponentType}, key: `"required"`},
		{name: "CheckboxGroupComponent omits required when nil", input: discord.CheckboxGroupComponent{Type: discord.CheckboxGroupComponentType}, key: `"required"`},
		{name: "SeparatorComponent omits divider when nil", input: discord.SeparatorComponent{Type: discord.SeparatorComponentType}, key: `"divider"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			if strings.Contains(string(raw), tc.key) {
				t.Errorf("expected JSON to omit %s, got: %s", tc.key, string(raw))
			}
		})
	}
}
