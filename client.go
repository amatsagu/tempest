package tempest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
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

type queuedComponent struct {
	Handler   func(*ComponentInteraction)
	Expire    time.Time
	OnTimeout func()
}

type queuedModal struct {
	Handler   func(*ModalInteraction)
	Expire    time.Time
	OnTimeout func()
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
	}

	return client
}

func (s *BaseClient) tracef(format string, v ...any) {
	s.traceLogger.Printf("[(BASE) CLIENT] "+format, v...)
}

// Makes client dynamically "listen" incoming component type interactions.
// When component custom id matches - it'll run onAction callback.
// If onAction returns false, it stops listening.
// If it reaches the timeout duration, it stops listening and calls onTimeout (if provided).
//
// Warning! Components handled this way will already be acknowledged.
func (client *BaseClient) AwaitComponent(customIDs []string, timeout time.Duration, onAction func(itx *ComponentInteraction) bool, onTimeout func()) error {
	client.staticComponents.mu.RLock()
	client.queuedComponents.mu.Lock()
	defer client.staticComponents.mu.RUnlock()
	defer client.queuedComponents.mu.Unlock()

	for _, id := range customIDs {
		if _, ok := client.staticComponents.cache[id]; ok {
			return fmt.Errorf("static component with custom ID %q is already registered", id)
		}

		if _, ok := client.queuedComponents.cache[id]; ok {
			return fmt.Errorf("dynamic component with custom ID %q is already registered", id)
		}
	}

	handler := func(itx *ComponentInteraction) {
		keepListening := onAction(itx)
		if !keepListening {
			client.queuedComponents.mu.Lock()
			for _, id := range customIDs {
				delete(client.queuedComponents.cache, id)
			}
			client.queuedComponents.mu.Unlock()
		}
	}

	expire := time.Now().Add(timeout)

	var once sync.Once
	var timeoutFunc func() = nil
	if onTimeout != nil {
		timeoutFunc = func() {
			once.Do(func() {
				client.queuedComponents.mu.Lock()
				for _, id := range customIDs {
					delete(client.queuedComponents.cache, id)
				}
				client.queuedComponents.mu.Unlock()
				onTimeout()
			})
		}
	} else {
		timeoutFunc = func() {
			once.Do(func() {
				client.queuedComponents.mu.Lock()
				for _, id := range customIDs {
					delete(client.queuedComponents.cache, id)
				}
				client.queuedComponents.mu.Unlock()
			})
		}
	}

	for _, id := range customIDs {
		client.queuedComponents.cache[id] = &queuedComponent{
			Handler:   handler,
			Expire:    expire,
			OnTimeout: timeoutFunc,
		}
	}

	client.tracef("Registered dynamic component(s) IDs = %+v", customIDs)
	client.sweeper.tryRun(client)

	return nil
}

