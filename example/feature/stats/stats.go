// Package stats renders a Components V2 runtime panel fed by the lifecycle hooks.
package stats

import (
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest"
)

// Stats accumulates interaction counts and the start time; every counter is atomic
// so the hot path never locks or allocates.
type Stats struct {
	started      time.Time
	completed    atomic.Uint64
	commands     atomic.Uint64
	autocomplete atomic.Uint64
	components   atomic.Uint64
	modals       atomic.Uint64
	errors       atomic.Uint64
}

// NewStats returns a Stats whose uptime starts now.
func NewStats() *Stats {
	return &Stats{started: time.Now()}
}

// Completed is the OnCompletedInteraction hook: it counts completions by route kind.
func (s *Stats) Completed(info tempest.InteractionInfo) {
	s.completed.Add(1)

	switch info.Route.Kind {
	case tempest.RouteCommand:
		s.commands.Add(1)
	case tempest.RouteAutocomplete:
		s.autocomplete.Add(1)
	case tempest.RouteComponent:
		s.components.Add(1)
	case tempest.RouteModal:
		s.modals.Add(1)
	}
}

// Failed is the OnInteractionError hook: it counts failures; main.go does the logging.
func (s *Stats) Failed(_ error, _ tempest.InteractionInfo, _ any) {
	s.errors.Add(1)
}

// Snapshot is a point-in-time copy of the counters and uptime.
type Snapshot struct {
	Uptime       time.Duration
	Total        uint64
	Commands     uint64
	Autocomplete uint64
	Components   uint64
	Modals       uint64
	Errors       uint64
}

func (s *Stats) Snapshot() Snapshot {
	return Snapshot{
		Uptime:       time.Since(s.started),
		Total:        s.completed.Load(),
		Commands:     s.commands.Load(),
		Autocomplete: s.autocomplete.Load(),
		Components:   s.components.Load(),
		Modals:       s.modals.Load(),
		Errors:       s.errors.Load(),
	}
}
