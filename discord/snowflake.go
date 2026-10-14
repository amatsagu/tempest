package discord

import (
	"bytes"
	"errors"
	"strconv"
	"time"
)

var ErrInvalidSnowflake = errors.New("invalid snowflake")

var nullLiteral = []byte("null")

const discordEpoch = 1420070400000 // Discord epoch in milliseconds

// https://docs.discord.com/developers/reference#snowflakes
type Snowflake uint64

// https://docs.discord.com/developers/reference#snowflakes
type UserID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type ChannelID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type RoleID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type EmojiID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type GuildID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type CommandID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type MessageID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type AttachmentID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type AppID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type IntegrationID struct{ Snowflake }

// https://docs.discord.com/developers/reference#snowflakes
type SkuID struct{ Snowflake }

func StringToSnowflake(s string) (Snowflake, error) {
	i, err := strconv.ParseUint(s, 10, 64)
	return Snowflake(i), err
}

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s Snowflake) CreationTimestamp() time.Time {
	return time.UnixMilli(int64(s>>22 + discordEpoch))
}

func (s Snowflake) Worker() uint8 {
	return uint8(s & 0x3E0000 >> 17)
}

func (s Snowflake) PID() uint8 {
	return uint8(s & 0x1F000 >> 12)
}

func (s Snowflake) Increment() uint16 {
	return uint16(s & 0xFFF)
}

func (s *Snowflake) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, nullLiteral) {
		*s = 0
		return nil
	}

	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}

	if len(b) == 0 {
		*s = 0
		return nil
	}

	val, ok := parseUint64Bytes(b)
	if !ok {
		return ErrInvalidSnowflake
	}

	*s = Snowflake(val)
	return nil
}

// AppendText lets encoding/json/v2 emit the snowflake as a quoted JSON string
// by appending straight into the encoder's buffer, avoiding the temporary
// []byte allocation that a MarshalJSON method would require.
func (s Snowflake) AppendText(b []byte) ([]byte, error) {
	return strconv.AppendUint(b, uint64(s), 10), nil
}

func (u UserID) Mention() string {
	if u.Snowflake == 0 {
		return ""
	}

	var buf [23]byte
	buf[0] = '<'
	buf[1] = '@'
	b := strconv.AppendUint(buf[:2], uint64(u.Snowflake), 10)
	b = append(b, '>')
	return string(b)
}

func (c ChannelID) Mention() string {
	if c.Snowflake == 0 {
		return ""
	}

	var buf [23]byte
	buf[0] = '<'
	buf[1] = '#'
	b := strconv.AppendUint(buf[:2], uint64(c.Snowflake), 10)
	b = append(b, '>')
	return string(b)
}

func (r RoleID) Mention() string {
	if r.Snowflake == 0 {
		return ""
	}

	var buf [24]byte
	buf[0] = '<'
	buf[1] = '@'
	buf[2] = '&'
	b := strconv.AppendUint(buf[:3], uint64(r.Snowflake), 10)
	b = append(b, '>')
	return string(b)
}

func (e EmojiID) Mention(name string, animated bool) string {
	if e.Snowflake == 0 {
		return ""
	}

	// '<a:' (3) + name + ':' (1) + 20 digits + '>' (1) = 25 + len(name) max
	prefixLen := 2
	if animated {
		prefixLen = 3
	}

	buf := make([]byte, 0, prefixLen+len(name)+22)
	buf = append(buf, '<')
	if animated {
		buf = append(buf, 'a')
	}
	buf = append(buf, ':')
	buf = append(buf, name...)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, uint64(e.Snowflake), 10)
	buf = append(buf, '>')

	return string(buf)
}

func (c CommandID) Mention(name string) string {
	if c.Snowflake == 0 || len(name) == 0 {
		return ""
	}

	buf := make([]byte, 0, 24+len(name))
	buf = append(buf, '<', '/')
	buf = append(buf, name...)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, uint64(c.Snowflake), 10)
	buf = append(buf, '>')

	return string(buf)
}

func parseUint64Bytes(b []byte) (uint64, bool) {
	if len(b) == 0 {
		return 0, false
	}

	var n uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}

		// Discord IDs are currently 17-19 digits, but check overflow (just in case)
		d := uint64(c - '0')
		if n > (18446744073709551615-d)/10 {
			return 0, false
		}
		n = n*10 + d
	}
	return n, true
}
