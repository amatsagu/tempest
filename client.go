package tempest

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

	preCommandHandler  func(cmd Command, itx *CommandInteraction) bool
	postCommandHandler func(cmd Command, itx *CommandInteraction)
	componentHandler   func(itx *ComponentInteraction)
	modalHandler       func(itx *ModalInteraction)

	queuedComponents *SharedMap[string, chan *ComponentInteraction]
	queuedModals     *SharedMap[string, chan *ModalInteraction]
}

type ClientOptions struct {
	Token                      string
	PublicKey                  string
	DefaultInteractionContexts []InteractionContextType

	PreCommandHook   func(cmd Command, itx *CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	PostCommandHook  func(cmd Command, itx *CommandInteraction)      // Function that runs after each command.
	ComponentHandler func(itx *ComponentInteraction)                 // Function that runs for each unhandled component.
	ModalHandler     func(itx *ModalInteraction)                     // Function that runs for each unhandled modal.
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

	contexts := []InteractionContextType{0}
	if opt.DefaultInteractionContexts != nil || len(opt.DefaultInteractionContexts) > 0 {
		contexts = opt.DefaultInteractionContexts
	}

	return Client{
		ApplicationID:      botUserID,
		PublicKey:          discordPublicKey,
		Rest:               NewRest(opt.Token),
		commands:           NewSharedMap[string, Command](),
		commandContexts:    contexts,
		staticComponents:   NewSharedMap[string, func(ComponentInteraction)](),
		staticModals:       NewSharedMap[string, func(ModalInteraction)](),
		preCommandHandler:  opt.PreCommandHook,
		postCommandHandler: opt.PostCommandHook,
		componentHandler:   opt.ComponentHandler,
		modalHandler:       opt.ModalHandler,
		queuedComponents:   NewSharedMap[string, chan *ComponentInteraction](),
		queuedModals:       NewSharedMap[string, chan *ModalInteraction](),
	}
}

// Makes client dynamically "listen" incoming component type interactions.
// When component custom id matches - it'll send back interaction through channel.
// Holder s responsible for calling cleanup function once done (check example app code for better understanding).
// You can use context to control timeout - Discord API allows to reply to interaction for max 15 minutes.
//
// Warning! Components handled this way will already be acknowledged.
func (client *Client) AwaitComponent(customIDs []string) (<-chan *ComponentInteraction, func(), error) {
	client.staticComponents.mu.RLock()
	for _, id := range customIDs {
		if client.staticComponents.cache[id] != nil {
			client.staticComponents.mu.RUnlock()
			return nil, nil, fmt.Errorf("static component with custom ID \"%s\" is already registered", id)
		}
	}
	client.staticComponents.mu.RUnlock()

	client.queuedComponents.mu.RLock()
	for _, id := range customIDs {
		if client.queuedComponents.cache[id] != nil {
			client.queuedComponents.mu.RUnlock()
			return nil, nil, fmt.Errorf("dynamic component with custom ID \"%s\" is already registered", id)
		}
	}
	client.queuedComponents.mu.RUnlock()

	signalChan := make(chan *ComponentInteraction)
	var once sync.Once

	cleanup := func() {
		once.Do(func() {
			client.queuedComponents.mu.Lock()
			for _, id := range customIDs {
				delete(client.queuedComponents.cache, id)
			}
			client.queuedComponents.mu.Unlock()
			close(signalChan)
		})
	}

	client.queuedComponents.mu.Lock()
	for _, id := range customIDs {
		client.queuedComponents.cache[id] = signalChan
	}
	client.queuedComponents.mu.Unlock()

	return signalChan, cleanup, nil
}

// Mirror method to Client.AwaitComponent but for handling modal interactions.
// Look comment on Client.AwaitComponent and see example bot/app code for more.
func (client *Client) AwaitModal(customIDs []string) (<-chan *ModalInteraction, func(), error) {
	client.staticModals.mu.RLock()
	for _, id := range customIDs {
		if client.staticModals.cache[id] != nil {
			client.staticModals.mu.RUnlock()
			return nil, nil, fmt.Errorf("static modal with custom ID \"%s\" is already registered", id)
		}
	}
	client.staticModals.mu.RUnlock()

	client.queuedModals.mu.RLock()
	for _, id := range customIDs {
		if client.queuedModals.cache[id] != nil {
			client.queuedModals.mu.RUnlock()
			return nil, nil, fmt.Errorf("dynamic modal with custom ID \"%s\" is already registered", id)
		}
	}
	client.queuedModals.mu.RUnlock()

	signalChan := make(chan *ModalInteraction)
	var once sync.Once

	cleanup := func() {
		once.Do(func() {
			client.queuedModals.mu.Lock()
			for _, id := range customIDs {
				delete(client.queuedModals.cache, id)
			}
			client.queuedModals.mu.Unlock()
			close(signalChan)
		})
	}

	client.queuedModals.mu.Lock()
	for _, id := range customIDs {
		client.queuedModals.cache[id] = signalChan
	}
	client.queuedModals.mu.Unlock()

	return signalChan, cleanup, nil
}

// Pings Discord API and returns time it took to get response.
func (client *Client) Ping() time.Duration {
	start := time.Now()
	client.Rest.Request(http.MethodGet, "/gateway", nil)
	return time.Since(start)
}

func (client *Client) SendMessage(channelID Snowflake, message Message, files []File) (Message, error) {
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
func (client *Client) SendPrivateMessage(userID Snowflake, content Message, files []File) (Message, error) {
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

// Returns all entitlements for a given app, active and expired.
//
// By default it will attempt to return all, existing entitlements - provide query filter to control this behavior.
//
// https://discord.com/developers/docs/resources/entitlement#list-entitlements
func (client *Client) FetchEntitlementsPage(queryFilter string) ([]Entitlement, error) {
	if queryFilter[0] != '?' {
		queryFilter = "?" + queryFilter
	}

	res := make([]Entitlement, 0)
	raw, err := client.Rest.Request(http.MethodGet, "/applications/"+client.ApplicationID.String()+"/entitlements"+queryFilter, nil)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(raw, &res)
	if err != nil {
		return res, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// https://discord.com/developers/docs/resources/entitlement#get-entitlement
func (client *Client) FetchEntitlement(entitlementID Snowflake) (Entitlement, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil)
	if err != nil {
		return Entitlement{}, err
	}

	res := Entitlement{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Entitlement{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// For One-Time Purchase consumable SKUs, marks a given entitlement for the user as consumed.
// The entitlement will have consumed: true when using Client.FetchEntitlements.
//
// https://discord.com/developers/docs/resources/entitlement#consume-an-entitlement
func (client *Client) ConsumeEntitlement(entitlementID Snowflake) error {
	_, err := client.Rest.Request(http.MethodPost, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String()+"/consume", nil)
	return err
}

// https://discord.com/developers/docs/resources/entitlement#create-test-entitlement
func (client *Client) CreateTestEntitlement(payload TestEntitlementPayload) error {
	_, err := client.Rest.Request(http.MethodPost, "/applications/"+client.ApplicationID.String()+"/entitlements", payload)
	return err
}

// https://discord.com/developers/docs/resources/entitlement#delete-test-entitlement
func (client *Client) DeleteTestEntitlement(entitlementID Snowflake) error {
	_, err := client.Rest.Request(http.MethodDelete, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil)
	return err
}
