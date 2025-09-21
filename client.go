package qord

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"qord/api"
	"qord/gateway"
	"qord/other"
	"qord/rest"
)

// A building bot for final Gateway or HTTPS Clients.
// It contains all shared fields & methods.
// There's probably no reason for anyone to ever use it - use GatewayClient or WebhookClient instead.
type Client struct {
	ApplicationID      api.Snowflake
	ItxManager         *other.InteractionManager
	Gateway            *gateway.ShardManager
	Rest               *rest.RestManager
	traceLogger        *log.Logger
	customEventHandler func(shardID uint16, packet gateway.EventPacket) // Provided from outside in ClientOptions
}

type ClientOptions struct {
	other.InteractionManagerOptions
	Token string
	// Client has own event handler for dealing with interactions but you can
	// still attach your own logic. It will be used before Client's default handler.
	EventHandler func(shardID uint16, packet gateway.EventPacket)
	Trace        bool // Whether to have ShardManager print debug logs.
}

func NewClient(opt ClientOptions) *Client {
	botUserID, err := other.ExtractUserIDFromToken(opt.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	opt.ApplicationID = botUserID
	self := Client{
		ApplicationID: botUserID,
		ItxManager:    other.NewInteractionManager(opt.InteractionManagerOptions),
		Rest:          rest.NewRestManager(opt.Token),
		traceLogger:   log.New(io.Discard, "[QORD] ", log.LstdFlags),
	}
	self.Gateway = gateway.NewShardManager(opt.Token, opt.Trace, self.eventHandler)

	if opt.Trace {
		self.traceLogger.SetOutput(os.Stdout)
		self.tracef("Main client tracing enabled.")
	}
	return &self
}

func (client *Client) BulkSyncCommandsWithDiscord(guildIDs []api.Snowflake, whitelist []string, reverseMode bool) error {
	raw, err := client.ItxManager.ExtractCommandDataForDiscordAPI(guildIDs, whitelist, reverseMode)
	if err != nil {
		return err
	}

	if len(guildIDs) == 0 {
		_, err := client.Rest.Request(http.MethodPut, "/applications/"+client.ApplicationID.String()+"/commands", raw)
		return err
	}

	for _, guildID := range guildIDs {
		_, err := client.Rest.Request(http.MethodPut, "/applications/"+client.ApplicationID.String()+"/guilds/"+guildID.String()+"/commands", raw)
		if err != nil {
			return err
		}
	}

	return nil
}

func (client *Client) SendMessage(channelID api.Snowflake, message api.Message, files []rest.File) (api.Message, error) {
	raw, err := client.Rest.RequestWithFiles(http.MethodPost, "/channels/"+channelID.String()+"/messages", message, files)
	if err != nil {
		return api.Message{}, err
	}

	res := api.Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return api.Message{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client *Client) SendLinearMessage(channelID api.Snowflake, content string) (api.Message, error) {
	return client.SendMessage(channelID, api.Message{Content: content}, nil)
}

// Creates (or fetches if already exists) user's private text channel (DM) and tries to send message into it.
// Warning! Discord's user channels endpoint has huge rate limits so please reuse api.Message#ChannelID whenever possible.
func (client *Client) SendPrivateMessage(userID api.Snowflake, content api.Message, files []rest.File) (api.Message, error) {
	res := make(map[string]interface{}, 0)
	res["recipient_id"] = userID

	raw, err := client.Rest.Request(http.MethodPost, "/users/@me/channels", res)
	if err != nil {
		return api.Message{}, err
	}

	err = json.Unmarshal(raw, &res)
	if err != nil {
		return api.Message{}, errors.New("failed to parse received data from discord")
	}

	channelID, err := api.StringToSnowflake(res["id"].(string))
	if err != nil {
		return api.Message{}, err
	}

	msg, err := client.SendMessage(channelID, content, files)
	msg.ChannelID = channelID // Just in case.

	return msg, err
}

func (client *Client) EditMessage(channelID api.Snowflake, messageID api.Snowflake, content api.Message) error {
	_, err := client.Rest.Request(http.MethodPatch, "/channels/"+channelID.String()+"/messages/"+messageID.String(), content)
	return err
}

func (client *Client) DeleteMessage(channelID api.Snowflake, messageID api.Snowflake) error {
	_, err := client.Rest.Request(http.MethodDelete, "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	return err
}

func (client *Client) CrosspostMessage(channelID api.Snowflake, messageID api.Snowflake) error {
	_, err := client.Rest.Request(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/crosspost", nil)
	return err
}

func (client *Client) FetchUser(id api.Snowflake) (api.User, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/users/"+id.String(), nil)
	if err != nil {
		return api.User{}, err
	}

	res := api.User{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return api.User{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

func (client *Client) FetchMember(guildID api.Snowflake, memberID api.Snowflake) (api.Member, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/guilds/"+guildID.String()+"/members/"+memberID.String(), nil)
	if err != nil {
		return api.Member{}, err
	}

	res := api.Member{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return api.Member{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// Returns all entitlements for a given app, active and expired.
//
// By default it will attempt to return all, existing entitlements - provide query filter to control this behavior.
//
// https://discord.com/developers/docs/resources/entitlement#list-entitlements
func (client *Client) FetchEntitlementsPage(queryFilter string) ([]api.Entitlement, error) {
	if queryFilter[0] != '?' {
		queryFilter = "?" + queryFilter
	}

	res := make([]api.Entitlement, 0)
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
func (client *Client) FetchEntitlement(entitlementID api.Snowflake) (api.Entitlement, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil)
	if err != nil {
		return api.Entitlement{}, err
	}

	res := api.Entitlement{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return api.Entitlement{}, errors.New("failed to parse received data from discord")
	}

	return res, nil
}

// For One-Time Purchase consumable SKUs, marks a given entitlement for the user as consumed.
// The entitlement will have consumed: true when using BaseClient.FetchEntitlements.
//
// https://discord.com/developers/docs/resources/entitlement#consume-an-entitlement
func (client *Client) ConsumeEntitlement(entitlementID api.Snowflake) error {
	_, err := client.Rest.Request(http.MethodPost, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String()+"/consume", nil)
	return err
}

// https://discord.com/developers/docs/resources/entitlement#create-test-entitlement
func (client *Client) CreateTestEntitlement(payload api.TestEntitlementPayload) error {
	_, err := client.Rest.Request(http.MethodPost, "/applications/"+client.ApplicationID.String()+"/entitlements", payload)
	return err
}

// https://discord.com/developers/docs/resources/entitlement#delete-test-entitlement
func (client *Client) DeleteTestEntitlement(entitlementID api.Snowflake) error {
	_, err := client.Rest.Request(http.MethodDelete, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil)
	return err
}

func (client *Client) tracef(format string, v ...interface{}) {
	client.traceLogger.Printf("[CLIENT] "+format, v...)
}
