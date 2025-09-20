package api

const (
	LIBRARY_VERSION                    = "v2.0"
	DISCORD_API_URL                    = "https://discord.com/api/v10"
	DISCORD_CDN_URL                    = "https://cdn.discordapp.com"
	DISCORD_EPOCH                      = 1420070400000 // Discord epoch in milliseconds
	USER_AGENT                         = "DiscordBot (https://github.com/amatsagu/qord, " + LIBRARY_VERSION + ")"
	CONTENT_TYPE_JSON                  = "application/json"
	CONTENT_TYPE_OCTET_STREAM          = "application/octet-stream"
	CONTENT_MULTIPART_JSON_DESCRIPTION = `form-data; name="payload_json"`
	MAX_REQUEST_BODY_SIZE              = 1024 * 1024 // 1024 KB
	ROOT_PLACEHOLDER                   = "-"
)
