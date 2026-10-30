// Package stats renders a Components V2 runtime panel.
package stats

import (
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest"
)

// Stats accumulates the interaction count and start time. The master runtime
// exposes only command lifecycle hooks, so this example counts commands.
type Stats struct {
	started  time.Time
	commands atomic.Uint64
}

// NewStats returns a Stats whose uptime starts now.
func NewStats() *Stats {
	return &Stats{started: time.Now()}
}

// PreCommand is installed as BaseClientOptions.PreCommandHook.
func (s *Stats) PreCommand(_ tempest.Command, _ *tempest.CommandInteraction) bool {
	s.commands.Add(1)
	return true
}

// Snapshot is a point-in-time copy of the counters and uptime.
type Snapshot struct {
	Uptime   time.Duration
	Commands uint64
}

// Snapshot returns the current counters.
func (s *Stats) Snapshot() Snapshot {
	return Snapshot{
		Uptime:   time.Since(s.started),
		Commands: s.commands.Load(),
	}
}
