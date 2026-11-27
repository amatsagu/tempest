package tempest

import (
	"fmt"
	"sync"
	"time"

	"github.com/amatsagu/tempest/discord"
)

const (
	USER_AGENT                         = "DiscordApp https://github.com/amatsagu/tempest"
	CONTENT_TYPE_JSON                  = "application/json"
	CONTENT_TYPE_OCTET_STREAM          = "application/octet-stream"
	CONTENT_MULTIPART_JSON_DESCRIPTION = `form-data; name="payload_json"`
	MAX_REQUEST_BODY_SIZE              = 1024 * 1024      // Interaction HTTP request body limit: 1 MiB.
	MAX_FILE_UPLOAD_SIZE               = 10 * 1024 * 1024 // 10 MB
	ROOT_PLACEHOLDER                   = "-"

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
	RATE_LIMITER_SWEEP_EVERY      = 50
	RATE_LIMITER_RESET_TIME_FUDGE = 100 * time.Millisecond
)

// discordAPIBaseURL is mutable so that unit tests (or advanced users)
// can replace it with a mock endpoint.
var (
	discordAPIBaseURL = "https://discord.com/api/v10"
	discordURLMu      sync.RWMutex
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = fmt.Appendf(nil, `{"type":%d}`, discord.PongResponseType)
	bodyAcknowledgeResponse    = fmt.Appendf(nil, `{"type":%d}`, discord.DeferredUpdateMessageResponseType)
	bodyUnknownCommandResponse = fmt.Appendf(nil, `{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.","flags":%d}}`, discord.ChannelMessageWithSourceResponseType, discord.EphemeralMessageFlag)
)

func DiscordAPIBaseURL() string {
	discordURLMu.RLock()
	url := discordAPIBaseURL
	discordURLMu.RUnlock()
	return url
}

func UpdateDiscordAPIBaseURL(url string) {
	discordURLMu.Lock()
	discordAPIBaseURL = url
	discordURLMu.Unlock()
}
