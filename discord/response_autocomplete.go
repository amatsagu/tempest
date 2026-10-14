package discord

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-response-object-autocomplete
type ResponseAutocompleteData struct {
	Choices []CommandOptionChoice `json:"choices"`
}
