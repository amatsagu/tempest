package tempest

import (
	"errors"
	"net/http"
)

func (client *Client) RegisterCommand(command Command) error {
	if client.State() != INIT_STATE {
		return errors.New("client is no longer in initialization state (avoid editing client's internals after it launches)")
	}

	if _, exists := client.commands[command.Name]; exists {
		return errors.New("client already has registered \"" + command.Name + "\" slash command (name already in use)")
	}

	if command.Type == 0 {
		command.Type = CHAT_INPUT_COMMAND_TYPE
	}

	tree := make(map[string]Command)
	tree[ROOT_PLACEHOLDER] = command
	client.commands[command.Name] = tree
	return nil
}

func (client *Client) RegisterSubCommand(subCommand Command, rootCommandName string) error {
	if client.State() != INIT_STATE {
		return errors.New("client is no longer in initialization state (avoid editing client's internals after it launches)")
	}

	if _, available := client.commands[rootCommandName]; !available {
		return errors.New("missing \"" + rootCommandName + "\" slash command in registry (root command needs to be registered in client before adding subcommands)")
	}

	if _, available := client.commands[rootCommandName][subCommand.Name]; available {
		return errors.New("client already has registered \"" + rootCommandName + "@" + subCommand.Name + "\" slash subcommand")
	}

	client.commands[rootCommandName][subCommand.Name] = subCommand
	return nil
}

// Bind function to all components with matching custom ids. App will automatically run bound function whenever receiving component interaction with matching custom id.
func (client *Client) RegisterComponent(customIDs []string, fn func(*ComponentInteraction)) error {
	if client.State() != INIT_STATE {
		return errors.New("client is no longer in initialization state (avoid editing client's internals after it launches)")
	}

	for _, ID := range customIDs {
		_, exists := client.components[ID]
		if exists {
			return errors.New("client already has registered \"" + ID + "\" component (custom id already in use)")
		}
	}

	for _, ID := range customIDs {
		client.components[ID] = fn
	}

	return nil
}

// Bind function to modal with matching custom id. App will automatically run bound function whenever receiving modal interaction with matching custom id.
func (client *Client) RegisterModal(customID string, fn func(*ModalInteraction)) error {
	if client.State() != INIT_STATE {
		return errors.New("client is no longer in initialization state (avoid editing client's internals after it launches)")
	}

	_, exists := client.modals[customID]
	if exists {
		return errors.New("client already has registered \"" + customID + "\" modal (custom id already in use)")
	}

	client.modals[customID] = fn
	return nil
}

// Sync currently cached slash commands to discord API. By default it'll try to make (bulk) global update (limit 100 updates per day), provide array with guild id snowflakes to update data only for specific guilds.
// You can also add second param -> slice with all command names you want to update (whitelist). There's also third, boolean param that when = true will reverse wishlist to work as blacklist.
func (client *Client) SyncCommands(guildIDs []Snowflake, whitelist []string, switchMode bool) error {
	payload := client.parseCommands(whitelist, switchMode)

	if len(guildIDs) == 0 {
		_, err := client.Rest.Request(http.MethodPut, "/applications/"+client.ApplicationID.String()+"/commands", payload)
		return err
	}

	for _, guildID := range guildIDs {
		_, err := client.Rest.Request(http.MethodPut, "/applications/"+client.ApplicationID.String()+"/guilds/"+guildID.String()+"/commands", payload)
		if err != nil {
			return err
		}
	}

	return nil
}

func (client *Client) seekCommand(itx CommandInteraction) (CommandInteraction, *Command, bool) {
	if len(itx.Data.Options) != 0 && itx.Data.Options[0].Type == SUB_OPTION_TYPE {
		command, available := client.commands[itx.Data.Name][itx.Data.Options[0].Name]
		if available {
			if itx.Member != nil {
				itx.Member.GuildID = itx.GuildID
			}

			itx.Data.Name, itx.Data.Options = itx.Data.Options[0].Name, itx.Data.Options[0].Options
			itx.Client = client
		}
		return itx, &command, available
	}

	if itx.Member != nil {
		itx.Member.GuildID = itx.GuildID
	}

	itx.Client = client
	command, available := client.commands[itx.Data.Name][ROOT_PLACEHOLDER]
	return itx, &command, available
}

// Parses registered commands into Discord format.
func (client *Client) parseCommands(whitelist []string, reverseMode bool) []Command {
	list := make([]Command, len(client.commands))
	var itx uint32 = 0

	for _, tree := range client.commands {
		command := tree[ROOT_PLACEHOLDER]

		if len(tree) > 1 {
			for key, subCommand := range tree {
				if key == ROOT_PLACEHOLDER {
					continue
				}

				command.Options = append(command.Options, CommandOption{
					Name:        subCommand.Name,
					Description: subCommand.Description,
					Type:        SUB_OPTION_TYPE,
					Options:     subCommand.Options,
				})
			}
		}

		list[itx] = command
		itx++
	}

	wls := len(whitelist)
	if wls == 0 {
		return list
	}

	itx = 0

	// Work as blacklist
	if reverseMode {
		filteredList := make([]Command, len(list)-wls)

		for itx, command := range list {
			blocked := false
			for _, cmdName := range whitelist {
				if command.Name == cmdName {
					blocked = true
					break
				}
			}

			if blocked {
				continue
			}

			filteredList[itx] = command
		}

		return filteredList
	}

	// Work as whitelist
	filteredList := make([]Command, wls)

	for _, command := range list {
		for _, cmdName := range whitelist {
			if command.Name == cmdName {
				filteredList[itx] = command
				itx++
			}
		}
	}

	return filteredList
}
