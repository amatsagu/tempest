package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"
)

type ClientOptions struct {
	ApplicationID     Snowflake // The app's user id. (default: <nil>)
	PublicKey         string    // Hash like key used to verify incoming payloads from Discord. (default: <nil>)
	Rest              *Rest
	CommandMiddleware func(itx CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	ComponentHandler  func(itx ComponentInteraction)    // Function that runs for each unhandled component.
	ModalHandler      func(itx ModalInteraction)        // Function that runs for each unhandled modal.
}

// Please avoid creating raw Client struct unless you know what you're doing. Use CreateClient function instead.
type Client struct {
	Rest          *Rest
	ApplicationID Snowflake
	PublicKey     ed25519.PublicKey

	commands   map[string]map[string]Command         // Internal cache for commands. Only writeable before starting application!
	components map[string]func(ComponentInteraction) // Internal cache for "static" components. Only writeable before starting application!
	modals     map[string]func(ModalInteraction)     // Internal cache for "static" modals. Only writeable before starting application!

	qMu              sync.RWMutex // Shated mutex for dynamic, components & modals.
	queuedComponents map[string]chan *ComponentInteraction
	queuedModals     map[string]chan *ModalInteraction

	commandMiddlewareHandler func(itx CommandInteraction) bool // From options, called before each slash command.
	componentHandler         func(itx ComponentInteraction)
	modalHandler             func(itx ModalInteraction)
	running                  bool // Whether client's web server is already launched.
}

// Makes client dynamically "listen" incoming component type interactions.
// When component custom id matches - it'll send back interaction through channel.
// On timeout (min 2s -> max 15min) - client will send <nil> through channel and automatically call close function.
func (client *Client) AwaitComponent(customIDs []string, timeout time.Duration) (<-chan *ComponentInteraction, func(), error) {
	for _, ID := range customIDs {
		_, exists := client.components[ID]
		if exists {
			return nil, nil, errors.New("client already has registered \"" + ID + "\" component as static (custom id already in use)")
		}
	}

	signalChannel := make(chan *ComponentInteraction)
	closeFunction := func() {
		if signalChannel != nil {
			client.qMu.Lock()
			for _, key := range customIDs {
				delete(client.queuedComponents, key)
			}
			client.qMu.Unlock()

			close(signalChannel)
			signalChannel = nil
		}
	}

	client.qMu.Lock()
	for _, ID := range customIDs {
		client.queuedComponents[ID] = signalChannel
	}
	client.qMu.Unlock()

	maxTime, minTime := time.Duration(time.Minute*15), time.Duration(time.Second*2)
	if timeout > maxTime {
		timeout = maxTime
	} else if timeout < minTime {
		timeout = minTime
	}

	time.AfterFunc(timeout, closeFunction)
	return signalChannel, closeFunction, nil
}

// Makes client dynamically "listen" incoming modal type interactions.
// When modal custom id matches - it'll send back interaction through channel.
// On timeout (min 30s -> max 15min) - client will send <nil> through channel and automatically call close function.
func (client *Client) AwaitModal(customID string, timeout time.Duration) (<-chan *ModalInteraction, func(), error) {
	_, exists := client.components[customID]
	if exists {
		return nil, nil, errors.New("client already has registered \"" + customID + "\" modal as static (custom id already in use)")
	}

	signalChannel := make(chan *ModalInteraction)
	closeFunction := func() {
		if signalChannel != nil {
			client.qMu.Lock()
			delete(client.queuedModals, customID)
			client.qMu.Unlock()
			close(signalChannel)
			signalChannel = nil
		}
	}

	client.qMu.Lock()
	client.queuedModals[customID] = signalChannel
	client.qMu.Unlock()

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
		return errors.New("client is already running")
	}

	if route == "" {
		route = "/"
	}

	client.running = true
	http.HandleFunc(route, client.handleRequest)
	return http.ListenAndServe(address, nil)
}

func (client *Client) ListenAndServeTLS(route string, address string, certFile, keyFile string) error {
	if client.running {
		return errors.New("client is already running")
	}

	if route == "" {
		route = "/"
	}

	client.running = true
	http.HandleFunc(route, client.handleRequest)
	return http.ListenAndServeTLS(address, certFile, keyFile, nil)
}

func NewClient(options ClientOptions) *Client {
	discordPublicKey, err := hex.DecodeString(options.PublicKey)
	if err != nil {
		panic("failed to decode \"%s\" discord's public key (check if it's correct key)")
	}

	return &Client{
		Rest:                     options.Rest,
		ApplicationID:            options.ApplicationID,
		PublicKey:                ed25519.PublicKey(discordPublicKey),
		commands:                 make(map[string]map[string]Command),
		components:               make(map[string]func(ComponentInteraction)),
		modals:                   make(map[string]func(ModalInteraction)),
		queuedComponents:         make(map[string]chan *ComponentInteraction),
		queuedModals:             make(map[string]chan *ModalInteraction),
		commandMiddlewareHandler: options.CommandMiddleware,
		componentHandler:         options.ComponentHandler,
		modalHandler:             options.ModalHandler,
		running:                  false,
	}
}
