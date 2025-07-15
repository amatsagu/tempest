package tempest

import (
	"fmt"
)

const (
	DISCORD_API_URL                    = "https://discord.com/api/v10"
	DISCORD_CDN_URL                    = "https://cdn.discordapp.com"
	DISCORD_EPOCH                      = 1420070400000 // Discord epoch in milliseconds
	USER_AGENT                         = "DiscordApp https://github.com/amatsagu/tempest"
	CONTENT_TYPE_JSON                  = "application/json"
	CONTENT_TYPE_OCTET_STREAM          = "application/octet-stream"
	CONTENT_MULTIPART_JSON_DESCRIPTION = `form-data; name="payload_json"`
	MAX_REQUEST_BODY_SIZE              = 1024 * 1024 // 1024 KB
	ROOT_PLACEHOLDER                   = "-"
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = fmt.Appendf(nil, `{"type":%d}`, PONG_RESPONSE_TYPE)
	bodyAcknowledgeResponse    = fmt.Appendf(nil, `{"type":%d}`, DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE)
	bodyUnknownCommandResponse = fmt.Appendf(nil, `{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.","flags":%d}}`, CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE, EPHEMERAL_MESSAGE_FLAG)
)
