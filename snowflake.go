package tempest

import (
	"bytes"
	"errors"
	"os"
	"strconv"
	"time"
)

// Represents a Discord's ID snowflake.
type Snowflake uint64

func StringToSnowflake(s string) (Snowflake, error) {
	i, err := strconv.ParseUint(s, 10, 64)
	return Snowflake(i), err
}

// Shortcut to calling os.Getenv method and casting to Snowflake.
func EnvToSnowflake(key string) (Snowflake, error) {
	return StringToSnowflake(os.Getenv(key))
}

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s Snowflake) CreationTimestamp() time.Time {
	return time.UnixMilli(int64(s>>22 + DISCORD_EPOCH))
}

func (s Snowflake) AppendText(b []byte) ([]byte, error) {
	return strconv.AppendUint(b, uint64(s), 10), nil
}

func (s *Snowflake) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		*s = 0
		return nil
	}

	val, ok := parseUint64Bytes(b)
	if !ok {
		return errors.New("cannot unmarshal invalid snowflake")
	}

	*s = Snowflake(val)
	return nil
}

func (s *Snowflake) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return errors.New("cannot unmarshal empty snowflake")
	}

	if len(b) == 4 && b[0] == 'n' && b[1] == 'u' && b[2] == 'l' && b[3] == 'l' {
		*s = 0
		return nil
	}

	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}

	if len(b) == 0 {
		*s = 0
		return nil
	}

	val, ok := parseUint64Bytes(b)
	if !ok {
		return errors.New("cannot unmarshal invalid snowflake")
	}

	*s = Snowflake(val)
	return nil
}

func parseUint64Bytes(b []byte) (uint64, bool) {
	if len(b) == 0 {
		return 0, false
	}

	var n uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}

		d := uint64(c - '0')
		if n > (18446744073709551615-d)/10 {
			return 0, false
		}
		n = n*10 + d
	}

	return n, true
}
