package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"net/http"
	"time"
)

type ClientOptions struct {
	ApplicationID     Snowflake                                         // The app's user id. (default: <nil>)
	PublicKey         string                                            // Hash like key used to verify incoming payloads from Discord. (default: <nil>)
	Token             string                                            // The auth token to use. Bot tokens should be prefixed with Bot (e.g. "Bot MTExIHlvdSAgdHJpZWQgMTEx.O5rKAA.dQw4w9WgXcQ_wpV-gGA4PSk_bm8"). Prefix-less bot tokens are deprecated. (default: <nil>)
	CommandMiddleware func(itx CommandInteraction) *ResponseMessageData // Functions that runs before each command. Returning *ResponseMessageData will send your payload (for example error or cooldown message) and stop command execution.
}

// Please avoid creating raw Client struct unless you know what you're doing. Use CreateClient function instead.
type Client struct {
	Rest          Rest
	User          User // Bot user will be defined after launching application.
	ApplicationID Snowflake
	PublicKey     ed25519.PublicKey

	commands                   map[string]map[string]Command
	components                 map[string]func(ComponentInteraction) // Cache for registered, "static" components
	modals                     map[string]func(ModalInteraction)     // Cache for registered, "static" modals
	queuedComponents           map[string]*(chan *ComponentInteraction)
	queuedModals               map[string]chan *ModalInteraction
	preCommandExecutionHandler func(itx CommandInteraction) *ResponseMessageData // From options, called before each slash command.
	running                    bool                                              // Whether client's web server is already launched.
}

// Makes client "listen" incoming component type interactions.
// When component custom id matches - it'll send back interaction through channel.
// On timeout (min 2s -> max 15min) - client will send <nil> through channel and automatically call close function.
//
// Warning: You still need to acknowledge received component interaction.
//
// Warning: Using this method creates state inpurity. Don't use this method if you want to build "cache-free" applications and scale them behind balance loader.
func (client Client) AwaitComponent(customIDs []string, timeout time.Duration) (chan *ComponentInteraction, func(), error) {
	for _, ID := range customIDs {
		_, exists := client.components[ID]
		if exists {
			return nil, nil, errors.New("client already has registered \"" + ID + "\" component as static (custom id already in use)")
		}
	}

	signalChannel := make(chan *ComponentInteraction)
	closeFunction := func() {
		if signalChannel != nil {
			for _, key := range customIDs {
				delete(client.queuedComponents, key)
			}

			close(signalChannel)
			signalChannel = nil
		}
	}

	for _, ID := range customIDs {
		client.queuedComponents[ID] = &signalChannel
	}

	maxTime, minTime := time.Duration(time.Minute*15), time.Duration(time.Second*2)
	if timeout > maxTime {
		timeout = maxTime
	} else if timeout < minTime {
		timeout = minTime
	}

	time.AfterFunc(timeout, closeFunction)
	return signalChannel, closeFunction, nil
}

// Makes client "listen" incoming modal type interactions.
// When modal custom id matches - it'll send back interaction through channel.
// On timeout (min 30s -> max 15min) - client will send <nil> through channel and automatically call close function.
//
// Warning: You still need to acknowledge received modal interaction.
//
// Warning: Using this method creates state inpurity. Don't use this method if you want to build "cache-free" applications and scale them behind balance loader.
func (client Client) AwaitModal(customID string, timeout time.Duration) (chan *ModalInteraction, func(), error) {
	_, exists := client.components[customID]
	if exists {
		return nil, nil, errors.New("client already has registered \"" + customID + "\" modal as static (custom id already in use)")
	}

	signalChannel := make(chan *ModalInteraction)
	closeFunction := func() {
		if signalChannel != nil {
			delete(client.queuedModals, customID)
			close(signalChannel)
			signalChannel = nil
		}
	}

	client.queuedModals[customID] = signalChannel
	maxTime, minTime := time.Duration(time.Minute*15), time.Duration(time.Second*30)
	if timeout > maxTime {
		timeout = maxTime
	} else if timeout < minTime {
		timeout = minTime
	}

	time.AfterFunc(timeout, closeFunction)
	return signalChannel, closeFunction, nil
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
		Rest:          CreateRest(options.Token),
		ApplicationID: options.ApplicationID,
		PublicKey:     ed25519.PublicKey(discordPublicKey),
		// commands:                   make(map[string]map[string]Command, 0),
		// queuedComponents:           make(map[string]*chan *Interaction, 0),
		preCommandExecutionHandler: options.CommandMiddleware,
		running:                    false,
	}

	return client
}
