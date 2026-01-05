package internal

import (
	"context"

	"github.com/JGpGH/golfu/internal/listop"
	"github.com/JGpGH/golfu/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type cachedStorage[T storage.Indexable] struct {
	units    listop.IndexedList[storage.Trashable[T]]
	cold     storage.ColdStorage[T]
	maxUnits int
	trash    storage.Trash[T]
	ctx      context.Context
	toCold   chan []storage.Trashable[T]
	tracer   trace.Tracer
}

type NoopTrash[T storage.Indexable] struct{}

func (s *cachedStorage[T]) Start(ctx context.Context) {
	// cache storing routine for non-blocking Set
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-s.toCold:
				s.cold.Set(ctx, in)
				currentLen := s.units.Len()
				if currentLen > s.maxUnits {
					s.evict(currentLen - s.maxUnits + s.maxUnits/5) // evict 20% of the cache + everything above max
				}
			}
		}
	}()
}

func DefaultSetOptions() *storage.SetOptions {
	return &storage.SetOptions{CanBeTrashed: true}
}

func (s *cachedStorage[T]) Set(ctx context.Context, values []T, options *storage.SetOptions) {
	ctx, span := s.tracer.Start(ctx, "cachedStorage.Set")
	defer span.End()
	span.SetAttributes(attribute.Int(ItemLengthAttribute, len(values)))

	if options == nil {
		options = DefaultSetOptions()
	}
	trashables := storage.NewTrashables(values, options.CanBeTrashed)
	s.units.Set(trashables)
	s.toCold <- trashables
}

func (s *cachedStorage[T]) Get(ctx context.Context, indexes []string) (map[string]T, error) {
	ctx, span := s.tracer.Start(ctx, "cachedStorage.Get")
	defer span.End()
	span.SetAttributes(attribute.Int(ItemLengthAttribute, len(indexes)))
	var result = make(map[string]T)
	var inMemoryHits int
	var toFetch []string
	cached := s.units.Get(indexes)
	for _, c := range indexes {
		if u, ok := cached[c]; ok {
			result[c] = u.Value()
			inMemoryHits++
		} else {
			toFetch = append(toFetch, c)
		}
	}
	span.SetAttributes(attribute.Int(InMemoryHitsAttribute, inMemoryHits))

	if len(toFetch) == 0 {
		return result, nil
	}

	fromCold, err := s.cold.Get(ctx, toFetch)
	if err != nil {
		span.RecordError(err)
	}

	toCache := make([]T, 0, len(fromCold))
	for k := range fromCold {
		result[k] = fromCold[k]
		toCache = append(toCache, fromCold[k])
	}

	s.units.Set(storage.NewTrashables(toCache, true))

	span.SetAttributes(attribute.Int(ColdHitsAttribute, len(fromCold)))

	totalHits := inMemoryHits + len(fromCold)
	if totalHits == 0 {
		span.SetStatus(codes.Error, Error)
	} else if totalHits < len(indexes) {
		span.SetStatus(codes.Ok, PartialSuccess)
	} else if totalHits > len(indexes) {
		span.SetStatus(codes.Error, TooManyItemsMessage)
	} else {
		span.SetStatus(codes.Ok, Success)
	}

	return result, err
}

func (s *cachedStorage[T]) evict(amount int) {
	if amount <= 0 {
		return
	}
	s.units.SortByReadCount()
	trashed := s.units.PopWhere(func(u storage.Trashable[T]) bool {
		return u.CanBeTrashed()
	}, amount)
	var trashedValues []T
	for _, t := range trashed {
		trashedValues = append(trashedValues, t.Value())
	}
	s.trash.Trash(s.ctx, trashedValues)
	s.units.ClearReadCounts()
}

func (n *NoopTrash[T]) Trash(ctx context.Context, values []T) error {
	return nil
}

func NewCachedStorage[T storage.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) storage.CachedStorage[T] {
	coldTrash, ok := cold.(storage.Trash[T])
	if !ok {
		coldTrash = &NoopTrash[T]{}
	}

	cache := &cachedStorage[T]{
		units:    listop.NewIndexedList[storage.Trashable[T]](),
		cold:     cold,
		maxUnits: maxUnits,
		trash:    coldTrash,
		ctx:      ctx,
		tracer:   otel.GetTracerProvider().Tracer(TracerName),
		toCold:   make(chan []storage.Trashable[T], maxUnits),
	}

	cache.Start(ctx)

	return cache
}
