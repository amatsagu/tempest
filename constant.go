package tempest

import (
	"fmt"

	"github.com/amatsagu/tempest/discord"
)

const (
	CONTENT_TYPE_JSON     = "application/json"
	MAX_REQUEST_BODY_SIZE = 1024 * 1024 // Interaction HTTP request body limit: 1 MiB.
	ROOT_PLACEHOLDER      = "-"
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = fmt.Appendf(nil, `{"type":%d}`, discord.PongResponseType)
	bodyAcknowledgeResponse    = fmt.Appendf(nil, `{"type":%d}`, discord.DeferredUpdateMessageResponseType)
	bodyUnknownCommandResponse = fmt.Appendf(nil, `{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.","flags":%d}}`, discord.ChannelMessageWithSourceResponseType, discord.EphemeralMessageFlag)
)
