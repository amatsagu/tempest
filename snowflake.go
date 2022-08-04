package tempest

import (
	"encoding/json"
	"strconv"
)

// Snowflake represents a Discord's id snowflake.
type Snowflake uint64

func StringToSnowflake(s string) Snowflake {
	i, _ := strconv.ParseUint(s, 10, 64)
	return Snowflake(i)
}

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
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
