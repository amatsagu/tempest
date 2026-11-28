package tempest

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"slices"
	"strings"

	"github.com/amatsagu/tempest/discord"
)

type (
	CommandHandler      func(ctx context.Context, itx *CommandInteraction) error
	AutoCompleteHandler func(ctx context.Context, itx *CommandInteraction) ([]discord.CommandOptionChoice, error)
	ComponentHandler    func(ctx context.Context, itx *ComponentInteraction) error
	ModalHandler        func(ctx context.Context, itx *ModalInteraction) error
)

// Wraps [discord.Command] with its slash and optional autocomplete handlers.
type Command struct {
	SlashCommandHandler CommandHandler      `json:"-"`
	AutoCompleteHandler AutoCompleteHandler `json:"-"`
	discord.Command
}

// Registers a root application command (slash or context menu).
func (c *Client) RegisterCommand(cmd Command) error {
	if cmd.Name == "" {
		return errors.New("command name must not be empty")
	}

	if cmd.Type == 0 {
		cmd.Type = discord.ChatInputCommandType
	}
	if cmd.ApplicationID.Snowflake == 0 {
		cmd.ApplicationID = c.ApplicationID
	}
	if len(cmd.Contexts) == 0 {
		cmd.Contexts = c.defaultContexts
	}

	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	if _, exists := c.commands[cmd.Name]; exists {
		return fmt.Errorf("command %q is already registered", cmd.Name)
	}

	c.commands[cmd.Name] = cmd
	c.tracef("Registered command %q.", cmd.Name)
	return nil
}

// Registers a subcommand under parentPath (1 name for "<parent>@<sub>",
// or 2 names for "<parent>@<group>@<sub>").
func (c *Client) RegisterSubCommand(sub Command, parentPath ...string) error {
	if len(parentPath) < 1 || len(parentPath) > 2 {
		return fmt.Errorf("RegisterSubCommand requires 1 or 2 parent path segments, got %d", len(parentPath))
	}
	if sub.Name == "" {
		return errors.New("subcommand name must not be empty")
	}

	parentName := parentPath[0]
	for _, seg := range parentPath {
		if seg == "" || strings.ContainsRune(seg, '@') {
			return fmt.Errorf("invalid parent path segment %q", seg)
		}
	}

	key := strings.Join(parentPath, "@") + "@" + sub.Name

	if sub.Type == 0 {
		sub.Type = discord.ChatInputCommandType
	}
	if sub.ApplicationID.Snowflake == 0 {
		sub.ApplicationID = c.ApplicationID
	}
	if len(sub.Contexts) == 0 {
		sub.Contexts = c.defaultContexts
	}

	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	if _, exists := c.commands[parentName]; !exists {
		return fmt.Errorf("parent command %q is not registered", parentName)
	}
	if _, exists := c.commands[key]; exists {
		return fmt.Errorf("subcommand %q is already registered", key)
	}

	c.commands[key] = sub
	c.tracef("Registered subcommand %q.", key)
	return nil
}

// Removes a registered command or subcommand ("parent@sub" or "parent@group@sub").
func (c *Client) UnregisterCommand(name string) bool {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	if _, exists := c.commands[name]; !exists {
		return false
	}

	delete(c.commands, name)
	c.tracef("Unregistered command %q.", name)
	return true
}

// Looks up a registered command or subcommand key.
func (c *Client) FindCommand(name string) (Command, bool) {
	c.cmdMu.RLock()
	cmd, ok := c.commands[name]
	c.cmdMu.RUnlock()
	return cmd, ok
}

