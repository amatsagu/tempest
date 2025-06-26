package tempest

import (
	"errors"
	"net/http"
	"strings"
)

func (client *Client) RegisterCommand(cmd Command) error {
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
	return nil
}

func (client *Client) RegisterSubCommand(subCommand Command, parentCommandName string) error {
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
	return nil
}

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (client *Client) RegisterComponent(customIDs []string, fn func(ComponentInteraction)) error {
	for _, ID := range customIDs {
		if client.staticComponents.Has(ID) {
			return errors.New("client already has registered static component with custom ID = " + ID + " (custom id already in use)")
		}

		if client.queuedComponents.Has(ID) {
			return errors.New("client already has registered dynamic (queued) component with custom ID = " + ID + " (custom id already in use)")
		}
	}

	client.queuedComponents.mu.Lock()
	for _, key := range customIDs {
		client.staticComponents.cache[key] = fn
	}
	client.queuedComponents.mu.Unlock()

	return nil
}

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (client *Client) RegisterModal(customID string, fn func(ModalInteraction)) error {
	if client.staticModals.Has(customID) {
		return errors.New("client already has registered static modal with custom ID = " + customID + " (custom id already in use)")
	}

	if client.queuedModals.Has(customID) {
		return errors.New("client already has registered dynamic (queued) modal with custom ID = " + customID + " (custom id already in use)")
	}

	client.staticModals.Set(customID, fn)
	return nil
}

func (client *Client) FindCommand(cmdName string) (Command, bool) {
	return client.commands.Get(cmdName)
}

func (client *Client) SyncCommandsWithDiscord(guildIDs []Snowflake, whitelist []string, reverseMode bool) error {
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

	return nil
}

func (client *Client) handleInteraction(itx CommandInteraction) (CommandInteraction, Command, bool) {
	if len(itx.Data.Options) > 0 && itx.Data.Options[0].Type == SUB_OPTION_TYPE {
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

	cmdTree := make(map[string]map[string]Command, len(commands.cache))
	cmdRootSymbol := "-"
	parsedCommands := make([]Command, 0, len(commands.cache))

	// Prepare nested map for reading later
	for fullName, command := range commands.cache {
		if strings.Contains(fullName, "@") {
			names := strings.Split(fullName, "@")
			cmdBranch := cmdTree[names[0]]
			cmdBranch[names[1]] = command
			cmdTree[names[0]] = cmdBranch
		}

		cmdBranch := make(map[string]Command, 0)
		cmdBranch[cmdRootSymbol] = command
		cmdTree[fullName] = cmdBranch
	}

	commands.mu.RUnlock()

	// Use nested map to build final array with structs matching Discord API
	for _, branch := range cmdTree {
		baseCommand := branch[cmdRootSymbol]

		if len(branch) > 1 {
			for key, subCommand := range branch {
				if key == cmdRootSymbol {
					continue
				}

				baseCommand.Options = append(baseCommand.Options, CommandOption{
					Name:        subCommand.Name,
					Description: subCommand.Description,
					Type:        SUB_OPTION_TYPE,
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
