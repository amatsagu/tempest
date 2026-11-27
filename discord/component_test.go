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
				Required: false,
			},
			expected: `"required":false`,
		},
		{
			name: "StringSelectComponent with Required false",
			input: discord.StringSelectComponent{
				Type:     discord.StringSelectComponentType,
				Required: false,
			},
			expected: `"required":false`,
		},
		{
			name: "TextInputComponent with Required false",
			input: discord.TextInputComponent{
				Type:     discord.TextInputComponentType,
				Required: false,
			},
			expected: `"required":false`,
		},
		{
			name: "SeparatorComponent with Divider false",
			input: discord.SeparatorComponent{
				Type:    discord.SeparatorComponentType,
				Divider: false,
			},
			expected: `"divider":false`,
		},
		{
			name: "CommandOption with MinValue 0.0",
			input: discord.CommandOption{
				Type:     discord.NumberOptionType,
				Name:     "amount",
				MinValue: func() *float64 { v := 0.0; return &v }(),
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
}
