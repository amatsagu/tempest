// Package stats renders a Components V2 runtime panel fed by ClientHooks.
package stats

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/amatsagu/tempest"
)

// Tracks interaction lifecycle counts across all Pre, Post, and Error hooks.
type Stats struct {
	started             time.Time
	commandsStarted     atomic.Uint64
	commandsCompleted   atomic.Uint64
	componentsStarted   atomic.Uint64
	componentsCompleted atomic.Uint64
	modalsStarted       atomic.Uint64
	modalsCompleted     atomic.Uint64
	errors              atomic.Uint64
}

// Creates a tracker whose uptime starts immediately.
func NewStats() *Stats {
	return &Stats{started: time.Now()}
}

func (s *Stats) PreCommand(_ context.Context, _ tempest.Command, _ *tempest.CommandInteraction) bool {
	s.commandsStarted.Add(1)
	return true
}

func (s *Stats) PostCommand(_ context.Context, _ tempest.Command, _ *tempest.CommandInteraction) {
	s.commandsCompleted.Add(1)
}

func (s *Stats) RecordCommandError(_ context.Context, _ tempest.Command, _ *tempest.CommandInteraction, _ error) {
	s.errors.Add(1)
}

func (s *Stats) PreComponent(_ context.Context, _ *tempest.ComponentInteraction) bool {
	s.componentsStarted.Add(1)
	return true
}

func (s *Stats) PostComponent(_ context.Context, _ *tempest.ComponentInteraction) {
	s.componentsCompleted.Add(1)
}

func (s *Stats) RecordComponentError(_ context.Context, _ *tempest.ComponentInteraction, _ error) {
	s.errors.Add(1)
}

func (s *Stats) PreModal(_ context.Context, _ *tempest.ModalInteraction) bool {
	s.modalsStarted.Add(1)
	return true
}

func (s *Stats) PostModal(_ context.Context, _ *tempest.ModalInteraction) {
	s.modalsCompleted.Add(1)
}

func (s *Stats) RecordModalError(_ context.Context, _ *tempest.ModalInteraction, _ error) {
	s.errors.Add(1)
}

// Point-in-time copy of the lifecycle counters and uptime.
type Snapshot struct {
	Uptime              time.Duration
	CommandsStarted     uint64
	CommandsCompleted   uint64
	ComponentsStarted   uint64
	ComponentsCompleted uint64
	ModalsStarted       uint64
	ModalsCompleted     uint64
	Errors              uint64
}

func (s *Stats) Snapshot() Snapshot {
	return Snapshot{
		Uptime:              time.Since(s.started),
		CommandsStarted:     s.commandsStarted.Load(),
		CommandsCompleted:   s.commandsCompleted.Load(),
		ComponentsStarted:   s.componentsStarted.Load(),
		ComponentsCompleted: s.componentsCompleted.Load(),
		ModalsStarted:       s.modalsStarted.Load(),
		ModalsCompleted:     s.modalsCompleted.Load(),
		Errors:              s.errors.Load(),
	}
}
