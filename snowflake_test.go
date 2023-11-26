package tempest

import (
	"testing"
)

func TestUserSnowflake(t *testing.T) {
	const RawSnowflake = "327690719085068289"

	t.Run("success", func(t *testing.T) {
		s, err := StringToSnowflake(RawSnowflake)
		if err != nil {
			t.Error(err)
		}

		if s.CreationTimestamp().UnixMilli() != 1498197955629 {
			t.Errorf("failed to read creation timestamp from %s snowflake", s.String())
		}
	})

	t.Run("failure/creation", func(t *testing.T) {
		_, err := StringToSnowflake(RawSnowflake + "a")
		if err == nil {
			t.Error(err)
		}
	})
}

func TestChannelSnowflake(t *testing.T) {
	const RawSnowflake = "1055582516565782599"

	t.Run("success", func(t *testing.T) {
		s, err := StringToSnowflake(RawSnowflake)
		if err != nil {
			t.Error(err)
		}

		if s.CreationTimestamp().UnixMilli() != 1671740883724 {
			t.Errorf("failed to read creation timestamp from %s snowflake", s.String())
		}
	})

	t.Run("failure/creation", func(t *testing.T) {
		_, err := StringToSnowflake(RawSnowflake + "a")
		if err == nil {
			t.Error(err)
		}
	})
}

func TestGuildSnowflake(t *testing.T) {
	const RawSnowflake = "613425648685547541"

	t.Run("success", func(t *testing.T) {
		s, err := StringToSnowflake(RawSnowflake)
		if err != nil {
			t.Error(err)
		}

		if s.CreationTimestamp().UnixMilli() != 1566322471544 {
			t.Errorf("failed to read creation timestamp from %s snowflake", s.String())
		}
	})

	t.Run("failure/creation", func(t *testing.T) {
		_, err := StringToSnowflake(RawSnowflake + "a")
		if err == nil {
			t.Error(err)
		}
	})
}
