package discord

import (
	"encoding/json/v2"
	"errors"
	"math"
	"testing"
	"time"
)

func TestStringToSnowflake(t *testing.T) {
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

func TestSnowflake_String(t *testing.T) {
	s := Snowflake(1159846879854792704)
	if got := s.String(); got != "1159846879854792704" {
		t.Errorf("String() = %v, want 1159846879854792704", got)
	}
}

func TestSnowflake_CreationTimestamp(t *testing.T) {
	s := Snowflake(175928847299117063)
	got := s.CreationTimestamp().UTC()
	expected := time.Date(2016, time.April, 30, 11, 18, 25, 796*1000000, time.UTC)

	if !got.Equal(expected) {
		t.Errorf("CreationTimestamp() = %v, want %v", got, expected)
	}
}

func TestSnowflake_JSON_Unmarshal(t *testing.T) {
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

func TestSnowflake_JSON_Marshal(t *testing.T) {
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

func TestSnowflake_JSON_Struct_RoundTrip(t *testing.T) {
	type DiscordPayload struct {
		ChannelID *Snowflake `json:"channel_id"`
		ID        Snowflake  `json:"id"`
	}

	channelID := Snowflake(123456789)
	payload := DiscordPayload{
		ID:        Snowflake(1159846879854792704),
		ChannelID: &channelID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	// Field order in the encoded object is intentionally not asserted: it can
	// change with struct field alignment, which Discord does not care about.
	var encoded map[string]string
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatalf("json.Unmarshal(marshaled) failed: %v", err)
	}
	if encoded["id"] != "1159846879854792704" || encoded["channel_id"] != "123456789" {
		t.Errorf("Marshal output mismatch: got %s", data)
	}

	var parsed DiscordPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	if parsed.ID != payload.ID {
		t.Errorf("ID got = %v, want %v", parsed.ID, payload.ID)
	}

	if parsed.ChannelID == nil || *parsed.ChannelID != *payload.ChannelID {
		t.Errorf("ChannelID got = %v, want %v", parsed.ChannelID, payload.ChannelID)
	}
}

func TestParseUint64Bytes(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   uint64
		wantOk bool
	}{
		{"empty", "", 0, false},
		{"single digit zero", "0", 0, true},
		{"typical discord id", "1159846879854792704", 1159846879854792704, true},
		{"max uint64 exact", "18446744073709551615", 18446744073709551615, true},
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

func TestStrictSnowflakes(t *testing.T) {
	msgID := MessageID{Snowflake: 1159846879854792704}
	attID := AttachmentID{Snowflake: 1159846879854792704}
	appID := AppID{Snowflake: 1159846879854792704}
	intID := IntegrationID{Snowflake: 1159846879854792704}
	skuID := SkuID{Snowflake: 1159846879854792704}

	if msgID.String() != "1159846879854792704" {
		t.Errorf("msgID.String() = %s, want 1159846879854792704", msgID.String())
	}
	if attID.String() != "1159846879854792704" {
		t.Errorf("attID.String() = %s, want 1159846879854792704", attID.String())
	}
	if appID.String() != "1159846879854792704" {
		t.Errorf("appID.String() = %s, want 1159846879854792704", appID.String())
	}
	if intID.String() != "1159846879854792704" {
		t.Errorf("intID.String() = %s, want 1159846879854792704", intID.String())
	}
	if skuID.String() != "1159846879854792704" {
		t.Errorf("skuID.String() = %s, want 1159846879854792704", skuID.String())
	}
}

func BenchmarkSnowflake_UnmarshalJSON(b *testing.B) {
	payload := []byte(`"1159846879854792704"`)
	b.ReportAllocs()
	for b.Loop() {
		var s Snowflake
		if err := s.UnmarshalJSON(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSnowflake_AppendText(b *testing.B) {
	s := Snowflake(1159846879854792704)
	buf := make([]byte, 0, 20)
	b.ReportAllocs()
	for b.Loop() {
		buf, _ = s.AppendText(buf[:0])
	}
}
