package internal

import (
	"context"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/listop"
	"github.com/JGpGH/golfu/storage"
	"go.opentelemetry.io/otel/trace"
)

type CachedStorage[T element.Indexable] struct {
	IsTrashable *bool
	InMemory    listop.IndexedList[T]
	Cold        storage.ColdStorage[T]
	MaxUnits    int
	Ctx         context.Context
	ToCold      chan []T
	Tracer      trace.Tracer
}

func (s *CachedStorage[T]) Start(ctx context.Context) {
	// cache storing routine for non-blocking Set
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-s.ToCold:
				s.Cold.Set(ctx, in)
				currentLen := s.InMemory.Len()
				if currentLen > s.MaxUnits {
					s.evict(currentLen - s.MaxUnits + s.MaxUnits/5) // evict 20% of the cache + everything above max
				}
			}
		}
	}()
}

func (s *CachedStorage[T]) Set(ctx context.Context, values []T) error {
	if s.IsTrashable == nil && len(values) > 0 {
		_, ok := any(values[0]).(element.Trashable)
		s.IsTrashable = &ok
	}
	s.InMemory.Set(values)
	s.ToCold <- values
	return nil
}

func (s *CachedStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	var result = make(map[string]T)
	var toFetch []string
	cached := s.InMemory.Gets(indexes)
	for _, c := range indexes {
		if u, ok := cached[c]; ok {
			result[c] = u
		} else {
			toFetch = append(toFetch, c)
		}
	}

	if len(toFetch) == 0 {
		return result, nil
	}

	fromCold, err := s.Cold.Gets(ctx, toFetch)

	toCache := make([]T, 0, len(fromCold))
	for k := range fromCold {
		result[k] = fromCold[k]
		toCache = append(toCache, fromCold[k])
	}

	s.InMemory.Set(toCache)

	return result, err
}

func (s *CachedStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	cached := s.InMemory.Get(index)
	if cached != nil {
		return cached, nil
	}

	fromCold, err := s.Cold.Get(ctx, index)
	if err != nil {
		return nil, err
	}

	s.InMemory.Set([]T{*fromCold})

	return fromCold, nil
}

func (s *CachedStorage[T]) evict(amount int) {
	if amount <= 0 {
		return
	}
	s.InMemory.SortByReadCount()
	if s.IsTrashable == nil || !*s.IsTrashable {
		trashed := s.InMemory.Pop(amount)
		s.Cold.Trash(s.Ctx, trashed)
	} else {
		trashed := s.InMemory.PopWhere(func(t T) bool {
			return any(t).(element.Trashable).CanBeTrashed()
		}, amount)
		s.Cold.Trash(s.Ctx, trashed)
	}
	s.InMemory.ClearReadCounts()
}
