// Package stats renders a Components V2 runtime panel: uptime, interaction
// counters, memory, and REST/gateway latency. The counters are fed by the
// client's lifecycle hooks, so they cost nothing when the panel is never used.
package stats

import (
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest"
)

// Stats accumulates interaction counts and remembers when the bot started. Every
// counter is atomic, so the hot path (the completed/error hooks) never locks and
// never allocates.
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

// Completed is the client's OnCompletedInteraction hook: it counts every handler
// that ran to completion and splits it by route kind. It is allocation free and
// safe to call concurrently.
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

// Failed is the client's OnInteractionError hook: it counts handler failures.
// The error, route info and interaction are deliberately unused here; main.go
// logs and answers the interaction with its own hook around this call.
func (s *Stats) Failed(_ error, _ tempest.InteractionInfo, _ any) {
	s.errors.Add(1)
}

// Snapshot is a point-in-time copy of the counters and uptime, taken on the
// panel's slow path.
type Snapshot struct {
	Uptime       time.Duration
	Total        uint64
	Commands     uint64
	Autocomplete uint64
	Components   uint64
	Modals       uint64
	Errors       uint64
}

// Snapshot reads every counter once and computes the uptime.
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
