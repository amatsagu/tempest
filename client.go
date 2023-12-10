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
	Addr             string
	CertFile         string
	KeyFile          string
	Handler          http.Handler
	PublicKey        string // Hash like key used to verify incoming payloads from Discord. (default: <nil>)
	Rest             Rest
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

	preCommandHandler     func(cmd *Command, itx *CommandInteraction) bool
	postCommandHandler    func(cmd *Command, itx *CommandInteraction)
	unknownCommandHandler func(itx *CommandInteraction) // Function to call when client receives not registered command. This only happens if there's desync between local Client#commands cache and Discord API.
	componentHandler      func(itx *ComponentInteraction)
	modalHandler          func(itx *ModalInteraction)

	commands   map[string]map[string]Command         // Internal cache for commands. Only writeable before starting application!
	components map[string]func(ComponentInteraction) // Internal cache for "static" components. Only writeable before starting application!
	modals     map[string]func(ModalInteraction)     // Internal cache for "static" modals. Only writeable before starting application!

	qMu              sync.RWMutex // Shated mutex for dynamic, components & modals.
	queuedComponents map[string]chan *ComponentInteraction
	queuedModals     map[string]chan *ModalInteraction

	running bool // Whether client's web server is already launched.
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

func NewDefaultClient(options ClientOptions) *Client {
	discordPublicKey, err := hex.DecodeString(options.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	botUserID, err := extractUserIDFromToken(options.Rest.Token())
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	return &Client{
		Rest:          options.Rest,
		ApplicationID: botUserID,
		PublicKey:     ed25519.PublicKey(discordPublicKey),
		httpServer:    &http.Server{Addr: options.Addr, Handler: options.Handler},

		preCommandHandler:  options.PreCommandHook,
		postCommandHandler: options.PostCommandHook,
		unknownCommandHandler: func(itx *CommandInteraction) {
			itx.SendLinearReply("Oh uh.. It looks like you tried to trigger (/) unknown command. Please report this bug to bot owner.", true)
		},
		componentHandler: options.ComponentHandler,
		modalHandler:     options.ModalHandler,

		commands: make(map[string]map[string]Command),
		running:  false,
	}
}
