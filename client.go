package tempest

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type ClientOptions struct {
	PublicKey        string // Hash like key used to verify incoming payloads from Discord. (default: <nil>)
	Rest             Rest
	HTTPServer       HTTPServer
	HTTPServeMux     HTTPServeMux
	PreCommandHook   func(cmd *Command, itx *CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	PostCommandHook  func(cmd *Command, itx *CommandInteraction)      // Function that runs after each command.
	ComponentHandler func(itx *ComponentInteraction)                  // Function that runs for each unhandled component.
	ModalHandler     func(itx *ModalInteraction)                      // Function that runs for each unhandled modal.
}

type Client struct {
	Rest          Rest
	ApplicationID Snowflake
	PublicKey     ed25519.PublicKey
	httpServer    HTTPServer
	httpServeMux  HTTPServeMux

	preCommandHandler  func(cmd *Command, itx *CommandInteraction) bool
	postCommandHandler func(cmd *Command, itx *CommandInteraction)
	componentHandler   func(itx *ComponentInteraction)
	modalHandler       func(itx *ModalInteraction)

	commands   map[string]map[string]Command         // Internal cache for commands. Only writeable before starting application!
	components map[string]func(ComponentInteraction) // Internal cache for "static" components. Only writeable before starting application!
	modals     map[string]func(ModalInteraction)     // Internal cache for "static" modals. Only writeable before starting application!

	qMu              sync.RWMutex // Shated mutex for dynamic, components & modals.
	queuedComponents map[string]chan *ComponentInteraction
	queuedModals     map[string]chan *ModalInteraction

	state atomic.Uint32
}

type ClientState uint8

const (
	INIT_STATE ClientState = iota
	RUNNING_STATE
	CLOSING_STATE
	CLOSED_STATE
)

func (client *Client) State() ClientState {
	return ClientState(client.state.Load())
}

// Makes client dynamically "listen" incoming component type interactions.
// When component custom id matches - it'll send back interaction through channel.
// On timeout (min 2s -> max 15min) - client will send <nil> through channel and automatically call close function.
//
// Warning! Components handled this way will already be acknowledged.
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
//
// Warning! Components handled this way will already be acknowledged.
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
	if client.State() != INIT_STATE {
		return errors.New("client is no longer in initialization state")
	}

	if route == "" {
		route = "/"
	}

	client.state.Store(uint32(RUNNING_STATE))
	client.httpServeMux.HandleFunc(route, client.handleRequest)
	client.httpServer.(*http.Server).Addr = address
	client.httpServer.(*http.Server).Handler = client.httpServeMux

	err := client.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (client *Client) ListenAndServeTLS(route string, address string, certFile, keyFile string) error {
	if client.State() != INIT_STATE {
		return errors.New("client is no longer in initialization state")
	}

	if route == "" {
		route = "/"
	}

	client.state.Store(uint32(RUNNING_STATE))
	client.httpServeMux.HandleFunc(route, client.handleRequest)
	client.httpServer.(*http.Server).Addr = address
	client.httpServer.(*http.Server).Handler = client.httpServeMux

	err := client.httpServer.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Tries to gracefully shutdown client. It'll clear all queued actions and shutdown underlying http server.
func (client *Client) Close(ctx context.Context) error {
	if client.State() == INIT_STATE {
		return errors.New("client is still in initiallization phase, there's nothing to shutdown")
	}

	if client.State() != RUNNING_STATE {
		return errors.New("client is already either closed or during closing process")
	}

	client.state.Store(uint32(CLOSING_STATE))
	client.preCommandHandler = func(cmd *Command, itx *CommandInteraction) bool {
		return false
	}

	for key, componentChannel := range client.queuedComponents {
		if _, open := <-componentChannel; open {
			close(componentChannel)
		}
		delete(client.queuedComponents, key)
	}

	for key, modalChannel := range client.queuedModals {
		if _, open := <-modalChannel; open {
			close(modalChannel)
		}
		delete(client.queuedModals, key)
	}

	err := client.httpServer.Shutdown(ctx)
	if err != nil {
		err2 := client.httpServer.Close()
		if err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}

	client.state.Store(uint32(CLOSED_STATE))
	return nil
}

func NewClient(options ClientOptions) *Client {
	discordPublicKey, err := hex.DecodeString(options.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	botUserID, err := extractUserIDFromToken(options.Rest.Token())
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	if options.HTTPServer == nil {
		options.HTTPServer = &http.Server{}
	}

	if options.HTTPServeMux == nil {
		options.HTTPServeMux = http.NewServeMux()
	}

	return &Client{
		Rest:          options.Rest,
		ApplicationID: botUserID,
		PublicKey:     ed25519.PublicKey(discordPublicKey),
		httpServer:    options.HTTPServer,
		httpServeMux:  options.HTTPServeMux,

		preCommandHandler:  options.PreCommandHook,
		postCommandHandler: options.PostCommandHook,
		componentHandler:   options.ComponentHandler,
		modalHandler:       options.ModalHandler,

		commands:         make(map[string]map[string]Command),
		components:       make(map[string]func(ComponentInteraction)),
		modals:           make(map[string]func(ModalInteraction)),
		queuedComponents: make(map[string]chan *ComponentInteraction),
		queuedModals:     make(map[string]chan *ModalInteraction),
	}
}
