package tags

import (
	"maps"
	"slices"
	"sync"

	"github.com/amatsagu/tempest/discord"
)

// tagStore keeps each server's tags in memory. State lives here rather than in
// the command so the mutex is captured once and shared by every invocation.
//
// ponytail: never persisted and never pruned; a real bot would back this with a
// database and cap the tags per server. The example only shows the shape.
type tagStore struct {
	mu      sync.RWMutex
	servers map[discord.GuildID]map[string]string
}

func newTagStore() *tagStore {
	return &tagStore{servers: make(map[discord.GuildID]map[string]string)}
}

func (s *tagStore) add(guild discord.GuildID, name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := s.servers[guild]
	if tags == nil {
		tags = make(map[string]string)
		s.servers[guild] = tags
	}
	tags[name] = value
}

func (s *tagStore) get(guild discord.GuildID, name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.servers[guild][name]
	return value, ok
}

func (s *tagStore) remove(guild discord.GuildID, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := s.servers[guild]
	if _, ok := tags[name]; !ok {
		return false
	}
	delete(tags, name)
	return true
}

// names returns a guild's tag names in a stable order, so the list reply and
// autocomplete suggestions do not shuffle between calls.
func (s *tagStore) names(guild discord.GuildID) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Sorted(maps.Keys(s.servers[guild]))
}
