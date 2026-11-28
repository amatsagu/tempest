// Package util holds shared option and embed helpers for the example bot.
package util

import (
	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
)

func StringOption(itx *tempest.CommandInteraction, name string) (string, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok
}

func BoolOption(itx *tempest.CommandInteraction, name string) (bool, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return false, false
	}
	value, ok := raw.(bool)
	return value, ok
}

// Discord sends ID options as strings, though tests or custom callers may pass numbers.
func SnowflakeOption(itx *tempest.CommandInteraction, name string) (discord.Snowflake, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return 0, false
	}

	switch value := raw.(type) {
	case string:
		id, err := discord.StringToSnowflake(value)
		return id, err == nil
	case float64:
		return discord.Snowflake(uint64(value)), true
	case discord.Snowflake:
		return value, true
	default:
		return 0, false
	}
}

func IntegerOption(itx *tempest.CommandInteraction, name string) (int64, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return 0, false
	}

	switch value := raw.(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case int:
		return int64(value), true
	default:
		return 0, false
	}
}

func FocusedString(itx *tempest.CommandInteraction) string {
	for _, option := range itx.Data.Options {
		if !option.Focused {
			continue
		}
		value, _ := option.Value.(string)
		return value
	}
	return ""
}

func EffectiveName(user discord.User) string {
	if user.GlobalName != "" {
		return user.GlobalName
	}
	return user.Username
}

func Embed(title, description string) discord.Embed {
	return discord.Embed{Title: title, Description: description}
}

func EmbedField(name, value string, inline bool) discord.EmbedField {
	return discord.EmbedField{Name: name, Value: value, Inline: inline}
}

func EmbedFooter(text string) discord.EmbedFooter {
	return discord.EmbedFooter{Text: text}
}
