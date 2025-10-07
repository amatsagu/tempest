package tempest

// https://discord.com/developers/docs/resources/guild#unavailable-guild-object
type UnavailableGuild struct {
	ID          Snowflake `json:"id"`
	Unavailable bool      `json:"unavailable"`
}
