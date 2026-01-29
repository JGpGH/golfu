package coldstorages

import (
	"context"
	"sync"

	"github.com/JGpGH/golfu/element"
	lfuErr "github.com/JGpGH/golfu/errors"
)

// for tests
type InMemoryStorage[T element.Indexable] struct {
	data map[string]T
	sync.RWMutex
}

func NewInMemoryStorage[T element.Indexable]() *InMemoryStorage[T] {
	return &InMemoryStorage[T]{
		data: make(map[string]T),
	}
}

func (s *InMemoryStorage[T]) Set(ctx context.Context, values []T) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range values {
		s.data[v.Index()] = v
	}
	return nil
}

func (s *InMemoryStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	s.RLock()
	defer s.RUnlock()
	value, exists := s.data[index]
	if !exists {
		return nil, lfuErr.ErrNotFound
	}
	return &value, nil
}

func (s *InMemoryStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	s.RLock()
	defer s.RUnlock()
	result := make(map[string]T)
	for _, index := range indexes {
		if value, exists := s.data[index]; exists {
			result[index] = value
		}
	}
	return result, nil
}

func (s *InMemoryStorage[T]) Delete(ctx context.Context, indexes []string) error {
	s.Lock()
	defer s.Unlock()
	for _, index := range indexes {
		delete(s.data, index)
	}
	return nil
}

func (s *InMemoryStorage[T]) OnEviction(ctx context.Context, values []T) error {
	return nil
}
