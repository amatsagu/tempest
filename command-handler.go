package tempest

import (
	"errors"
	"net/http"
	"sync"
)

// compile-time interface assertion
var _ SlashCommandRegistry = (*BaseSlashCommandRegistry)(nil)

type SlashCommandHandler interface {
	Data() Command // Command definition metadata (name, description, options, etc.)
	AutoCompleteHandler(itx *CommandInteraction) []Choice
	CommandHandler(itx CommandInteraction)
}

type SlashCommandRegistry interface {
	ApplicationID() Snowflake // ID of bot/application
	Register(handler SlashCommandHandler) error
	RegisterSub(subHandler SlashCommandHandler, rootCommandName string) error
	Find(cmdName string, subCmdName string) (SlashCommandHandler, bool)
	HandleInteraction(itx CommandInteraction) (CommandInteraction, SlashCommandHandler, bool)
	SyncWithDiscord(rest RestHandler, guildIDs []Snowflake, whitelist []string, reverseMode bool) error
	PreCommandHook(itx CommandInteraction) bool // Function that runs before each command. Return type signals whether to continue command execution (return with false to stop early).
	PostCommandHook(itx CommandInteraction)     // Function that runs after each command.
}

type BaseSlashCommandRegistry struct {
	mu                         sync.RWMutex
	applicationID              Snowflake
	commands                   map[string]map[string]SlashCommandHandler
	defaultInteractionContexts []InteractionContextType
}

func NewBaseSlashCommandRegistry(applicationID Snowflake, defaultInteractionContexts []InteractionContextType) SlashCommandRegistry {
	return &BaseSlashCommandRegistry{
		applicationID: applicationID,
		commands:      make(map[string]map[string]SlashCommandHandler),
	}
}

func (reg *BaseSlashCommandRegistry) ApplicationID() Snowflake {
	return reg.applicationID
}

func (reg *BaseSlashCommandRegistry) Register(handler SlashCommandHandler) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	data := handler.Data()
	if _, exists := reg.commands[data.Name]; exists {
		return errors.New("client already has registered \"" + data.Name + "\" slash command (name already in use)")
	}

	if data.Type == 0 {
		data.Type = CHAT_INPUT_COMMAND_TYPE
	}

	if data.ApplicationID == 0 {
		data.ApplicationID = reg.applicationID
	}

	if len(data.Contexts) == 0 {
		data.Contexts = reg.defaultInteractionContexts
	}

	tree := make(map[string]SlashCommandHandler)
	tree[ROOT_PLACEHOLDER] = handler // ROOT_PLACEHOLDER = "-"
	reg.commands[data.Name] = tree
	return nil
}

func (reg *BaseSlashCommandRegistry) RegisterSub(subHandler SlashCommandHandler, rootCommandName string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if _, available := reg.commands[rootCommandName]; !available {
		return errors.New("missing \"" + rootCommandName + "\" slash command in registry (root command needs to be registered in client before adding subcommands)")
	}

	data := subHandler.Data()
	if _, available := reg.commands[rootCommandName][data.Name]; available {
		return errors.New("client already has registered \"" + rootCommandName + "@" + data.Name + "\" slash subcommand")
	}

	if data.Type == 0 {
		data.Type = CHAT_INPUT_COMMAND_TYPE
	}

	if data.ApplicationID == 0 {
		data.ApplicationID = reg.applicationID
	}

	if len(data.Contexts) == 0 {
		data.Contexts = reg.defaultInteractionContexts
	}

	reg.commands[rootCommandName][data.Name] = subHandler
	return nil
}

func (reg *BaseSlashCommandRegistry) Find(cmdName string, subCmdName string) (SlashCommandHandler, bool) {
	reg.mu.RLock()
	// Unlock on each return point instead using defer to increase this hot spot performance
	// defer reg.mu.RUnlock()

	tree, ok := reg.commands[cmdName]
	if !ok {
		reg.mu.RUnlock()
		return nil, false
	}

	if subCmdName != "" {
		handler, ok := tree[subCmdName]
		reg.mu.RUnlock()
		return handler, ok
	}

	handler, ok := tree[ROOT_PLACEHOLDER]
	reg.mu.RUnlock()
	return handler, ok
}

func (reg *BaseSlashCommandRegistry) HandleInteraction(itx CommandInteraction) (CommandInteraction, SlashCommandHandler, bool) {
	if len(itx.Data.Options) != 0 && itx.Data.Options[0].Type == SUB_OPTION_TYPE {
		sub, available := reg.commands[itx.Data.Name][itx.Data.Options[0].Name]
		if available {
			if itx.Member != nil {
				itx.Member.GuildID = itx.GuildID
			}

			itx.Data.Name, itx.Data.Options = itx.Data.Options[0].Name, itx.Data.Options[0].Options
		}
		return itx, sub, available
	}

	if itx.Member != nil {
		itx.Member.GuildID = itx.GuildID
	}

	root, available := reg.commands[itx.Data.Name][ROOT_PLACEHOLDER]
	return itx, root, available
}

func (reg *BaseSlashCommandRegistry) SyncWithDiscord(rest RestHandler, guildIDs []Snowflake, whitelist []string, reverseMode bool) error {
	commands := reg.parseCommandsForDiscordAPI(whitelist, reverseMode)

	if len(guildIDs) == 0 {
		_, err := rest.Request(http.MethodPut, "/applications/"+reg.applicationID.String()+"/commands", commands)
		return err
	}

	for _, guildID := range guildIDs {
		_, err := rest.Request(http.MethodPut, "/applications/"+reg.applicationID.String()+"/guilds/"+guildID.String()+"/commands", commands)
		if err != nil {
			return err
		}
	}

	return nil
}

func (reg *BaseSlashCommandRegistry) parseCommandsForDiscordAPI(whitelist []string, reverseMode bool) []Command {
	var commands []Command

	for _, tree := range reg.commands {
		rootData := tree[ROOT_PLACEHOLDER].Data()

		if len(tree) > 1 {
			for key, sub := range tree {
				if key == ROOT_PLACEHOLDER {
					continue
				}

				subData := sub.Data()
				rootData.Options = append(rootData.Options, CommandOption{
					Name:        subData.Name,
					Description: subData.Description,
					Type:        SUB_OPTION_TYPE,
					Options:     subData.Options,
				})
			}
		}

		commands = append(commands, rootData)
	}

	if len(whitelist) == 0 {
		return commands
	}

	// Build map for fast lookup
	filterMap := make(map[string]struct{}, len(whitelist))
	for _, name := range whitelist {
		filterMap[name] = struct{}{}
	}

	var filtered []Command

	if reverseMode {
		// BLACKLIST: exclude listed commands
		for _, cmd := range commands {
			if _, blocked := filterMap[cmd.Name]; blocked {
				continue
			}
			filtered = append(filtered, cmd)
		}
	} else {
		// WHITELIST: include only listed commands
		for _, cmd := range commands {
			if _, allowed := filterMap[cmd.Name]; allowed {
				filtered = append(filtered, cmd)
			}
		}
	}

	return filtered
}

func (reg *BaseSlashCommandRegistry) PreCommandHook(itx CommandInteraction) bool {
	return true
}

func (reg *BaseSlashCommandRegistry) PostCommandHook(itx CommandInteraction) {
}
