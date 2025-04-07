package tempest

import (
	"fmt"
)

const (
	DISCORD_API_URL   = "https://discord.com/api/v10"
	DISCORD_CDN_URL   = "https://cdn.discordapp.com"
	DISCORD_EPOCH     = 1420070400000 // Discord epoch in milliseconds
	USER_AGENT        = "DiscordApp https://github.com/amatsagu/tempest"
	CONTENT_TYPE_JSON = "application/json"
	ROOT_PLACEHOLDER  = "-"
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = []byte(fmt.Sprintf(`{"type":%d}`, PONG_RESPONSE_TYPE))
	bodyAcknowledgeResponse    = []byte(fmt.Sprintf(`{"type":%d}`, DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE))
	bodyUnknownCommandResponse = []byte(fmt.Sprintf(`{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to trigger (/) unknown command. Please report this bug to bot owner.","flags":64}}`, CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE))
)

var (
	requestSwapNullArray  = []byte("[null]")
	requestSwapEmptyArray = []byte("[]")
)
