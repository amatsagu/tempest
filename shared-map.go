package tempest

import (
	"iter"
	"sync"
)

// Map but wrapped with mutex so it's safe to use between goroutines.
// It also supports few extra, helper methods that are new to std Map.
type SharedMap[K comparable, V any] struct {
	mu    sync.RWMutex
	cache map[K]V
}

func NewSharedMap[K comparable, V any]() *SharedMap[K, V] {
	return &SharedMap[K, V]{
		mu:    sync.RWMutex{},
		cache: make(map[K]V),
	}
}

func (sm *SharedMap[K, V]) Has(key K) bool {
	sm.mu.RLock()
	_, available := sm.cache[key]
	sm.mu.RUnlock()
	return available
}

func (sm *SharedMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	sm.cache[key] = value
	sm.mu.Unlock()
}

func (sm *SharedMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	item, available := sm.cache[key]
	sm.mu.RUnlock()
	return item, available
}

func (sm *SharedMap[K, V]) Delete(key K) {
	sm.mu.Lock()
	delete(sm.cache, key)
	sm.mu.Unlock()
}

func (sm *SharedMap[K, V]) Reset() {
	sm.mu.Lock()
	clear(sm.cache)
	sm.mu.Unlock()
}

func (sm *SharedMap[K, V]) Size() int {
	sm.mu.RLock()
	length := len(sm.cache)
	sm.mu.RUnlock()
	return length
}

// Creates a copy of a given shared map,
// filtered down to just the elements from the given array that pass the test implemented by the provided function.
func (sm *SharedMap[K, V]) FilterMap(filter func(key K, value V) bool, limit int) *SharedMap[K, V] {
	res := NewSharedMap[K, V]()
	sm.mu.RLock()

	i := 0
	if limit == 0 {
		limit = len(sm.cache)
	}

	for key, value := range sm.cache {
		if filter(key, value) {
			res.Set(key, value)
			i++
			if i == limit {
				break
			}
		}
	}

	sm.mu.RUnlock()
	return res
}

// Same as FilterMap but returns slice of values that pass the provided test function.
func (sm *SharedMap[K, V]) FilterValues(filter func(key K, value V) bool, limit int) []V {
	var res []V

	if limit == 0 {
		res = make([]V, 0)
	} else {
		res = make([]V, 0, limit)
	}

	sm.mu.RLock()
	i := 0
	if limit == 0 {
		limit = len(sm.cache)
	}

	for key, value := range sm.cache {
		if filter(key, value) {
			res = append(res, value)
			i++
			if i == limit {
				break
			}
		}
	}

	sm.mu.RUnlock()
	return res
}

// Same as FilterMap but returns slice of keys that pass the provided test function.
func (sm *SharedMap[K, V]) FilterKeys(filter func(key K, value V) bool, limit int) []K {
	var res []K

	if limit == 0 {
		res = make([]K, 0)
	} else {
		res = make([]K, 0, limit)
	}

	sm.mu.RLock()
	i := 0
	if limit == 0 {
		limit = len(sm.cache)
	}

	for key, value := range sm.cache {
		if filter(key, value) {
			res = append(res, key)
			i++
			if i == limit {
				break
			}
		}
	}

	sm.mu.RUnlock()
	return res
}

// Deletes items that satisfy the provided filter function within 1 mutex lock.
func (sm *SharedMap[K, V]) Sweep(filter func(key K, value V) bool) {
	sm.mu.Lock()

	for key, value := range sm.cache {
		if filter(key, value) {
			delete(sm.cache, key)
		}
	}

	sm.mu.Unlock()
}

// Returns an iterator over the map's key-value pairs.
// Map contents stays locked for the entire duration of the iteration.
//
// Warning! Do not perform any heavy computation or blocking operations (like I/O) inside the loop,
// as it will block all other map operations (including Writes) for the entire duration of the loop.
func (sm *SharedMap[K, V]) Entries() iter.Seq2[K, V] {
	return func(yieldFn func(K, V) bool) {
		sm.mu.RLock()
		defer sm.mu.RUnlock()
		for k, v := range sm.cache {
			if !yieldFn(k, v) {
				return
			}
		}
	}
}

// Returns an iterator over the map's keys.
// Map contents stays locked for the entire duration of the iteration.
//
// Warning! Do not perform any heavy computation or blocking operations (like I/O) inside the loop,
// as it will block all other map operations (including Writes) for the entire duration of the loop.
func (sm *SharedMap[K, V]) Keys() iter.Seq[K] {
	return func(yieldFn func(K) bool) {
		sm.mu.RLock()
		defer sm.mu.RUnlock()
		for k := range sm.cache {
			if !yieldFn(k) {
				return
			}
		}
	}
}

// Returns an iterator over the map's values.
// Map contents stays locked for the entire duration of the iteration.
//
// Warning! Do not perform any heavy computation or blocking operations (like I/O) inside the loop,
// as it will block all other map operations (including Writes) for the entire duration of the loop.
func (sm *SharedMap[K, V]) Values() iter.Seq[V] {
	return func(yieldFn func(V) bool) {
		sm.mu.RLock()
		defer sm.mu.RUnlock()
		for _, v := range sm.cache {
			if !yieldFn(v) {
				return
			}
		}
	}
}
