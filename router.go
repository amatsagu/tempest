package tempest

import (
	"context"
	"encoding/json/jsontext"
	"strings"
	"sync"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/route"
)

// CommandHandler handles a command invocation. The response is sent through the
// interaction itself; a returned error goes to Config.OnInteractionError.
type CommandHandler func(ctx context.Context, itx *discord.CommandInteraction) error

// AutocompleteHandler handles an autocomplete invocation and returns choices.
type AutocompleteHandler func(ctx context.Context, itx *discord.CommandInteraction) ([]discord.CommandOptionChoice, error)

// ComponentHandler handles a component interaction. params holds any values
// captured by the pattern's {name} wildcards (nil for exact IDs).
type ComponentHandler func(ctx context.Context, itx *discord.ComponentInteraction, params *route.Params) error

// ModalHandler handles a modal submit interaction, with the same params as Component.
type ModalHandler func(ctx context.Context, itx *discord.ModalInteraction, params *route.Params) error

// EventHandler handles a gateway event that was explicitly subscribed to.
type EventHandler func(ctx context.Context, shard int, event string, data jsontext.Value)

// handlerSet routes interactions by command name and by custom ID (exact or pattern).
type handlerSet struct {
	commands     map[string]CommandHandler
	autocomplete map[string]AutocompleteHandler
	components   idRouter[ComponentHandler]
	modals       idRouter[ModalHandler]
	mu           sync.RWMutex
}

func newHandlerSet() *handlerSet {
	return &handlerSet{
		commands:     make(map[string]CommandHandler),
		autocomplete: make(map[string]AutocompleteHandler),
	}
}

func (h *handlerSet) command(name string, handler CommandHandler) {
	h.mu.Lock()
	h.commands[name] = handler
	h.mu.Unlock()
}

func (h *handlerSet) commandHandler(name string) (CommandHandler, bool) {
	h.mu.RLock()
	handler, ok := h.commands[name]
	h.mu.RUnlock()
	return handler, ok
}

func (h *handlerSet) autocompleteCommand(name string, handler AutocompleteHandler) {
	h.mu.Lock()
	h.autocomplete[name] = handler
	h.mu.Unlock()
}

func (h *handlerSet) autocompleteHandler(name string) (AutocompleteHandler, bool) {
	h.mu.RLock()
	handler, ok := h.autocomplete[name]
	h.mu.RUnlock()
	return handler, ok
}

// idRouter matches a custom ID either exactly (O(1) map lookup) or against
// registered patterns. All operations take the same lock, and exact IDs are stored
// in a plain map so removal frees the entry under churn of one-shot dynamic IDs.
type idRouter[T any] struct {
	exact    map[string]T
	patterns route.Router[T]
	mu       sync.RWMutex
}

// register adds or replaces a handler for id; an ID containing '{' is a pattern.
func (r *idRouter[T]) register(id string, handler T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if strings.ContainsRune(id, '{') {
		// Remove first so re-registering a pattern swaps the handler; a genuine
		// conflict removes nothing and still reports an error.
		r.patterns.Remove(id)
		return r.patterns.Handle(id, handler)
	}

	if r.exact == nil {
		r.exact = make(map[string]T)
	}
	r.exact[id] = handler
	return nil
}

func (r *idRouter[T]) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if strings.ContainsRune(id, '{') {
		r.patterns.Remove(id)
		return
	}
	delete(r.exact, id)
}

// match resolves id to a handler and its route. params may be nil for exact
// registrations, in which case it is only written when a pattern is consulted.
func (r *idRouter[T]) match(id string, params *route.Params) (T, Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if handler, ok := r.exact[id]; ok {
		return handler, Route{Pattern: id, Exact: true}, true
	}

	if params == nil {
		var scratch route.Params
		params = &scratch
	}
	handler, pattern, ok := r.patterns.MatchRoute(id, params)
	if !ok {
		return handler, Route{}, false
	}
	return handler, Route{Pattern: pattern}, true
}
