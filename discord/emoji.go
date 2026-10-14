package discord

// https://docs.discord.com/developers/resources/emoji#emoji-object
type Emoji struct {
	User          *User    `json:"user,omitzero"`
	Name          string   `json:"name,omitzero"`
	Roles         []RoleID `json:"roles,omitzero"`
	ID            EmojiID  `json:"id,omitzero"`
	RequireColons bool     `json:"require_colons,omitzero"`
	Managed       bool     `json:"managed,omitzero"`
	Animated      bool     `json:"animated,omitzero"`
	Available     bool     `json:"available,omitzero"`
}

func (e Emoji) Mention() string {
	if e.ID.Snowflake == 0 {
		return e.Name
	}
	return e.ID.Mention(e.Name, e.Animated)
}
