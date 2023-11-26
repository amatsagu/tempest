package tempest

import (
	"strings"
	"testing"
)

func TestUserToken(t *testing.T) {
	const exampleBotToken = "MTE3ODAzNjYxNjU1NjcyODM2MA.GWWLzQ.xI3VzmU5Pj1we5IjRLeKQVjBiHCYJmAdTlfWEo"
	const exampleBotUserID Snowflake = 1178036616556728360

	t.Run("success", func(t *testing.T) {
		ID, err := extractUserIDFromToken(exampleBotToken)
		if err != nil {
			t.Error(err)
		}

		if ID != exampleBotUserID {
			t.Error("extracted ID is invalid")
		}
	})

	t.Run("failure/creation", func(t *testing.T) {
		_, err := StringToSnowflake(strings.ReplaceAll(exampleBotToken, ".", ""))
		if err == nil {
			t.Error(err)
		}
	})

	t.Run("failure/extraction", func(t *testing.T) {
		ID, err := extractUserIDFromToken(strings.ReplaceAll(exampleBotToken, "z", "x"))
		if err != nil {
			t.Error(err)
		}

		if ID == exampleBotUserID {
			t.Error("extracted ID is invalid")
		}
	})
}
