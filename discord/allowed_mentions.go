package discord

// https://docs.discord.com/developers/resources/message#allowed-mentions-object-allowed-mention-types
type AllowedMentionType string

const (
	AllowedMentionRoles    AllowedMentionType = "roles"
	AllowedMentionUsers    AllowedMentionType = "users"
	AllowedMentionEveryone AllowedMentionType = "everyone"
)

// https://docs.discord.com/developers/resources/message#allowed-mentions-object
type AllowedMentions struct {
	Parse       []AllowedMentionType `json:"parse,omitzero"`
	Roles       []RoleID             `json:"roles,omitzero"`
	Users       []UserID             `json:"users,omitzero"`
	RepliedUser bool                 `json:"replied_user,omitzero"`
}
