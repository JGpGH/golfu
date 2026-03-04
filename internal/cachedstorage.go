package internal

import (
	"context"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/listop"
	"github.com/JGpGH/golfu/storage"
)

type CachedStorage[T element.Indexable] struct {
	IsEvictable bool
	InMemory    listop.IndexedList[T]
	Cold        storage.ColdStorage[T]
	MaxUnits    int
	Ctx         context.Context
	ToCold      chan []T
}

func (s *CachedStorage[T]) Start(ctx context.Context) {
	batchSize := max(s.MaxUnits/20, 10)
	// cache storing routine for non-blocking Set
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-s.ToCold:
				batch := in
				for range batchSize - 1 {
					select {
					case more := <-s.ToCold:
						batch = append(batch, more...)
					default:
						goto drain
					}
				}
			drain:
				s.Cold.Set(ctx, batch)
				currentLen := s.InMemory.Len()
				if currentLen > s.MaxUnits {
					s.evict(currentLen - s.MaxUnits + s.MaxUnits/5) // evict 20% of the cache + everything above max
				}
			}
		}
	}()
}

func (s *CachedStorage[T]) Set(ctx context.Context, values []T) error {
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
	if err != nil || fromCold == nil {
		return fromCold, err
	}

	s.InMemory.Set([]T{*fromCold})

	return fromCold, nil
}

func (s *CachedStorage[T]) Delete(ctx context.Context, indexes []string) error {
	if err := s.Cold.Delete(ctx, indexes); err != nil {
		return err
	}
	s.InMemory.Remove(indexes)
	return nil
}

func (s *CachedStorage[T]) evict(amount int) {
	if amount <= 0 {
		return
	}
	s.InMemory.SortByReadCount()
	if !s.IsEvictable {
		evicted := s.InMemory.Pop(amount)
		s.Cold.OnEviction(s.Ctx, evicted)
	} else {
		evicted := s.InMemory.PopWhere(func(t T) bool {
			return any(t).(element.Evictable).CanBeEvicted()
		}, amount)
		s.Cold.OnEviction(s.Ctx, evicted)
	}
	s.InMemory.ClearReadCounts()
}

func NewCachedStorage[T element.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) *CachedStorage[T] {
	var elementInspection T

	_, isEvictable := any(elementInspection).(element.Evictable)

	cache := &CachedStorage[T]{
		InMemory:    listop.NewIndexedList[T](),
		Cold:        cold,
		MaxUnits:    maxUnits,
		Ctx:         ctx,
		ToCold:      make(chan []T, max(maxUnits/20, 10)),
		IsEvictable: isEvictable,
	}

	cache.Start(ctx)

	return cache
}
