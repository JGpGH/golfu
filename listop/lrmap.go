package listop

import (
	"sync"
	"sync/atomic"

	"github.com/JGpGH/golfu/element"
)

type side[T element.Indexable] struct {
	mu   sync.RWMutex
	data map[string]T
}

// LRMap is a left-right concurrent map. Readers are lock-free (RLock).
// Writers are serialized by a mutex.
type LRMap[T element.Indexable] struct {
	read    atomic.Pointer[side[T]]
	write   *side[T]
	applyMu sync.Mutex
}

func NewLRMap[T element.Indexable]() *LRMap[T] {
	lr := &LRMap[T]{
		write: &side[T]{data: make(map[string]T)},
	}
	r := &side[T]{data: make(map[string]T)}
	lr.read.Store(r)
	return lr
}

// Get returns a copy of the value for the given key. Safe from any goroutine.
func (lr *LRMap[T]) Get(key string) (T, bool) {
	r := lr.read.Load()
	r.mu.RLock()
	val, ok := r.data[key]
	r.mu.RUnlock()
	return val, ok
}

// Gets returns values for the given keys. Safe from any goroutine.
func (lr *LRMap[T]) Gets(keys []string) map[string]T {
	r := lr.read.Load()
	r.mu.RLock()
	result := make(map[string]T, len(keys))
	for _, key := range keys {
		if val, ok := r.data[key]; ok {
			result[key] = val
		}
	}
	r.mu.RUnlock()
	return result
}

// Len returns the number of items. Safe from any goroutine.
func (lr *LRMap[T]) Len() int {
	r := lr.read.Load()
	r.mu.RLock()
	n := len(r.data)
	r.mu.RUnlock()
	return n
}

// Set inserts or updates items.
func (lr *LRMap[T]) Set(items []T) {
	lr.apply(func(m map[string]T) {
		for _, item := range items {
			m[item.Index()] = item
		}
	})
}

// Remove deletes keys.
func (lr *LRMap[T]) Remove(keys []string) {
	lr.apply(func(m map[string]T) {
		for _, key := range keys {
			delete(m, key)
		}
	})
}

// apply runs fn on the write side, swaps, waits for readers to drain, then runs fn on the old read side.
func (lr *LRMap[T]) apply(fn func(m map[string]T)) {
	lr.applyMu.Lock()
	defer lr.applyMu.Unlock()

	lr.write.mu.Lock()
	fn(lr.write.data)
	lr.write.mu.Unlock()

	lr.write = lr.read.Swap(lr.write)

	lr.write.mu.Lock()
	fn(lr.write.data)
	lr.write.mu.Unlock()
}
