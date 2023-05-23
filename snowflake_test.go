package tempest

import (
	"testing"
)

// Tried to encode & decode few example snowflakes
func TestSnowflake(t *testing.T) {
	const userRawSnowflake = "327690719085068289"
	const channelRawSnowflake = "1055582516565782599"
	const guildRawSnowflake = "613425648685547541"

	s, err := StringToSnowflake(userRawSnowflake)
	if err != nil {
		t.Error(err)
	}

	if s.CreationTimestamp().UnixMilli() != 1498197955629 {
		t.Errorf("failed to read creation timestamp from %s snowflake", s.String())
	}

	s, err = StringToSnowflake(channelRawSnowflake)
	if err != nil {
		t.Error(err)
	}

	if s.CreationTimestamp().UnixMilli() != 1671740883724 {
		t.Errorf("failed to read creation timestamp from %s snowflake", s.String())
	}

	s, err = StringToSnowflake(guildRawSnowflake)
	if err != nil {
		t.Error(err)
	}

	if s.CreationTimestamp().UnixMilli() != 1566322471544 {
		t.Errorf("failed to read creation timestamp from %s snowflake", s.String())
	}
}
