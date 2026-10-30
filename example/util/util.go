// Package util holds the small helpers shared by the example features.
package util

import "github.com/amatsagu/tempest"

// Ptr returns a pointer to v, for optional pointer fields on Discord objects.
func Ptr[T any](v T) *T { return &v }

// CommandHandler adapts an error-returning feature handler to the library's
// handler shape, which has no error return.
func CommandHandler(fn func(*tempest.CommandInteraction) error) func(*tempest.CommandInteraction) {
	return func(itx *tempest.CommandInteraction) {
		if err := fn(itx); err != nil {
			Logger.Printf("command %q failed (interaction %s): %v", itx.Data.Name, itx.ID, err)
		}
	}
}

// ComponentHandler adapts an error-returning feature handler to the library's
// component handler shape.
func ComponentHandler(fn func(*tempest.ComponentInteraction) error) func(tempest.ComponentInteraction) {
	return func(itx tempest.ComponentInteraction) {
		if err := fn(&itx); err != nil {
			Logger.Printf("component %q failed (interaction %s): %v", itx.Data.CustomID, itx.ID, err)
		}
	}
}

// ModalHandler adapts an error-returning feature handler to the library's modal
// handler shape.
func ModalHandler(fn func(*tempest.ModalInteraction) error) func(tempest.ModalInteraction) {
	return func(itx tempest.ModalInteraction) {
		if err := fn(&itx); err != nil {
			Logger.Printf("modal %q failed (interaction %s): %v", itx.Data.CustomID, itx.ID, err)
		}
	}
}

// StringOption returns a string option, if present.
func StringOption(itx *tempest.CommandInteraction, name string) (string, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok
}

// BoolOption returns a boolean option, if present.
func BoolOption(itx *tempest.CommandInteraction, name string) (bool, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return false, false
	}
	value, ok := raw.(bool)
	return value, ok
}

// SnowflakeOption returns a snowflake option. Discord sends ID options as strings,
// but this also accepts numeric JSON values.
func SnowflakeOption(itx *tempest.CommandInteraction, name string) (tempest.Snowflake, bool) {
	raw, ok := itx.GetOptionValue(name)
	if !ok {
		return 0, false
	}

	switch value := raw.(type) {
	case string:
		id, err := tempest.StringToSnowflake(value)
		return id, err == nil
	case float64:
		return tempest.Snowflake(uint64(value)), true
	case tempest.Snowflake:
		return value, true
	default:
		return 0, false
	}
}

// IntegerOption returns an integer option, if present.
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

// ChannelMention renders a channel ID as a Discord channel mention.
func ChannelMention(id tempest.Snowflake) string {
	return "<#" + id.String() + ">"
}

// EffectiveName returns the display name Discord shows for a user.
func EffectiveName(user tempest.User) string {
	if user.GlobalName != "" {
		return user.GlobalName
	}
	return user.Username
}

// Embed starts a rich embed with a title and description.
func Embed(title, description string) tempest.Embed {
	return tempest.Embed{Title: title, Description: description}
}

// EmbedField builds one embed field.
func EmbedField(name, value string, inline bool) tempest.EmbedField {
	return tempest.EmbedField{Name: name, Value: value, Inline: inline}
}

// EmbedFooter builds an embed footer.
func EmbedFooter(text string) tempest.EmbedFooter {
	return tempest.EmbedFooter{Text: text}
}