// Mirror method to Client.AwaitComponent but for handling modal interactions.
// Look comment on Client.AwaitComponent and see example bot/app code for more.
func (client *BaseClient) AwaitModal(customIDs []string, timeout time.Duration, onAction func(itx *ModalInteraction) bool, onTimeout func()) error {
	client.staticModals.mu.RLock()
	client.queuedModals.mu.Lock()
	defer client.staticModals.mu.RUnlock()
	defer client.queuedModals.mu.Unlock()

	for _, id := range customIDs {
		if _, ok := client.staticModals.cache[id]; ok {
			return fmt.Errorf("static modal with custom ID %q is already registered", id)
		}

		if _, ok := client.queuedModals.cache[id]; ok {
			return fmt.Errorf("dynamic modal with custom ID %q is already registered", id)
		}
	}

	handler := func(itx *ModalInteraction) {
		keepListening := onAction(itx)
		if !keepListening {
			client.queuedModals.mu.Lock()
			for _, id := range customIDs {
				delete(client.queuedModals.cache, id)
			}
			client.queuedModals.mu.Unlock()
		}
	}

	expire := time.Now().Add(timeout)

	var once sync.Once
	var timeoutFunc func() = nil
	if onTimeout != nil {
		timeoutFunc = func() {
			once.Do(func() {
				client.queuedModals.mu.Lock()
				for _, id := range customIDs {
					delete(client.queuedModals.cache, id)
				}
				client.queuedModals.mu.Unlock()
				onTimeout()
			})
		}
	} else {
		timeoutFunc = func() {
			once.Do(func() {
				client.queuedModals.mu.Lock()
				for _, id := range customIDs {
					delete(client.queuedModals.cache, id)
				}
				client.queuedModals.mu.Unlock()
			})
		}
	}

	for _, id := range customIDs {
		client.queuedModals.cache[id] = &queuedModal{
			Handler:   handler,
			Expire:    expire,
			OnTimeout: timeoutFunc,
		}
	}

	client.tracef("Registered dynamic modal(s) IDs = %+v", customIDs)
	client.sweeper.tryRun(client)

	return nil
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

func (client *BaseClient) RegisterCommand(cmd Command) error {
	if client.commands.Has(cmd.Name) {
		return errors.New("client already has registered \"" + cmd.Name + "\" slash command (name already in use)")
	}

	if cmd.Type == 0 {
		cmd.Type = CHAT_INPUT_COMMAND_TYPE
	}

	if cmd.ApplicationID == 0 {
		cmd.ApplicationID = client.ApplicationID
	}

	if len(cmd.Contexts) == 0 {
		cmd.Contexts = client.commandContexts
	}

	client.commands.Set(cmd.Name, cmd)
	client.tracef("Registered %s command.", cmd.Name)
	return nil
}

func (client *BaseClient) RegisterSubCommand(subCommand Command, parentCommandName string) error {
	if !client.commands.Has(parentCommandName) {
		return errors.New("missing \"" + parentCommandName + "\" slash command in registry (parent command needs to be registered in client before adding subcommands)")
	}

	finalName := parentCommandName + "@" + subCommand.Name
	if client.commands.Has(finalName) {
		return errors.New("client already has registered \"" + finalName + "\" slash command (name for subcommand is already in use)")
	}

	if subCommand.Type == 0 {
		subCommand.Type = CHAT_INPUT_COMMAND_TYPE
	}

	if subCommand.ApplicationID == 0 {
		subCommand.ApplicationID = client.ApplicationID
	}

	if len(subCommand.Contexts) == 0 {
		subCommand.Contexts = client.commandContexts
	}

	client.commands.Set(finalName, subCommand)
	client.tracef("Registered %s sub command (part of %s command).", finalName, parentCommandName)
	return nil
}

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (client *BaseClient) RegisterComponent(customIDs []string, fn func(ComponentInteraction)) error {
	client.staticComponents.mu.Lock()
	client.queuedComponents.mu.RLock()
	defer client.staticComponents.mu.Unlock()
	defer client.queuedComponents.mu.RUnlock()

	for _, id := range customIDs {
		if _, ok := client.staticComponents.cache[id]; ok {
			return fmt.Errorf("client already has registered static component with custom ID %q (custom id already in use)", id)
		}

		if _, ok := client.queuedComponents.cache[id]; ok {
			return fmt.Errorf("client already has registered dynamic (queued) component with custom ID %q (custom id already in use elsewhere)", id)
		}
	}

	for _, key := range customIDs {
		client.staticComponents.cache[key] = fn
	}

	client.tracef("Registered static component handler for custom IDs = %+v", customIDs)
	return nil
}

// Bind function to modal with matching custom id. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (client *BaseClient) RegisterModal(customID string, fn func(ModalInteraction)) error {
	client.staticModals.mu.Lock()
	client.queuedModals.mu.RLock()
	defer client.staticModals.mu.Unlock()
	defer client.queuedModals.mu.RUnlock()

	if _, ok := client.staticModals.cache[customID]; ok {
		return fmt.Errorf("client already has registered static modal with custom ID %q (custom id already in use)", customID)
	}

	if _, ok := client.queuedModals.cache[customID]; ok {
		return fmt.Errorf("client already has registered dynamic (queued) modal with custom ID %q (custom id already in use elsewhere)", customID)
	}

	client.staticModals.cache[customID] = fn
	client.tracef("Registered static modal handler for custom ID = %s", customID)
	return nil
}

// Removes previously registered, static components that match any of provided custom IDs.
func (client *BaseClient) DeleteComponent(customIDs []string) error {
	client.staticComponents.mu.Lock()
	client.queuedComponents.mu.RLock()
	defer client.staticComponents.mu.Unlock()
	defer client.queuedComponents.mu.RUnlock()

	for _, id := range customIDs {
		if _, ok := client.queuedComponents.cache[id]; ok {
			return fmt.Errorf("client already has registered dynamic (queued) component with custom ID %q (custom id already in use elsewhere)", id)
		}
	}

	for _, key := range customIDs {
		delete(client.staticComponents.cache, key)
	}

	client.tracef("Removed static component handler for custom IDs = %+v", customIDs)
	return nil
}

// Removes previously registered, static modals that match any of provided custom IDs.
func (client *BaseClient) DeleteModal(customIDs []string) error {
	client.staticModals.mu.Lock()
	client.queuedModals.mu.RLock()
	defer client.staticModals.mu.Unlock()
	defer client.queuedModals.mu.RUnlock()

	for _, id := range customIDs {
		if _, ok := client.queuedModals.cache[id]; ok {
			return fmt.Errorf("client already has registered dynamic (queued) modal with custom ID %q (custom id already in use elsewhere)", id)
		}
	}

	for _, key := range customIDs {
		delete(client.staticModals.cache, key)
	}

	client.tracef("Removed static modal handler for custom IDs = %+v", customIDs)
	return nil
}

func (client *BaseClient) FindCommand(cmdName string) (Command, bool) {
	return client.commands.Get(cmdName)
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
