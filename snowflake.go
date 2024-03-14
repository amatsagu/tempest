package tempest

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

// Snowflake represents a Discord's id snowflake.
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
	b := strconv.FormatUint(uint64(s), 10)
	return json.Marshal(b)
}

func (s *Snowflake) UnmarshalJSON(b []byte) error {
	str, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	i, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(i)
	return nil
}
