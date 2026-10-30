package tempest

import (
	json "encoding/json/v2"
	"testing"
)

func TestSnowflakeJSONRoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(Snowflake(123456789012345678))
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	if got, want := string(encoded), `"123456789012345678"`; got != want {
		t.Fatalf("Marshal = %s, want %s", got, want)
	}

	var decoded Snowflake
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded != 123456789012345678 {
		t.Fatalf("Unmarshal = %d, want 123456789012345678", decoded)
	}
}

func TestSnowflakeUnmarshalRejectsEmptyInput(t *testing.T) {
	t.Parallel()

	var decoded Snowflake
	if err := decoded.UnmarshalJSON(nil); err == nil {
		t.Fatal("UnmarshalJSON(nil) returned nil error")
	}

	if err := decoded.UnmarshalText(nil); err != nil {
		// Empty text clears snowflake to 0
		t.Fatalf("UnmarshalText(nil) returned error: %v", err)
	}

	if decoded != 0 {
		t.Fatalf("expected 0, got %d", decoded)
	}
}

func TestSnowflakeAppendText(t *testing.T) {
	t.Parallel()

	s := Snowflake(123456789012345678)
	buf := make([]byte, 0, 32)
	out, err := s.AppendText(buf)
	if err != nil {
		t.Fatalf("AppendText error: %v", err)
	}

	if string(out) != "123456789012345678" {
		t.Fatalf("AppendText = %s, want 123456789012345678", string(out))
	}
}
