package tempest_test

import (
	json "encoding/json/v2"
	"strings"
	"testing"

	"github.com/amatsagu/tempest"
)

func TestComponentRequiredSerialization(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name: "SelectComponent with Required false",
			input: tempest.SelectComponent{
				Type:     tempest.ROLE_SELECT_COMPONENT_TYPE,
				Required: false,
			},
			expected: `"required":false`,
		},
		{
			name: "StringSelectComponent with Required false",
			input: tempest.StringSelectComponent{
				Type:     tempest.STRING_SELECT_COMPONENT_TYPE,
				Required: false,
			},
			expected: `"required":false`,
		},
		{
			name: "TextInputComponent with Required false",
			input: tempest.TextInputComponent{
				Type:     tempest.TEXT_INPUT_COMPONENT_TYPE,
				Required: false,
			},
			expected: `"required":false`,
		},
		{
			name: "SeparatorComponent with Divider false",
			input: tempest.SeparatorComponent{
				Type:    tempest.SEPARATOR_COMPONENT_TYPE,
				Divider: false,
			},
			expected: `"divider":false`,
		},
		{
			name: "CommandOption with MinValue 0.0",
			input: tempest.CommandOption{
				Type:     tempest.NUMBER_OPTION_TYPE,
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
