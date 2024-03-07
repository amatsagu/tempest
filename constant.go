package tempest

import "fmt"

const (
	DiscordAPIURL    = "https://discord.com/api/v10"
	DiscordCDNURL    = "https://cdn.discordapp.com"
	DiscordEpoch     = 1420070400000 // Discord epoch in milliseconds
	UserAgent        = "DiscordApp https://github.com/Amatsagu/tempest"
	ContentTypeJSON  = "application/json"
	ROOT_PLACEHOLDER = "-"
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = []byte(fmt.Sprintf(`{"type":%d}`, PONG_RESPONSE_TYPE))
	bodyAcknowledgeResponse    = []byte(fmt.Sprintf(`{"type":%d}`, DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE))
	bodyUnknownCommandResponse = []byte(fmt.Sprintf(`{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to trigger (/) unknown command. Please report this bug to bot owner.","flags":64}}`, CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE))
)
