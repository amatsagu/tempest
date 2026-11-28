package tags

import (
	"maps"
	"slices"
	"sync"

	"github.com/amatsagu/tempest/discord"
)

// In-memory guild tag store, reset on restart.
type tagStore struct {
	servers map[discord.GuildID]map[string]string
	mu      sync.RWMutex
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

func (s *tagStore) names(guild discord.GuildID) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Sorted(maps.Keys(s.servers[guild]))
}

func (s *tagStore) clear(guild discord.GuildID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.servers, guild)
}
