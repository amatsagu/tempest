package tempest

import (
	"fmt"
	"sync"
	"time"
)

const (
	DISCORD_EPOCH                      = 1420070400000 // Discord epoch in milliseconds
	USER_AGENT                         = "DiscordApp https://github.com/amatsagu/tempest"
	GATEWAY_AGENT                      = "tempest"
	CONTENT_TYPE_JSON                  = "application/json"
	CONTENT_TYPE_OCTET_STREAM          = "application/octet-stream"
	CONTENT_MULTIPART_JSON_DESCRIPTION = `form-data; name="payload_json"`
	MAX_REQUEST_BODY_SIZE              = 1024 * 1024      // Interaction HTTP request body limit: 1 MiB.
	MAX_FILE_UPLOAD_SIZE               = 10 * 1024 * 1024 // 10 MB
	ROOT_PLACEHOLDER                   = "-"

	// Gateway events such as GUILD_CREATE can legitimately exceed the 1 MiB
	// interaction body limit. This is a hard ceiling checked while reading, not
	// an upfront allocation, so it only costs memory for messages actually sent.
	MAX_GATEWAY_MESSAGE_SIZE = 16 * 1024 * 1024 // 16 MiB

	GATEWAY_READ_TIMEOUT      = 90 * time.Second
	GATEWAY_WRITE_TIMEOUT     = 10 * time.Second
	GATEWAY_HANDSHAKE_TIMEOUT = 30 * time.Second
	GATEWAY_RECONNECT_DELAY   = 1 * time.Second
	GATEWAY_DIAL_RETRY_DELAY  = 5 * time.Second

	REST_DIAL_TIMEOUT             = 5 * time.Second
	REST_KEEP_ALIVE               = 30 * time.Second
	REST_MAX_IDLE_CONNS           = 256
	REST_MAX_IDLE_CONNS_PER_HOST  = 64
	REST_IDLE_CONN_TIMEOUT        = 90 * time.Second
	REST_TLS_HANDSHAKE_TIMEOUT    = 10 * time.Second
	REST_EXPECT_CONTINUE_TIMEOUT  = time.Second
	REST_FORCE_ATTEMPT_HTTP2      = true
	REST_DEFAULT_MAX_WAIT_TIME    = 10 * time.Second
	REST_DEFAULT_MAX_RETRIES      = 3
	REST_DEFAULT_RETRY_THRESHOLD  = 60
	RATE_LIMITER_SWEEP_INTERVAL   = 30 * time.Minute
	RATE_LIMITER_SWEEP_THRESHOLD  = 2500
	RATE_LIMITER_RESET_TIME_FUDGE = 100 * time.Millisecond
)

// discordAPIBaseURL / discordCDNBaseURL are mutable so that unit tests
// (or advanced users) can replace them with mock endpoints.
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
