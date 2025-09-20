package other

import (
	"encoding/json"
	"errors"
	"fmt"
	"qord/api"
	"strings"
	"sync"
)

// InteractionManager is controlling anything related to command interactions.
// Use it to store, seek, sync, use slash commands data.
type InteractionManager struct {
	ApplicationID      api.Snowflake
	Commands           *SharedMap[string, api.Command]
	commandContexts    []api.InteractionContextType
	Components         *SharedMap[string, func(api.ComponentInteraction)]
	Modals             *SharedMap[string, func(api.ModalInteraction)]
	PreCommandHandler  func(cmd api.Command, itx *api.CommandInteraction) bool
	PostCommandHandler func(cmd api.Command, itx *api.CommandInteraction)
	ComponentHandler   func(itx *api.ComponentInteraction)
	ModalHandler       func(itx *api.ModalInteraction)

	QueuedComponents *SharedMap[string, chan *api.ComponentInteraction]
	QueuedModals     *SharedMap[string, chan *api.ModalInteraction]
}

type InteractionManagerOptions struct {
	ApplicationID              api.Snowflake
	DefaultInteractionContexts []api.InteractionContextType
	PreCommandHook             func(cmd api.Command, itx *api.CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	PostCommandHook            func(cmd api.Command, itx *api.CommandInteraction)      // Function that runs after each command.
	ComponentHandler           func(itx *api.ComponentInteraction)                     // Function that runs for each unhandled component.
	ModalHandler               func(itx *api.ModalInteraction)                         // Function that runs for each unhandled modal.
}

func NewInteractionManager(opt InteractionManagerOptions) *InteractionManager {
	if opt.ApplicationID == 0 {
		panic("interaction manager requires app/bot ID")
	}

	if len(opt.DefaultInteractionContexts) == 0 {
		opt.DefaultInteractionContexts = []api.InteractionContextType{
			api.GUILD_CONTEXT_TYPE,
		}
	}

	return &InteractionManager{
		ApplicationID:      opt.ApplicationID,
		Commands:           NewSharedMap[string, api.Command](),
		commandContexts:    opt.DefaultInteractionContexts,
		Components:         NewSharedMap[string, func(api.ComponentInteraction)](),
		Modals:             NewSharedMap[string, func(api.ModalInteraction)](),
		PreCommandHandler:  opt.PreCommandHook,
		PostCommandHandler: opt.PostCommandHook,
		ComponentHandler:   opt.ComponentHandler,
		ModalHandler:       opt.ModalHandler,
		QueuedComponents:   NewSharedMap[string, chan *api.ComponentInteraction](),
		QueuedModals:       NewSharedMap[string, chan *api.ModalInteraction](),
	}
}

// Makes client dynamically "listen" incoming component type interactions.
// When component custom id matches - it'll send back interaction through channel.
// Holder s responsible for calling cleanup function once done (check example app code for better understanding).
// You can use context to control timeout - Discord API allows to reply to interaction for max 15 minutes.
//
// Warning! Components handled this way will already be acknowledged.
func (m *InteractionManager) AwaitComponent(customIDs []string) (<-chan *api.ComponentInteraction, func(), error) {
	m.Components.mu.RLock()
	for _, id := range customIDs {
		if m.Components.cache[id] != nil {
			m.Components.mu.RUnlock()
			return nil, nil, fmt.Errorf("static component with custom ID \"%s\" is already registered", id)
		}
	}
	m.Components.mu.RUnlock()

	m.QueuedComponents.mu.RLock()
	for _, id := range customIDs {
		if m.QueuedComponents.cache[id] != nil {
			m.QueuedComponents.mu.RUnlock()
			return nil, nil, fmt.Errorf("dynamic component with custom ID \"%s\" is already registered", id)
		}
	}
	m.QueuedComponents.mu.RUnlock()

	signalChan := make(chan *api.ComponentInteraction)
	var once sync.Once

	cleanup := func() {
		once.Do(func() {
			m.QueuedComponents.mu.Lock()
			for _, id := range customIDs {
				delete(m.QueuedComponents.cache, id)
			}
			m.QueuedComponents.mu.Unlock()
			close(signalChan)
		})
	}

	m.QueuedComponents.mu.Lock()
	for _, id := range customIDs {
		m.QueuedComponents.cache[id] = signalChan
	}
	m.QueuedComponents.mu.Unlock()

	return signalChan, cleanup, nil
}

// Mirror method to Client.AwaitComponent but for handling modal interactions.
// Look comment on Client.AwaitComponent and see example bot/app code for more.
func (m *InteractionManager) AwaitModal(customIDs []string) (<-chan *api.ModalInteraction, func(), error) {
	m.Modals.mu.RLock()
	for _, id := range customIDs {
		if m.Modals.cache[id] != nil {
			m.Modals.mu.RUnlock()
			return nil, nil, fmt.Errorf("static modal with custom ID \"%s\" is already registered", id)
		}
	}
	m.Modals.mu.RUnlock()

	m.QueuedModals.mu.RLock()
	for _, id := range customIDs {
		if m.QueuedModals.cache[id] != nil {
			m.QueuedModals.mu.RUnlock()
			return nil, nil, fmt.Errorf("dynamic modal with custom ID \"%s\" is already registered", id)
		}
	}
	m.QueuedModals.mu.RUnlock()

	signalChan := make(chan *api.ModalInteraction)
	var once sync.Once

	cleanup := func() {
		once.Do(func() {
			m.QueuedModals.mu.Lock()
			for _, id := range customIDs {
				delete(m.QueuedModals.cache, id)
			}
			m.QueuedModals.mu.Unlock()
			close(signalChan)
		})
	}

	m.QueuedModals.mu.Lock()
	for _, id := range customIDs {
		m.QueuedModals.cache[id] = signalChan
	}
	m.QueuedModals.mu.Unlock()

	return signalChan, cleanup, nil
}

func (m *InteractionManager) RegisterCommand(cmd api.Command) error {
	if m.Commands.Has(cmd.Name) {
		return errors.New("client already has registered \"" + cmd.Name + "\" slash command (name already in use)")
	}

	if cmd.Type == 0 {
		cmd.Type = api.CHAT_INPUT_COMMAND_TYPE
	}

	if cmd.ApplicationID == 0 {
		cmd.ApplicationID = m.ApplicationID
	}

	if len(cmd.Contexts) == 0 {
		cmd.Contexts = m.commandContexts
	}

	m.Commands.Set(cmd.Name, cmd)
	return nil
}

func (m *InteractionManager) RegisterSubCommand(subCommand api.Command, parentCommandName string) error {
	if !m.Commands.Has(parentCommandName) {
		return errors.New("missing \"" + parentCommandName + "\" slash command in registry (parent command needs to be registered in client before adding subcommands)")
	}

	finalName := parentCommandName + "@" + subCommand.Name
	if m.Commands.Has(finalName) {
		return errors.New("client already has registered \"" + finalName + "\" slash command (name for subcommand is already in use)")
	}

	if subCommand.Type == 0 {
		subCommand.Type = api.CHAT_INPUT_COMMAND_TYPE
	}

	if subCommand.ApplicationID == 0 {
		subCommand.ApplicationID = m.ApplicationID
	}

	if len(subCommand.Contexts) == 0 {
		subCommand.Contexts = m.commandContexts
	}

	m.Commands.Set(finalName, subCommand)
	return nil
}

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (m *InteractionManager) RegisterComponent(customIDs []string, fn func(api.ComponentInteraction)) error {
	for _, ID := range customIDs {
		if m.Components.Has(ID) {
			return errors.New("client already has registered static component with custom ID = " + ID + " (custom id already in use)")
		}

		if m.QueuedComponents.Has(ID) {
			return errors.New("client already has registered dynamic (queued) component with custom ID = " + ID + " (custom id already in use)")
		}
	}

	m.QueuedComponents.mu.Lock()
	for _, key := range customIDs {
		m.Components.cache[key] = fn
	}
	m.QueuedComponents.mu.Unlock()

	return nil
}

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (m *InteractionManager) RegisterModal(customID string, fn func(api.ModalInteraction)) error {
	if m.Modals.Has(customID) {
		return errors.New("client already has registered static modal with custom ID = " + customID + " (custom id already in use)")
	}

	if m.QueuedModals.Has(customID) {
		return errors.New("client already has registered dynamic (queued) modal with custom ID = " + customID + " (custom id already in use)")
	}

	m.Modals.Set(customID, fn)
	return nil
}

// Parses currently added slash commands (and their sub-commands) and returns
// formatted slice of Discord Command objects that can be then used with
// Discord API (REST) or with other APIs like recently added Top.gg Commands.
func (m *InteractionManager) ExtractCommandDataForDiscordAPI(guildIDs []api.Snowflake, whitelist []string, reverseMode bool) ([]byte, error) {
	commands := parseCommandsForDiscordAPI(m.Commands, whitelist, reverseMode)
	return json.Marshal(commands)
}

// Advanced form of InteractionManager.Commands.Get().
// It finds matching command data (/slash, user or message cmd) and returns updated cmd interaction struct.
// It updates cmd itx struct so if it's a subcommand - it's seen as normal, not nested command.
// Read third return param to verify whether command was found.
func (m *InteractionManager) HandleCommandInteraction(itx api.CommandInteraction) (api.CommandInteraction, api.Command, bool) {
	if len(itx.Data.Options) > 0 && itx.Data.Options[0].Type == api.SUB_OPTION_TYPE {
		finalName := itx.Data.Name + "@" + itx.Data.Options[0].Name
		subCommand, available := m.Commands.Get(finalName)
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

	command, available := m.Commands.Get(itx.Data.Name)
	return itx, command, available
}

func parseCommandsForDiscordAPI(commands *SharedMap[string, api.Command], whitelist []string, reverseMode bool) []api.Command {
	commands.mu.RLock()

	tree := make(map[string]map[string]api.Command, len(commands.cache))
	parsedCommands := make([]api.Command, 0, len(commands.cache))

	// First loop - prepare nested space for potential sub commands
	for name, command := range commands.cache {
		if strings.Contains(name, "@") {
			continue
		}

		group := make(map[string]api.Command, 0)
		group[api.ROOT_PLACEHOLDER] = command
		tree[name] = group
	}

	// Second loop - assign commands
	for name, command := range commands.cache {
		if strings.Contains(name, "@") {
			parts := strings.Split(name, "@")
			group := tree[parts[0]]

			command.Type = api.CommandType(api.SUB_OPTION_TYPE)
			group[parts[1]] = command
			tree[parts[0]] = group
		}
	}

	commands.mu.RUnlock()

	// Use nested map to build final array with structs matching Discord API
	for _, branch := range tree {
		baseCommand := branch[api.ROOT_PLACEHOLDER]

		if len(branch) > 1 {
			for key, subCommand := range branch {
				if key == api.ROOT_PLACEHOLDER {
					continue
				}

				baseCommand.Options = append(baseCommand.Options, api.CommandOption{
					Name:        subCommand.Name,
					Description: subCommand.Description,
					Type:        api.SUB_OPTION_TYPE,
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

	var filtered []api.Command

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
