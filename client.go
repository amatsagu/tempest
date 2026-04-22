package tempest

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

// BaseClient is the core tempest entrypoint. It's used to create either HTTP or Gateway clients.
// You should avoid using base version unless you know what you're doing.
type BaseClient struct {
	ApplicationID Snowflake
	Rest          *Rest

	traceLogger *log.Logger // Inherited from HTTPClient or GatewayClient

	commands         *SharedMap[string, Command]
	commandContexts  []InteractionContextType
	staticComponents *SharedMap[string, func(ComponentInteraction)]
	staticModals     *SharedMap[string, func(ModalInteraction)]

	preCommandHandler  func(cmd Command, itx *CommandInteraction) bool
	postCommandHandler func(cmd Command, itx *CommandInteraction)
	componentHandler   func(itx *ComponentInteraction)
	modalHandler       func(itx *ModalInteraction)

	queuedComponents *SharedMap[string, *queuedComponent]
	queuedModals     *SharedMap[string, *queuedModal]

	sweeper interactionSweeper
}

type BaseClientOptions struct {
	Token                      string
	DefaultInteractionContexts []InteractionContextType
	RestOptions                RestOptions

	PreCommandHook   func(cmd Command, itx *CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	PostCommandHook  func(cmd Command, itx *CommandInteraction)      // Function that runs after each command.
	ComponentHandler func(itx *ComponentInteraction)                 // Function that runs for each unhandled component.
	ModalHandler     func(itx *ModalInteraction)                     // Function that runs for each unhandled modal.

	Logger *log.Logger // Optional custom logger. If tracing is enabled, this logger will be used for all internal messages. If none is provided, the default Stdout logger will be used instead.
}

func NewBaseClient(opt BaseClientOptions) *BaseClient {
	botUserID, err := extractUserIDFromToken(opt.Token)
	if err != nil {
		panic("failed to extract bot user ID from bot token: " + err.Error())
	}

	contexts := []InteractionContextType{0}
	if opt.DefaultInteractionContexts != nil || len(opt.DefaultInteractionContexts) > 0 {
		contexts = opt.DefaultInteractionContexts
	}

	traceLogger := opt.Logger
	if traceLogger == nil {
		traceLogger = log.New(io.Discard, "[TEMPEST] ", log.LstdFlags)
	}

	if opt.RestOptions.Token == "" {
		opt.RestOptions.Token = opt.Token
	}

	client := &BaseClient{
		ApplicationID:      botUserID,
		Rest:               NewRest(opt.RestOptions),
		traceLogger:        traceLogger,
		commands:           NewSharedMap[string, Command](),
		commandContexts:    contexts,
		staticComponents:   NewSharedMap[string, func(ComponentInteraction)](),
		staticModals:       NewSharedMap[string, func(ModalInteraction)](),
		preCommandHandler:  opt.PreCommandHook,
		postCommandHandler: opt.PostCommandHook,
		componentHandler:   opt.ComponentHandler,
		modalHandler:       opt.ModalHandler,
		queuedComponents:   NewSharedMap[string, *queuedComponent](),
		queuedModals:       NewSharedMap[string, *queuedModal](),
		sweeper: interactionSweeper{
			signal: make(chan struct{}, 1),
		},
	}

	return client
}

func (s *BaseClient) tracef(format string, v ...any) {
	s.traceLogger.Printf("[(BASE) CLIENT] "+format, v...)
}

func (client *BaseClient) SendMessage(channelID Snowflake, message Message, files []File) (Message, error) {
	raw, err := client.Rest.RequestWithFiles(http.MethodPost, "/channels/"+channelID.String()+"/messages", message, files)
	if err != nil {
		return Message{}, err
	}

	res := Message{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Message{}, errors.New("failed to parse received data from discord")
	}

	client.tracef("Successfully sent message ID = %d to channel ID = %d.", res.ID, channelID)
	return res, nil
}

func (client *BaseClient) SendLinearMessage(channelID Snowflake, content string) (Message, error) {
	return client.SendMessage(channelID, Message{Content: content}, nil)
}

// Creates (or fetches if already exists) user's private text channel (DM) and tries to send message into it.
// Warning! Discord's user channels endpoint has huge rate limits so please reuse Message#ChannelID whenever possible.
func (client *BaseClient) SendPrivateMessage(userID Snowflake, content Message, files []File) (Message, error) {
	res := make(map[string]any, 0)
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

func (client *BaseClient) EditMessage(channelID Snowflake, messageID Snowflake, content Message) error {
	_, err := client.Rest.Request(http.MethodPatch, "/channels/"+channelID.String()+"/messages/"+messageID.String(), content)
	if err == nil {
		client.tracef("Successfully edited message ID = %d to channel ID = %d.", messageID, channelID)
	}
	return err
}

func (client *BaseClient) DeleteMessage(channelID Snowflake, messageID Snowflake) error {
	_, err := client.Rest.Request(http.MethodDelete, "/channels/"+channelID.String()+"/messages/"+messageID.String(), nil)
	if err == nil {
		client.tracef("Successfully deleted message ID = %d to channel ID = %d.", messageID, channelID)
	}
	return err
}

func (client *BaseClient) CrosspostMessage(channelID Snowflake, messageID Snowflake) error {
	_, err := client.Rest.Request(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/crosspost", nil)
	if err == nil {
		client.tracef("Successfully crossposted message ID = %d to channel ID = %d.", messageID, channelID)
	}
	return err
}

func (client *BaseClient) FetchUser(id Snowflake) (User, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/users/"+id.String(), nil)
	if err != nil {
		return User{}, err
	}

	res := User{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return User{}, errors.New("failed to parse received data from discord")
	}

	client.tracef("Successfully fetched \"%s\" (ID = %d) user data.", res.GlobalName, res.ID)
	return res, nil
}

func (client *BaseClient) FetchMember(guildID Snowflake, memberID Snowflake) (Member, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/guilds/"+guildID.String()+"/members/"+memberID.String(), nil)
	if err != nil {
		return Member{}, err
	}

	res := Member{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Member{}, errors.New("failed to parse received data from discord")
	}

	client.tracef("Successfully fetched \"%s\" (ID = %d) member data.", res.User.GlobalName, res.User.ID)
	return res, nil
}

// Returns all entitlements for a given app, active and expired.
//
// By default it will attempt to return all, existing entitlements - provide query filter to control this behavior.
//
// https://docs.discord.com/developers/resources/entitlement#list-entitlements
func (client *BaseClient) FetchEntitlementsPage(queryFilter string) ([]Entitlement, error) {
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

	client.tracef("Successfully fetched %d entitlement(s).", len(res))
	return res, nil
}

// https://docs.discord.com/developers/resources/entitlement#get-entitlement
func (client *BaseClient) FetchEntitlement(entitlementID Snowflake) (Entitlement, error) {
	raw, err := client.Rest.Request(http.MethodGet, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil)
	if err != nil {
		return Entitlement{}, err
	}

	res := Entitlement{}
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return Entitlement{}, errors.New("failed to parse received data from discord")
	}

	client.tracef("Successfully fetched entitlement with ID = %d.", entitlementID)
	return res, nil
}

// For One-Time Purchase consumable SKUs, marks a given entitlement for the user as consumed.
// The entitlement will have consumed: true when using Client.FetchEntitlements.
//
// https://docs.discord.com/developers/resources/entitlement#consume-an-entitlement
func (client *BaseClient) ConsumeEntitlement(entitlementID Snowflake) error {
	_, err := client.Rest.Request(http.MethodPost, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String()+"/consume", nil)
	if err != nil {
		client.tracef("Successfully consumed entitlement with ID = %d.", entitlementID)
	}
	return err
}

// https://docs.discord.com/developers/resources/entitlement#create-test-entitlement
func (client *BaseClient) CreateTestEntitlement(payload TestEntitlementPayload) error {
	_, err := client.Rest.Request(http.MethodPost, "/applications/"+client.ApplicationID.String()+"/entitlements", payload)
	if err != nil {
		client.tracef("Successfully created test entitlement.")
	}
	return err
}

// https://docs.discord.com/developers/resources/entitlement#delete-test-entitlement
func (client *BaseClient) DeleteTestEntitlement(entitlementID Snowflake) error {
	_, err := client.Rest.Request(http.MethodDelete, "/applications/"+client.ApplicationID.String()+"/entitlements/"+entitlementID.String(), nil)
	if err != nil {
		client.tracef("Successfully deleted test entitlement.")
	}
	return err
}

func (client *BaseClient) SyncCommandsWithDiscord(guildIDs []Snowflake, whitelist []string, reverseMode bool) error {
	commands := parseCommandsForDiscordAPI(client.commands, whitelist, reverseMode)

	if len(guildIDs) == 0 {
		_, err := client.Rest.Request(http.MethodPut, "/applications/"+client.ApplicationID.String()+"/commands", commands)
		return err
	}

	for _, guildID := range guildIDs {
		_, err := client.Rest.Request(http.MethodPut, "/applications/"+client.ApplicationID.String()+"/guilds/"+guildID.String()+"/commands", commands)
		if err != nil {
			return err
		}
	}

	client.tracef("Successfully synced command data with discord.")
	return nil
}

func (client *BaseClient) handleInteraction(itx CommandInteraction) (CommandInteraction, Command, bool) {
	if len(itx.Data.Options) > 0 && itx.Data.Options[0].Type == SUB_COMMAND_OPTION_TYPE {
		finalName := itx.Data.Name + "@" + itx.Data.Options[0].Name
		subCommand, available := client.commands.Get(finalName)
		if available {
			if itx.Member != nil {
				itx.Member.GuildID = itx.GuildID
			}

			itx.Data.Name, itx.Data.Options = finalName, itx.Data.Options[0].Options
		}
		return itx, subCommand, available
	}

	if itx.Member != nil {
		itx.Member.GuildID = itx.GuildID
	}

	command, available := client.commands.Get(itx.Data.Name)
	return itx, command, available
}

func parseCommandsForDiscordAPI(commands *SharedMap[string, Command], whitelist []string, reverseMode bool) []Command {
	commands.mu.RLock()

	tree := make(map[string]map[string]Command, len(commands.cache))
	parsedCommands := make([]Command, 0, len(commands.cache))

	// First loop - prepare nested space for potential sub commands
	for name, command := range commands.cache {
		if strings.Contains(name, "@") {
			continue
		}

		group := make(map[string]Command, 0)
		group[ROOT_PLACEHOLDER] = command
		tree[name] = group
	}

	// Second loop - assign commands
	for name, command := range commands.cache {
		if strings.Contains(name, "@") {
			parts := strings.Split(name, "@")
			group := tree[parts[0]]

			command.Type = CommandType(SUB_COMMAND_OPTION_TYPE)
			group[parts[1]] = command
			tree[parts[0]] = group
		}
	}

	commands.mu.RUnlock()

	// Use nested map to build final array with structs matching Discord API
	for _, branch := range tree {
		baseCommand := branch[ROOT_PLACEHOLDER]

		if len(branch) > 1 {
			for key, subCommand := range branch {
				if key == ROOT_PLACEHOLDER {
					continue
				}

				baseCommand.Options = append(baseCommand.Options, CommandOption{
					Name:        subCommand.Name,
					Description: subCommand.Description,
					Type:        SUB_COMMAND_OPTION_TYPE,
					Options:     subCommand.Options,
				})
			}
		}

		parsedCommands = append(parsedCommands, baseCommand)
	}

	if len(whitelist) == 0 {
		return parsedCommands
	}

	// Build map for fast lookup
	filterMap := make(map[string]struct{}, len(whitelist))
	for _, name := range whitelist {
		filterMap[name] = struct{}{}
	}

	var filtered []Command

	if reverseMode {
		// BLACKLIST: exclude listed commands
		for _, cmd := range parsedCommands {
			if _, blocked := filterMap[cmd.Name]; blocked {
				continue
			}
			filtered = append(filtered, cmd)
		}
	} else {
		// WHITELIST: include only listed commands
		for _, cmd := range parsedCommands {
			if _, allowed := filterMap[cmd.Name]; allowed {
				filtered = append(filtered, cmd)
			}
		}
	}

	return filtered
}
