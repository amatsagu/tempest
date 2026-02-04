package tempest

import (
	"fmt"
	"sync"
)

const (
	DISCORD_EPOCH                      = 1420070400000 // Discord epoch in milliseconds
	USER_AGENT                         = "DiscordApp https://github.com/amatsagu/tempest"
	CONTENT_TYPE_JSON                  = "application/json"
	CONTENT_TYPE_OCTET_STREAM          = "application/octet-stream"
	CONTENT_MULTIPART_JSON_DESCRIPTION = `form-data; name="payload_json"`
	MAX_REQUEST_BODY_SIZE              = 1024 * 1024 // 1024 KB
	ROOT_PLACEHOLDER                   = "-"
)

var (
	discordAPIBaseURL = "https://discord.com/api/v10"
	discordCDNBaseURL = "https://cdn.discordapp.com"
	discordURLMu      sync.RWMutex
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = fmt.Appendf(nil, `{"type":%d}`, PONG_RESPONSE_TYPE)
	bodyAcknowledgeResponse    = fmt.Appendf(nil, `{"type":%d}`, DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE)
	bodyUnknownCommandResponse = fmt.Appendf(nil, `{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.","flags":%d}}`, CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE, EPHEMERAL_MESSAGE_FLAG)
)

func DiscordAPIBaseURL() string {
	discordURLMu.RLock()
	url := discordAPIBaseURL
	discordURLMu.RUnlock()
	return url
}

func DiscordCDNBaseURL() string {
	discordURLMu.RLock()
	url := discordCDNBaseURL
	discordURLMu.RUnlock()
	return url
}

func UpdateDiscordAPIBaseURL(url string) {
	discordURLMu.Lock()
	discordAPIBaseURL = url
	discordURLMu.Unlock()
}

func UpdateDiscordCDNBaseURL(url string) {
	discordURLMu.Lock()
	discordCDNBaseURL = url
	discordURLMu.Unlock()
}
