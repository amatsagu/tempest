package tempest

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

func extractUserIDFromToken(token string) (Snowflake, error) {
	strs := strings.Split(token, ".")
	if len(strs) == 0 {
		return 0, errors.New("token is not in a valid format")
	}

	byteID, err := base64.RawStdEncoding.DecodeString(strs[0])
	if err != nil {
		return 0, err
	}

	ID, err := strconv.ParseUint(string(byteID), 10, 64)
	return Snowflake(ID), err
}
