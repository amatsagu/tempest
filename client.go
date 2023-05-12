package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
)

type ClientOptions struct {
	ApplicationID              Snowflake                                         // The app's user id. (default: <nil>)
	PublicKey                  string                                            // Hash like key used to verify incoming payloads from Discord. (default: <nil>)
	Token                      string                                            // The auth token to use. Bot tokens should be prefixed with Bot (e.g. "Bot MTExIHlvdSAgdHJpZWQgMTEx.O5rKAA.dQw4w9WgXcQ_wpV-gGA4PSk_bm8"). Prefix-less bot tokens are deprecated. (default: <nil>)
	PreCommandExecutionHandler func(itx CommandInteraction) *ResponseMessageData // Function to call after doing initial processing but before executing slash command. Allows to attach own, global logic to all slash commands (similar to routing). Return pointer to ResponseData struct if you want to send messageand stop execution or <nil> to continue. (default: <nil>)
	InteractionHandler         func(itx Interaction)                             // Function to call on all unhandled interactions. (default: <nil>)
}

// Please avoid creating raw Client struct unless you know what you're doing. Use CreateClient function instead.
type Client struct {
	Rest          Rest
	User          User // Bot user will be defined after launching application.
	ApplicationID Snowflake
	PublicKey     ed25519.PublicKey

	commands                   map[string]map[string]Command
	preCommandExecutionHandler func(itx CommandInteraction) *ResponseMessageData // From options, called before each slash command.
	interactionHandler         func(itx Interaction)                             // From options, called on all unhandled interactions.
	running                    bool                                              // Whether client's web server is already launched.
}

// Starts bot on set route aka "endpoint". Setting example route = "/bot" and address = "192.168.0.7:9070" would make bot work under http://192.168.0.7:9070/bot.
// Set route as "/" or leave empty string to make it work on any URI (default).
func (client *Client) ListenAndServe(route string, address string) error {
	if client.running {
		panic("client's web server is already launched")
	}

	user, err := client.FetchUser(client.ApplicationID)
	if err != nil {
		panic("failed to fetch bot user's details (check if application id is correct & your internet connection works)\n")
	}
	client.User = user

	if route == "" {
		route = "/"
	}

	http.HandleFunc(route, client.handleDiscordWebhookRequests)
	return http.ListenAndServe(address, nil)
}

func (client *Client) ListenAndServeTLS(route string, address string, certFile, keyFile string) error {
	if client.running {
		panic("client's web server is already launched")
	}

	user, err := client.FetchUser(client.ApplicationID)
	if err != nil {
		panic("failed to fetch bot user's details (check if application id is correct & your internet connection works)\n")
	}
	client.User = user

	if route == "" {
		route = "/"
	}

	http.HandleFunc(route, client.handleDiscordWebhookRequests)
	return http.ListenAndServeTLS(address, certFile, keyFile, nil)
}

func CreateClient(options ClientOptions) Client {
	discordPublicKey, err := hex.DecodeString(options.PublicKey)
	if err != nil {
		panic("failed to decode \"%s\" discord's public key (check if it's correct key)")
	}

	client := Client{
		Rest:                       CreateRest(options.Token),
		ApplicationID:              options.ApplicationID,
		PublicKey:                  ed25519.PublicKey(discordPublicKey),
		commands:                   make(map[string]map[string]Command, 0),
		preCommandExecutionHandler: options.PreCommandExecutionHandler,
		interactionHandler:         options.InteractionHandler,
		running:                    false,
	}

	return client
}
