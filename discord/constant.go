package discord

import (
	"fmt"
)

const (
	DISCORD_API_URL     = "https://discord.com/api/v10"
	DISCORD_GATEWAY_URL = "wss://gateway.discord.gg/?v=10&encoding=json"
	DISCORD_CDN_URL     = "https://cdn.discordapp.com"
	LIBRARY_NAME        = "Qord"
	DISCORD_EPOCH       = 1420070400000 // Discord epoch in milliseconds
	USER_AGENT          = "DiscordApp https://github.com/amatsagu/tempest"
	CONTENT_TYPE_JSON   = "application/json"
	ROOT_PLACEHOLDER    = "-"
)

// Prepare those replies as they never change so there's no point in re-creating them each time.
var (
	bodyPingResponse           = fmt.Appendf(nil, `{"type":%d}`, PONG_RESPONSE_TYPE)
	bodyAcknowledgeResponse    = fmt.Appendf(nil, `{"type":%d}`, DEFERRED_UPDATE_MESSAGE_RESPONSE_TYPE)
	bodyUnknownCommandResponse = fmt.Appendf(nil, `{"type":%d,"data":{"content":"Oh uh.. It looks like you tried to use outdated/unknown slash command. Please report this bug to bot owner.","flags":%d}}`, CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE_TYPE, EPHEMERAL_MESSAGE_FLAG)
)

var (
	requestSwapNullArray  = []byte("[{}]")
	requestSwapEmptyArray = []byte("[]")
)
