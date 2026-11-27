package discord

import (
	json "encoding/json/v2"
	"errors"
	"math"
	"testing"
	"time"
)

func TestStringToSnowflake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Snowflake
		wantErr bool
	}{
		{
			name:    "valid snowflake",
			input:   "1159846879854792704",
			want:    1159846879854792704,
			wantErr: false,
		},
		{
			name:    "zero value",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "max uint64",
			input:   "18446744073709551615",
			want:    math.MaxUint64,
			wantErr: false,
		},
		{
			name:    "invalid characters",
			input:   "123abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "overflow",
			input:   "18446744073709551616",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToSnowflake(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("StringToSnowflake() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("StringToSnowflake() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnowflakeString(t *testing.T) {
	t.Parallel()

	s := Snowflake(1159846879854792704)
	if got := s.String(); got != "1159846879854792704" {
		t.Errorf("String() = %v, want 1159846879854792704", got)
	}
}

func TestSnowflakeCreationTimestamp(t *testing.T) {
	t.Parallel()

	s := Snowflake(175928847299117063)
	got := s.CreationTimestamp().UTC()
	expected := time.Date(2016, time.April, 30, 11, 18, 25, 796*1000000, time.UTC)

	if !got.Equal(expected) {
		t.Errorf("CreationTimestamp() = %v, want %v", got, expected)
	}
}

func TestSnowflakeJSONUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  Snowflake
		expectErr bool
	}{
		{
			name:      "quoted valid string",
			input:     `"1159846879854792704"`,
			expected:  1159846879854792704,
			expectErr: false,
		},
		{
			name:      "unquoted valid number",
			input:     `1159846879854792704`,
			expected:  1159846879854792704,
			expectErr: false,
		},
		{
			name:      "null value",
			input:     `null`,
			expected:  0,
			expectErr: false,
		},
		{
			name:      "empty string",
			input:     `""`,
			expected:  0,
			expectErr: false,
		},
		{
			name:      "whitespace padded quotes",
			input:     " \t\n \"1159846879854792704\" \r\n",
			expected:  1159846879854792704,
			expectErr: false,
		},
		{
			name:      "empty byte slice",
			input:     ``,
			expected:  0,
			expectErr: false,
		},
		{
			name:      "invalid non-digit chars",
			input:     `"987xyz123"`,
			expected:  0,
			expectErr: true,
		},
		{
			name:      "overflow value",
			input:     `"18446744073709551616"`,
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Snowflake
			err := s.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.expectErr {
				t.Fatalf("UnmarshalJSON() error = %v, expectErr %v", err, tt.expectErr)
			}

			if tt.expectErr && !errors.Is(err, ErrInvalidSnowflake) {
				t.Errorf("expected ErrInvalidSnowflake, got: %v", err)
			}

			if !tt.expectErr && s != tt.expected {
				t.Errorf("UnmarshalJSON() got = %v, want %v", s, tt.expected)
			}
		})
	}
}

func TestSnowflakeJSONMarshal(t *testing.T) {
	t.Parallel()

	s := Snowflake(1159846879854792704)

	want := `1159846879854792704`
	got, err := s.AppendText(nil)
	if err != nil {
		t.Fatalf("AppendText() error: %v", err)
	}

	if string(got) != want {
		t.Errorf("AppendText() got = %s, want %s", string(got), want)
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	if string(data) != `"`+want+`"` {
		t.Errorf("json.Marshal() got = %s, want %q", string(data), want)
	}
}

func TestSnowflakeTextAndStructRoundTrip(t *testing.T) {
	t.Parallel()

	var decoded Snowflake
	if err := decoded.UnmarshalText(nil); err != nil {
		t.Fatalf("UnmarshalText(nil) returned error: %v", err)
	}

	if decoded != 0 {
		t.Fatalf("expected 0, got %d", decoded)
	}

	if err := decoded.UnmarshalText([]byte("abc")); !errors.Is(err, ErrInvalidSnowflake) {
		t.Fatalf("UnmarshalText(abc) = %v, want ErrInvalidSnowflake", err)
	}

	type DiscordPayload struct {
		ChannelID *ChannelID `json:"channel_id"`
		OmittedID GuildID    `json:"omitted_id,omitzero"`
		ID        UserID     `json:"id"`
	}

	channelID := ChannelID{123456789}
	payload := DiscordPayload{
		ID:        UserID{1159846879854792704},
		ChannelID: &channelID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	var encoded map[string]string
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatalf("json.Unmarshal(marshaled) failed: %v", err)
	}

	if encoded["id"] != "1159846879854792704" || encoded["channel_id"] != "123456789" {
		t.Errorf("Marshal output mismatch: got %s", data)
	}

	if _, exists := encoded["omitted_id"]; exists {
		t.Errorf("expected zero GuildID with omitzero to be omitted, got %s", data)
	}

	var parsed DiscordPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(parsed) failed: %v", err)
	}

	if parsed.ID != payload.ID || parsed.ChannelID == nil || *parsed.ChannelID != channelID {
		t.Errorf("RoundTrip mismatch: got %+v, want %+v", parsed, payload)
	}
}

func TestParseUint64Bytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   uint64
		wantOk bool
	}{
		{"empty", "", 0, false},
		{"single digit zero", "0", 0, true},
		{"typical discord id", "1159846879854792704", 1159846879854792704, true},
		{"max uint64 exact", "18446744073709551615", math.MaxUint64, true},
		{"max uint64 + 1 (overflow)", "18446744073709551616", 0, false},
		{"21 digits (overflow)", "184467440737095516150", 0, false},
		{"non digit start", "a123", 0, false},
		{"non digit middle", "123x456", 0, false},
		{"non digit end", "123456!", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseUint64Bytes([]byte(tt.input))
			if ok != tt.wantOk {
				t.Fatalf("parseUint64Bytes(%q) ok = %v, wantOk %v", tt.input, ok, tt.wantOk)
			}

			if got != tt.want {
				t.Errorf("parseUint64Bytes(%q) got = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStrictSnowflakesAndMentions(t *testing.T) {
	t.Parallel()

	msgID := MessageID{1159846879854792704}
	attID := AttachmentID{1159846879854792704}
	appID := AppID{1159846879854792704}
	intID := IntegrationID{1159846879854792704}
	skuID := SkuID{1159846879854792704}

	if msgID.String() != "1159846879854792704" || attID.String() != "1159846879854792704" ||
		appID.String() != "1159846879854792704" || intID.String() != "1159846879854792704" ||
		skuID.String() != "1159846879854792704" {
		t.Fatal("strict snowflake String() mismatch")
	}

	if got := (UserID{42}).Mention(); got != "<@42>" {
		t.Fatalf("UserID.Mention() = %q, want <@42>", got)
	}

	if got := (UserID{}).Mention(); got != "" {
		t.Fatalf("zero UserID.Mention() = %q, want empty", got)
	}

	if got := (ChannelID{42}).Mention(); got != "<#42>" {
		t.Fatalf("ChannelID.Mention() = %q, want <#42>", got)
	}

	if got := (RoleID{42}).Mention(); got != "<@&42>" {
		t.Fatalf("RoleID.Mention() = %q, want <@&42>", got)
	}

	if got := (EmojiID{42}).Mention("wave", true); got != "<a:wave:42>" {
		t.Fatalf("EmojiID.Mention() = %q, want <a:wave:42>", got)
	}

	if got := (CommandID{42}).Mention("ping"); got != "</ping:42>" {
		t.Fatalf("CommandID.Mention() = %q, want </ping:42>", got)
	}
}