// Iterates over registered command keys ("cmd", "parent@sub", or "parent@group@sub").
func (c *Client) RegisteredCommands() iter.Seq2[string, Command] {
	return func(yield func(string, Command) bool) {
		c.cmdMu.RLock()
		snapshot := make(map[string]Command, len(c.commands))
		for k, v := range c.commands {
			snapshot[k] = v
		}
		c.cmdMu.RUnlock()

		keys := make([]string, 0, len(snapshot))
		for k := range snapshot {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		for _, k := range keys {
			if !yield(k, snapshot[k]) {
				return
			}
		}
	}
}

// Registers a component handler for an exact custom ID or {name} wildcard pattern.
func (c *Client) RegisterComponent(pattern string, handler ComponentHandler) error {
	if handler == nil {
		return errors.New("component handler must not be nil")
	}

	if err := c.components.register(pattern, handler); err != nil {
		return err
	}

	c.tracef("Registered component pattern %q.", pattern)
	return nil
}

// Removes a component pattern or exact custom ID. Reports whether an entry existed.
func (c *Client) UnregisterComponent(pattern string) bool {
	removed := c.components.unregister(pattern)
	if removed {
		c.tracef("Unregistered component pattern %q.", pattern)
	}
	return removed
}

// Removes all registered component handlers.
func (c *Client) ClearComponents() {
	c.components.clear()
	c.tracef("Cleared all registered component handlers.")
}

// Iterates over all registered component IDs and patterns.
func (c *Client) RegisteredComponents() iter.Seq[string] {
	return c.components.all()
}

// Registers a modal handler for an exact custom ID or {name} wildcard pattern.
func (c *Client) RegisterModal(pattern string, handler ModalHandler) error {
	if handler == nil {
		return errors.New("modal handler must not be nil")
	}

	if err := c.modals.register(pattern, handler); err != nil {
		return err
	}

	c.tracef("Registered modal pattern %q.", pattern)
	return nil
}

// Removes a modal pattern or exact custom ID. Reports whether an entry existed.
func (c *Client) UnregisterModal(pattern string) bool {
	removed := c.modals.unregister(pattern)
	if removed {
		c.tracef("Unregistered modal pattern %q.", pattern)
	}
	return removed
}

// Removes all registered modal handlers.
func (c *Client) ClearModals() {
	c.modals.clear()
	c.tracef("Cleared all registered modal handlers.")
}

// Iterates over all registered modal IDs and patterns.
func (c *Client) RegisteredModals() iter.Seq[string] {
	return c.modals.all()
}

// Bulk-overwrites registered commands globally (when guildIDs is empty) or per guild.
// When filter is non-empty, excludeMode=false keeps only listed names and excludeMode=true excludes them.
func (c *Client) SyncCommands(ctx context.Context, guildIDs []discord.GuildID, filter []string, excludeMode bool) error {
	commands := c.buildSyncCommands(filter, excludeMode)

	if len(guildIDs) == 0 {
		err := c.Rest.Request(ctx, http.MethodPut, "/applications/"+c.ApplicationID.String()+"/commands", commands, nil)
		if err == nil {
			c.tracef("Synced %d command(s) globally with Discord.", len(commands))
		}
		return err
	}

	for _, guildID := range guildIDs {
		err := c.Rest.Request(ctx, http.MethodPut, "/applications/"+c.ApplicationID.String()+"/guilds/"+guildID.String()+"/commands", commands, nil)
		if err != nil {
			return err
		}
	}

	c.tracef("Synced %d command(s) across %d guild(s) with Discord.", len(commands), len(guildIDs))
	return nil
}

func (c *Client) resolveCommand(itx *CommandInteraction) (Command, bool) {
	if itx.Member != nil {
		itx.Member.GuildID = itx.GuildID
	}

	lookupName := itx.Data.Name
	subOpts := itx.Data.Options

	if len(itx.Data.Options) > 0 {
		first := itx.Data.Options[0]
		switch first.Type {
		case discord.SubCommandOptionType:
			lookupName = itx.Data.Name + "@" + first.Name
			subOpts = first.Options
		case discord.SubCommandGroupOptionType:
			if len(first.Options) > 0 && first.Options[0].Type == discord.SubCommandOptionType {
				sub := first.Options[0]
				lookupName = itx.Data.Name + "@" + first.Name + "@" + sub.Name
				subOpts = sub.Options
			}
		}
	}

	c.cmdMu.RLock()
	cmd, ok := c.commands[lookupName]
	c.cmdMu.RUnlock()
	if ok {
		itx.Data.Name = lookupName
		itx.Data.Options = subOpts
	}

	return cmd, ok
}

func (c *Client) buildSyncCommands(filter []string, excludeMode bool) []discord.Command {
	c.cmdMu.RLock()
	snapshot := make(map[string]Command, len(c.commands))
	for k, v := range c.commands {
		snapshot[k] = v
	}
	c.cmdMu.RUnlock()

	type rootTree struct {
		groups map[string][]discord.CommandOption
		subs   []discord.CommandOption
		root   discord.Command
	}

	roots := make(map[string]*rootTree)
	for key, cmd := range snapshot {
		if !strings.ContainsRune(key, '@') {
			copied := cmd.Command
			if len(copied.Options) > 0 {
				opts := make([]discord.CommandOption, len(copied.Options))
				copy(opts, copied.Options)
				copied.Options = opts
			}
			roots[key] = &rootTree{root: copied}
		}
	}

	for key, cmd := range snapshot {
		if !strings.ContainsRune(key, '@') {
			continue
		}

		parts := strings.Split(key, "@")
		tree, ok := roots[parts[0]]
		if !ok {
			continue
		}

		subOpt := discord.CommandOption{
			Name:                     cmd.Name,
			NameLocalizations:        cmd.NameLocalizations,
			Description:              cmd.Description,
			DescriptionLocalizations: cmd.DescriptionLocalizations,
			Type:                     discord.SubCommandOptionType,
			Options:                  cmd.Options,
		}

		if len(parts) == 2 {
			tree.subs = append(tree.subs, subOpt)
			continue
		}

		if len(parts) == 3 {
			if tree.groups == nil {
				tree.groups = make(map[string][]discord.CommandOption)
			}
			tree.groups[parts[1]] = append(tree.groups[parts[1]], subOpt)
		}
	}

	rootNames := make([]string, 0, len(roots))
	for name := range roots {
		rootNames = append(rootNames, name)
	}
	slices.Sort(rootNames)

	parsed := make([]discord.Command, 0, len(rootNames))
	for _, name := range rootNames {
		tree := roots[name]
		slices.SortFunc(tree.subs, func(a, b discord.CommandOption) int {
			return strings.Compare(a.Name, b.Name)
		})
		tree.root.Options = append(tree.root.Options, tree.subs...)

		if len(tree.groups) > 0 {
			groupNames := make([]string, 0, len(tree.groups))
			for gName := range tree.groups {
				groupNames = append(groupNames, gName)
			}
			slices.Sort(groupNames)

			for _, gName := range groupNames {
				opts := tree.groups[gName]
				slices.SortFunc(opts, func(a, b discord.CommandOption) int {
					return strings.Compare(a.Name, b.Name)
				})

				merged := false
				for i := range tree.root.Options {
					if tree.root.Options[i].Type == discord.SubCommandGroupOptionType && tree.root.Options[i].Name == gName {
						tree.root.Options[i].Options = append(tree.root.Options[i].Options, opts...)
						merged = true
						break
					}
				}
				if !merged {
					tree.root.Options = append(tree.root.Options, discord.CommandOption{
						Name:        gName,
						Description: gName,
						Type:        discord.SubCommandGroupOptionType,
						Options:     opts,
					})
				}
			}
		}

		parsed = append(parsed, tree.root)
	}

	if len(filter) == 0 {
		return parsed
	}

	filterSet := make(map[string]struct{}, len(filter))
	for _, name := range filter {
		filterSet[name] = struct{}{}
	}

	filtered := make([]discord.Command, 0, len(parsed))
	for _, cmd := range parsed {
		_, listed := filterSet[cmd.Name]
		if (excludeMode && !listed) || (!excludeMode && listed) {
			filtered = append(filtered, cmd)
		}
	}

	return filtered
}
