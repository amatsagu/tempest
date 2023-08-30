package tempest

import (
	"strconv"
	"time"

	fjson "github.com/goccy/go-json"
)

// Snowflake represents a Discord's id snowflake.
type Snowflake uint64

func StringToSnowflake(s string) (Snowflake, error) {
	i, err := strconv.ParseUint(s, 10, 64)
	return Snowflake(i), err
}

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s Snowflake) CreationTimestamp() time.Time {
	return time.UnixMilli(int64(s>>22 + EPOCH))
}

func (s Snowflake) MarshalJSON() ([]byte, error) {
	b := strconv.FormatUint(uint64(s), 10)
	return fjson.Marshal(b)
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
