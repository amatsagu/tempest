package tags

import (
	"maps"
	"slices"
	"sync"

	"github.com/amatsagu/tempest"
)

// tagStore keeps each server's tags in memory, shared by every invocation.
//
// This is intentionally never persisted or pruned; a real bot would back it with
// a database and cap the tags per server.
type tagStore struct {
	servers map[tempest.Snowflake]map[string]string
	mu      sync.RWMutex
}

func newTagStore() *tagStore {
	return &tagStore{servers: make(map[tempest.Snowflake]map[string]string)}
}

func (s *tagStore) add(guild tempest.Snowflake, name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := s.servers[guild]
	if tags == nil {
		tags = make(map[string]string)
		s.servers[guild] = tags
	}
	tags[name] = value
}

func (s *tagStore) get(guild tempest.Snowflake, name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.servers[guild][name]
	return value, ok
}

func (s *tagStore) remove(guild tempest.Snowflake, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tags := s.servers[guild]
	if _, ok := tags[name]; !ok {
		return false
	}
	delete(tags, name)
	return true
}

// names returns a guild's tag names in a stable order.
func (s *tagStore) names(guild tempest.Snowflake) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Sorted(maps.Keys(s.servers[guild]))
}

func (s *tagStore) clear(guild tempest.Snowflake) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.servers, guild)
}
