package tempest

import (
	"encoding/json"
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

func (s Snowflake) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatUint(uint64(s), 10))
}

func (s *Snowflake) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && b[0] == 'n' { // "null"
		*s = 0
		return nil
	}

	// Trim quotes without allocation
	if b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}

	i, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(i)
	return nil
}
