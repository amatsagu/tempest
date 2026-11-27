package discord

import "sync"

const DiscordEpoch = 1420070400000 // Discord epoch in milliseconds

// discordCDNBaseURL is mutable so that unit tests (or advanced users)
// can replace it with a mock endpoint.
var (
	discordCDNBaseURL = "https://cdn.discordapp.com"
	discordCDNURLMu   sync.RWMutex
)

func DiscordCDNBaseURL() string {
	discordCDNURLMu.RLock()
	url := discordCDNBaseURL
	discordCDNURLMu.RUnlock()
	return url
}

func UpdateDiscordCDNBaseURL(url string) {
	discordCDNURLMu.Lock()
	discordCDNBaseURL = url
	discordCDNURLMu.Unlock()
}
