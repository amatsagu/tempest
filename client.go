package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"
)

// Client is the core tempest entrypoint
type Client struct {
	ApplicationID Snowflake
	PublicKey     ed25519.PublicKey
	Rest          *Rest

	commands         *SharedMap[string, Command]
	commandContexts  []InteractionContextType
	staticComponents *SharedMap[string, func(ComponentInteraction)]
	staticModals     *SharedMap[string, func(ModalInteraction)]
	jsonBufferPool   *sync.Pool

	preCommandHandler  func(cmd Command, itx CommandInteraction) bool
	postCommandHandler func(cmd Command, itx CommandInteraction)
	componentHandler   func(itx ComponentInteraction)
	modalHandler       func(itx ModalInteraction)

	queuedComponents *SharedMap[string, chan ComponentInteraction]
	queuedModals     *SharedMap[string, chan ModalInteraction]
}

type ClientOptions struct {
	Token                      string
	PublicKey                  string
	JSONBufferSize             uint
	DefaultInteractionContexts []InteractionContextType

	PreCommandHook   func(cmd Command, itx CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	PostCommandHook  func(cmd Command, itx CommandInteraction)      // Function that runs after each command.
	ComponentHandler func(itx ComponentInteraction)                 // Function that runs for each unhandled component.
	ModalHandler     func(itx ModalInteraction)                     // Function that runs for each unhandled modal.
}

func NewClient(opt ClientOptions) Client {
	discordPublicKey, err := hex.DecodeString(opt.PublicKey)
	if err != nil {
		panic("failed to decode discord's public key (check if it's correct key): " + err.Error())
	}

	botUserID, err := extractUserIDFromToken(opt.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	var poolSize uint = 4096
	if opt.JSONBufferSize > poolSize {
		poolSize = opt.JSONBufferSize
	}

	contexts := []InteractionContextType{0}
	if opt.DefaultInteractionContexts != nil || len(opt.DefaultInteractionContexts) > 0 {
		contexts = opt.DefaultInteractionContexts
	}

	return Client{
		ApplicationID:    botUserID,
		PublicKey:        discordPublicKey,
		Rest:             NewRest(opt.Token),
		commands:         NewSharedMap[string, Command](),
		commandContexts:  contexts,
		staticComponents: NewSharedMap[string, func(ComponentInteraction)](),
		staticModals:     NewSharedMap[string, func(ModalInteraction)](),
		jsonBufferPool: &sync.Pool{
			New: func() any {
				buf := make([]byte, poolSize) // start with a decent buffer
				return &buf
			},
		},
		preCommandHandler:  opt.PreCommandHook,
		postCommandHandler: opt.PostCommandHook,
		componentHandler:   opt.ComponentHandler,
		modalHandler:       opt.ModalHandler,
		queuedComponents:   NewSharedMap[string, chan ComponentInteraction](),
		queuedModals:       NewSharedMap[string, chan ModalInteraction](),
	}
}

// Makes client dynamically "listen" incoming component type interactions.
// When component custom id matches - it'll send back interaction through channel.
// On timeout (min 1s -> max 15min) - client will automatically call close function.
//
// Warning! Components handled this way will already be acknowledged.
func (client *Client) AwaitComponent(customIDs []string, timeout time.Duration) (<-chan ComponentInteraction, func(), error) {
	for _, ID := range customIDs {
		if client.staticComponents.Has(ID) {
			return nil, nil, errors.New("client already has registered static component with custom ID = " + ID + " (custom id already in use)")
		}

		if client.queuedComponents.Has(ID) {
			return nil, nil, errors.New("client already has registered dynamic (queued) component with custom ID = " + ID + " (custom id already in use)")
		}
	}

	signalChannel := make(chan ComponentInteraction)
	closeFunction := func() {
		if signalChannel != nil {
			client.queuedComponents.mu.Lock()
			for _, key := range customIDs {
				delete(client.queuedComponents.cache, key)
			}
			client.queuedComponents.mu.Unlock()

			close(signalChannel)
			signalChannel = nil
		}
	}

	client.queuedComponents.mu.Lock()
	for _, key := range customIDs {
		client.queuedComponents.cache[key] = signalChannel
	}
	client.queuedComponents.mu.Unlock()

	maxTime, minTime := time.Duration(time.Minute*15), time.Duration(time.Second)
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
// On timeout (min 30s -> max 15min) - client will automatically call close function.
//
// Warning! Components handled this way will already be acknowledged.
func (client *Client) AwaitModal(customID string, timeout time.Duration) (<-chan ModalInteraction, func(), error) {
	if client.queuedModals.Has(customID) {
		return nil, nil, errors.New("client already has registered static modal with custom ID = " + customID + " (custom id already in use)")
	}

	signalChannel := make(chan ModalInteraction)
	closeFunction := func() {
		if signalChannel != nil {
			client.queuedModals.Delete(customID)
			close(signalChannel)
			signalChannel = nil
		}
	}

	client.queuedModals.Set(customID, signalChannel)
	maxTime, minTime := time.Duration(time.Minute*15), time.Duration(time.Second*30)
	if timeout > maxTime {
		timeout = maxTime
	} else if timeout < minTime {
		timeout = minTime
	}

	time.AfterFunc(timeout, closeFunction)
	return signalChannel, closeFunction, nil
}

// Pings Discord API and returns time it took to get response.
func (client *Client) Ping() time.Duration {
	start := time.Now()
	client.Rest.Request(http.MethodGet, "/gateway", nil)
	return time.Since(start)
}

func (client *Client) SendMessage(channelID Snowflake, message Message, files []os.File) (Message, error) {
	raw, err := client.Rest.RequestWithFiles(http.MethodPost, "/channels/"+channelID.String()+"/messages", message, files)
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client *Client) SendLinearMessage(channelID Snowflake, content string) (Message, error) {
	return client.SendMessage(channelID, Message{Content: content}, nil)
}

// Creates (or fetches if already exists) user's private text channel (DM) and tries to send message into it.
// Warning! Discord's user channels endpoint has huge rate limits so please reuse Message#ChannelID whenever possible.
func (client *Client) SendPrivateMessage(userID Snowflake, content Message, files []os.File) (Message, error) {
	res := make(map[string]interface{}, 0)
	res["recipient_id"] = userID

	raw, err := client.Rest.Request(http.MethodPost, "/users/@me/channels", res)
	if err != nil {
		return Message{}, err
	}

	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	channelID, err := StringToSnowflake(res["id"].(string))
	if err != nil {
		return Message{}, err
	}

	msg, err := client.SendMessage(channelID, content, files)
	msg.ChannelID = channelID // Just in case.

	return msg, err
}

func (client *Client) EditMessage(channelID Snowflake, messageID Snowflake, content Message) error {
	_, err := client.Rest.Request(http.MethodPatch, "/channels/"+channelID.String()+"/messages/"+messageID.String(), content)
	return err
}

func (client *Client) DeleteMessage(channelID Snowflake, messageID Snowflake) error {
	_, err := client.Rest.Request(http.MethodDelete, "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	return err
}

func (client *Client) CrosspostMessage(channelID Snowflake, messageID Snowflake) error {
	_, err := client.Rest.Request(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/crosspost", nil)
	return err
}

func (client *Client) FetchUser(id Snowflake) (User, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/users/"+id.String(), nil)
	if err != nil {
		return User{}, err
	}

	res := User{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return User{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client *Client) FetchMember(guildID Snowflake, memberID Snowflake) (Member, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/guilds/"+guildID.String()+"/members/"+memberID.String(), nil)
	if err != nil {
		return Member{}, err
	}

	res := Member{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Member{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}
