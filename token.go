package tempest

import (
	"encoding/base64"
	"errors"
	"strings"
)

func extractUserIDFromToken(token string) (Snowflake, error) {
	strs := strings.Split(token, ".")
	if len(strs) == 0 {
		return 0, errors.New("token is not in a valid format")
	}

	hexID := strings.Replace(strs[0], "Bot ", "", 1)

	byteID, err := base64.RawStdEncoding.DecodeString(hexID)
	if err != nil {
		return 0, err
	}

	return StringToSnowflake(string(byteID))
}
