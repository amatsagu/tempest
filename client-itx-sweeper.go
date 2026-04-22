package tempest

import (
	"sync"
	"time"
)

type queuedComponent struct {
	Handler   func(*ComponentInteraction)
	Expire    time.Time
	OnTimeout func()
}

type queuedModal struct {
	Handler   func(*ModalInteraction)
	Expire    time.Time
	OnTimeout func()
}

type interactionSweeper struct {
	running     bool
	mu          sync.Mutex
	signal      chan struct{}
	lowestTimes [3]time.Time
}

func insertLowestTime(arr *[3]time.Time, expire time.Time) {
	if arr[0].IsZero() || expire.Before(arr[0]) {
		arr[2] = arr[1]
		arr[1] = arr[0]
		arr[0] = expire
	} else if arr[1].IsZero() || expire.Before(arr[1]) {
		arr[2] = arr[1]
		arr[1] = expire
	} else if arr[2].IsZero() || expire.Before(arr[2]) {
		arr[2] = expire
	}
}

func (s *interactionSweeper) tryRun(client *BaseClient, expire time.Time) {
	s.mu.Lock()

	oldShortest := s.lowestTimes[0]

	insertLowestTime(&s.lowestTimes, expire)

	isNewShortest := oldShortest.IsZero() || expire.Before(oldShortest)

	if !s.running {
		s.running = true
		s.mu.Unlock()
		client.tracef("Starting interaction sweeper (initial timer: %s)", time.Until(expire).Round(time.Millisecond))
		go s.run(client)
		return
	}
	s.mu.Unlock()

	if isNewShortest {
		client.tracef("Interaction Sweeper interrupted! New listener expires earlier (in %s).", time.Until(expire).Round(time.Millisecond))
		select {
		case s.signal <- struct{}{}:
		default:
		}
	} else {
		client.tracef("Interaction Sweeper ignored new listener (expires in %s) as current lowest timer has lower time value", time.Until(expire).Round(time.Millisecond))
	}
}

func (s *interactionSweeper) run(client *BaseClient) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	for {
		s.mu.Lock()
		var shortest time.Duration = -1
		lowestTimesCopy := s.lowestTimes
		if !s.lowestTimes[0].IsZero() {
			now := time.Now()
			if now.After(s.lowestTimes[0]) || now.Equal(s.lowestTimes[0]) {
				shortest = 0
			} else {
				shortest = s.lowestTimes[0].Sub(now)
			}
		}
		s.mu.Unlock()

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		if shortest >= 0 {
			timer.Reset(shortest)

			var upcoming []time.Duration
			now := time.Now()
			for _, t := range lowestTimesCopy {
				if !t.IsZero() {
					upcoming = append(upcoming, t.Sub(now).Round(time.Millisecond))
				}
			}
			client.tracef("Interaction Sweeper timer set to %s. Run cleanup in: %v", shortest.Round(time.Millisecond), upcoming)
		}

		select {
		case <-timer.C:
		case <-s.signal:
			client.tracef("Interaction Sweeper timer reset to adapt to new shorter listener timeouts.")
			continue
		}

		now := time.Now()
		var newLowest [3]time.Time

		client.queuedComponents.Sweep(func(key string, value *queuedComponent) bool {
			if now.After(value.Expire) {
				if value.OnTimeout != nil {
					go value.OnTimeout()
				}
				return true
			}
			insertLowestTime(&newLowest, value.Expire)
			return false
		})

		client.queuedModals.Sweep(func(key string, value *queuedModal) bool {
			if now.After(value.Expire) {
				if value.OnTimeout != nil {
					go value.OnTimeout()
				}
				return true
			}
			insertLowestTime(&newLowest, value.Expire)
			return false
		})

		s.mu.Lock()
		s.lowestTimes = newLowest
		if client.queuedComponents.Size() == 0 && client.queuedModals.Size() == 0 {
			s.running = false
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
	}
}
