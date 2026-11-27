package discord

import (
	json "encoding/json/v2"
	"testing"
)

func TestEnvToSnowflake(t *testing.T) {
	const key = "TEMPEST_TEST_SNOWFLAKE"

	t.Setenv(key, "")
	if _, err := EnvToSnowflake(key); err == nil {
		t.Fatal("expected error for empty env var")
	}

	t.Setenv(key, "not-a-snowflake")
	if _, err := EnvToSnowflake(key); err == nil {
		t.Fatal("expected error for invalid env var")
	}

	t.Setenv(key, "1159846879854792704")
	sf, err := EnvToSnowflake(key)
	if err != nil {
		t.Fatalf("EnvToSnowflake failed: %v", err)
	}
	if sf != 1159846879854792704 {
		t.Fatalf("got %d, want 1159846879854792704", sf)
	}
}

func TestSnowflakeBitFieldsAndErrors(t *testing.T) {
	const (
		msOffset  = uint64(1000)
		worker    = uint8(5)
		pid       = uint8(17)
		increment = uint16(42)
	)
	sf := Snowflake((msOffset << 22) | (uint64(worker) << 17) | (uint64(pid) << 12) | uint64(increment))

	if sf.Worker() != worker {
		t.Fatalf("Worker = %d, want %d", sf.Worker(), worker)
	}
	if sf.PID() != pid {
		t.Fatalf("PID = %d, want %d", sf.PID(), pid)
	}
	if sf.Increment() != increment {
		t.Fatalf("Increment = %d, want %d", sf.Increment(), increment)
	}
	if got := sf.CreationTimestamp().UnixMilli(); got != DiscordEpoch+int64(msOffset) {
		t.Fatalf("CreationTimestamp ms = %d, want %d", got, DiscordEpoch+int64(msOffset))
	}

	var decoded Snowflake
	if err := decoded.UnmarshalText([]byte("abc")); err == nil {
		t.Fatal("expected error from UnmarshalText for non-numeric input")
	}
}

func TestTypedSnowflakesAllMentionsAndRoundTrip(t *testing.T) {
	if (GuildID{}).Snowflake != 0 || (GuildID{Snowflake: 1}).Snowflake == 0 {
		t.Fatal("unexpected zero state on GuildID")
	}
	if (ChannelID{}).Mention() != "" || (RoleID{}).Mention() != "" || (CommandID{}).Mention("a") != "" || (EmojiID{}).Mention("a", true) != "" {
		t.Fatal("expected empty mention string for zero snowflake IDs")
	}

	if got := (UserID{Snowflake: 1}).Mention(); got != "<@1>" {
		t.Fatalf("UserID.Mention = %q", got)
	}
	if got := (ChannelID{Snowflake: 2}).Mention(); got != "<#2>" {
		t.Fatalf("ChannelID.Mention = %q", got)
	}
	if got := (RoleID{Snowflake: 3}).Mention(); got != "<@&3>" {
		t.Fatalf("RoleID.Mention = %q", got)
	}
	if got := (CommandID{Snowflake: 4}).Mention("ping"); got != "</ping:4>" {
		t.Fatalf("CommandID.Mention = %q", got)
	}
	if got := (EmojiID{Snowflake: 5}).Mention("ok", false); got != "<:ok:5>" {
		t.Fatalf("EmojiID static Mention = %q", got)
	}
	if got := (EmojiID{Snowflake: 5}).Mention("ok", true); got != "<a:ok:5>" {
		t.Fatalf("EmojiID animated Mention = %q", got)
	}

	type allIDs struct {
		User         UserID         `json:"user"`
		Channel      ChannelID      `json:"channel"`
		Role         RoleID         `json:"role"`
		Emoji        EmojiID        `json:"emoji"`
		Guild        GuildID        `json:"guild"`
		Command      CommandID      `json:"command"`
		Message      MessageID      `json:"message"`
		Attachment   AttachmentID   `json:"attachment"`
		App          AppID          `json:"app"`
		Integration  IntegrationID  `json:"integration"`
		Sku          SkuID          `json:"sku"`
		Entitlement  EntitlementID  `json:"entitlement"`
		Subscription SubscriptionID `json:"subscription"`
	}

	in := allIDs{
		User:         UserID{Snowflake: 1},
		Channel:      ChannelID{Snowflake: 2},
		Role:         RoleID{Snowflake: 3},
		Emoji:        EmojiID{Snowflake: 4},
		Guild:        GuildID{Snowflake: 5},
		Command:      CommandID{Snowflake: 6},
		Message:      MessageID{Snowflake: 7},
		Attachment:   AttachmentID{Snowflake: 8},
		App:          AppID{Snowflake: 9},
		Integration:  IntegrationID{Snowflake: 10},
		Sku:          SkuID{Snowflake: 11},
		Entitlement:  EntitlementID{Snowflake: 12},
		Subscription: SubscriptionID{Snowflake: 13},
	}

	raw, err := json.Marshal(&in)
	if err != nil {
		t.Fatalf("marshal allIDs failed: %v", err)
	}
	var out allIDs
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal allIDs failed: %v", err)
	}
	if out != in {
		t.Fatalf("got %+v, want %+v", out, in)
	}

	type singleSF struct {
		ID Snowflake `json:"id"`
	}
	var esc singleSF
	if err := json.Unmarshal([]byte(`{"id":"\u0031\u0032"}`), &esc); err == nil {
		t.Fatalf("expected error for escaped snowflake string, got %+v", esc)
	}
	var nullSF singleSF
	if err := json.Unmarshal([]byte(`{"id":null}`), &nullSF); err != nil || nullSF.ID != 0 {
		t.Fatalf("expected 0 for null snowflake, got %+v, err=%v", nullSF, err)
	}
}

func BenchmarkSnowflake_AppendText(b *testing.B) {
	sf := Snowflake(1159846879854792704)
	var buf [32]byte

	b.ReportAllocs()
	for b.Loop() {
		_, err := sf.AppendText(buf[:0])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSnowflake_UnmarshalJSON(b *testing.B) {
	quoted := []byte(`"1159846879854792704"`)
	numeric := []byte(`1159846879854792704`)

	b.Run("quoted", func(b *testing.B) {
		var sf Snowflake
		b.ReportAllocs()
		for b.Loop() {
			if err := sf.UnmarshalJSON(quoted); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("numeric", func(b *testing.B) {
		var sf Snowflake
		b.ReportAllocs()
		for b.Loop() {
			if err := sf.UnmarshalJSON(numeric); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSnowflake_JSONSlice(b *testing.B) {
	ids := []UserID{
		{Snowflake: 1159846879854792704},
		{Snowflake: 1159846879854792705},
		{Snowflake: 1159846879854792706},
		{Snowflake: 1159846879854792707},
		{Snowflake: 1159846879854792708},
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed []UserID
		if err := json.Unmarshal(raw, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(ids); err != nil {
			b.Fatal(err)
		}
	}
}
