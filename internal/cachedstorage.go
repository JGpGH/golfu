package internal

import (
	"context"

	"github.com/JGpGH/golfu/internal/listop"
	"github.com/JGpGH/golfu/storage"
)

func (s *cachedStorage[T]) Start(ctx context.Context, trash storage.Trash[T]) {
	// cache storing routine for non-blocking Set
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-s.toCold:
				for _, u := range in {
					u.SetPersisted()
				}
				s.cold.Set(asReadOnlyUnits(in))
				s.units.Set(in)
				currentLen := s.units.Len()
				if currentLen > s.maxUnits {
					evicted := s.evict(currentLen - s.maxUnits + s.maxUnits/5) // evict 20% of the cache + everything above max
					trash.Trash(evicted)
				}
			}
		}
	}()
}

func (s *cachedStorage[T]) Set(values []T) {
	var toCache []persistable[T]
	for _, v := range values {
		toCache = append(toCache, persistable[T]{value: v, isPersisted: false})
	}
	units := toUnits(toCache)
	s.units.Set(units)
	s.toCold <- units
}

func (s *cachedStorage[T]) Get(indexes []string) (map[string]T, error) {
	var result = make(map[string]T)
	var toFetch []string
	cached := s.units.Get(indexes)
	for _, c := range indexes {
		if u, ok := cached[c]; ok {
			result[c] = u.Read()
		} else {
			toFetch = append(toFetch, c)
		}
	}

	if len(toFetch) == 0 {
		return result, nil
	}

	persisted, err := s.cold.Get(toFetch)
	if err != nil {
		return nil, err
	}

	toCache := make([]T, len(persisted))
	for k, v := range persisted {
		result[k] = v
		toCache = append(toCache, v)
	}

	s.Set(toCache)
	return result, nil
}

func (s *cachedStorage[T]) evict(amount int) []T {
	if amount <= 0 {
		return []T{}
	}
	s.units.SortByReadCount()
	trashed := s.units.PopWhere(func(u *unit[T]) bool {
		return u.IsPersisted()
	}, amount)
	s.units.ClearReadCounts()
	return values(trashed)
}

type cachedStorage[T storage.Indexable] struct {
	units    listop.IndexedList[*unit[T]]
	cold     storage.ColdStorage[T]
	maxUnits int
	ctx      context.Context
	toCold   chan []*unit[T]
}

func NewCachedStorage[T storage.Indexable](ctx context.Context, cold storage.ColdStorage[T], trash storage.Trash[T], maxUnits int) storage.CachedStorage[T] {
	cache := &cachedStorage[T]{
		units:    listop.NewIndexedList[*unit[T]](),
		cold:     cold,
		maxUnits: maxUnits,
		ctx:      ctx,
		toCold:   make(chan []*unit[T], maxUnits),
	}
	cache.Start(ctx, trash)
	return cache
}
