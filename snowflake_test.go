package tempest

import (
	"encoding/json"
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
}
