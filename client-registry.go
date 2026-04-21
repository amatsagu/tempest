package tempest

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

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
