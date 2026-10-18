package tempest

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/amatsagu/tempest/discord"
)

// appIDFromToken extracts the ID from a token's first segment (base64url of the ID).
func appIDFromToken(token string) (discord.AppID, bool) {
	dot := strings.IndexByte(token, '.')
	if dot <= 0 {
		return discord.AppID{}, false
	}

	seg := token[:dot]

	raw, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		padded := seg + strings.Repeat("=", (4-len(seg)%4)%4)
		if raw, err = base64.StdEncoding.DecodeString(padded); err != nil {
			return discord.AppID{}, false
		}
	}

	id, err := strconv.ParseUint(string(raw), 10, 64)
	if err != nil || id == 0 {
		return discord.AppID{}, false
	}

	return discord.AppID{Snowflake: discord.Snowflake(id)}, true
}
