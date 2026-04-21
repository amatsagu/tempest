package tempest

import (
	"sync"
	"time"
)

type interactionSweeper struct {
	running bool
	mu      sync.Mutex
}

func (s *interactionSweeper) tryRun(client *BaseClient) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run(client)
}

func (s *interactionSweeper) run(client *BaseClient) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		client.queuedComponents.Sweep(func(key string, value *queuedComponent) bool {
			if now.After(value.Expire) {
				if value.OnTimeout != nil {
					go value.OnTimeout()
				}
				return true
			}
			return false
		})

		client.queuedModals.Sweep(func(key string, value *queuedModal) bool {
			if now.After(value.Expire) {
				if value.OnTimeout != nil {
					go value.OnTimeout()
				}
				return true
			}
			return false
		})

		s.mu.Lock()
		if client.queuedComponents.Size() == 0 && client.queuedModals.Size() == 0 {
			s.running = false
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
	}
}
