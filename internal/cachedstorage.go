package internal

import (
	"context"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/listop"
	"github.com/JGpGH/golfu/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type cachedStorage[T element.Indexable] struct {
	isTrashable *bool
	inMemory    listop.IndexedList[T]
	cold        storage.ColdStorage[T]
	maxUnits    int
	ctx         context.Context
	toCold      chan []T
	tracer      trace.Tracer
}

type NoopTrash[T element.Indexable] struct{}

func (s *cachedStorage[T]) Start(ctx context.Context) {
	// cache storing routine for non-blocking Set
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-s.toCold:
				s.cold.Set(ctx, in)
				currentLen := s.inMemory.Len()
				if currentLen > s.maxUnits {
					s.evict(currentLen - s.maxUnits + s.maxUnits/5) // evict 20% of the cache + everything above max
				}
			}
		}
	}()
}

func (s *cachedStorage[T]) Set(ctx context.Context, values []T) error {
	ctx, span := s.tracer.Start(ctx, "cachedelement.Set")
	defer span.End()
	if s.isTrashable == nil && len(values) > 0 {
		_, ok := any(values[0]).(element.Trashable)
		s.isTrashable = &ok
	}
	span.SetAttributes(attribute.Int(ItemLengthAttribute, len(values)))
	s.inMemory.Set(values)
	s.toCold <- values
	return nil
}

func (s *cachedStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	ctx, span := s.tracer.Start(ctx, "cachedelement.Get")
	defer span.End()
	span.SetAttributes(attribute.Int(ItemLengthAttribute, len(indexes)))
	var result = make(map[string]T)
	var inMemoryHits int
	var toFetch []string
	cached := s.inMemory.Gets(indexes)
	for _, c := range indexes {
		if u, ok := cached[c]; ok {
			result[c] = u
			inMemoryHits++
		} else {
			toFetch = append(toFetch, c)
		}
	}
	span.SetAttributes(attribute.Int(InMemoryHitsAttribute, inMemoryHits))

	if len(toFetch) == 0 {
		return result, nil
	}

	fromCold, err := s.cold.Gets(ctx, toFetch)
	if err != nil {
		span.RecordError(err)
	}

	toCache := make([]T, 0, len(fromCold))
	for k := range fromCold {
		result[k] = fromCold[k]
		toCache = append(toCache, fromCold[k])
	}

	s.inMemory.Set(toCache)

	span.SetAttributes(attribute.Int(ColdHitsAttribute, len(fromCold)))

	return result, err
}

func (s *cachedStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	cached := s.inMemory.Get(index)
	if cached != nil {
		return cached, nil
	}

	fromCold, err := s.cold.Get(ctx, index)
	if err != nil {
		return nil, err
	}

	s.inMemory.Set([]T{*fromCold})

	return fromCold, nil
}

func (s *cachedStorage[T]) evict(amount int) {
	if amount <= 0 {
		return
	}
	s.inMemory.SortByReadCount()
	if s.isTrashable == nil || !*s.isTrashable {
		trashed := s.inMemory.Pop(amount)
		s.cold.Trash(s.ctx, trashed)
	} else {
		trashed := s.inMemory.PopWhere(func(t T) bool {
			return any(t).(element.Trashable).CanBeTrashed()
		}, amount)
		s.cold.Trash(s.ctx, trashed)
	}
	s.inMemory.ClearReadCounts()
}

func (n *NoopTrash[T]) Trash(ctx context.Context, values []T) error {
	return nil
}

func NewCachedStorage[T element.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) storage.CachedStorage[T] {
	cache := &cachedStorage[T]{
		inMemory: listop.NewIndexedList[T](),
		cold:     cold,
		maxUnits: maxUnits,
		ctx:      ctx,
		tracer:   otel.GetTracerProvider().Tracer(TracerName),
		toCold:   make(chan []T, maxUnits),
	}

	cache.Start(ctx)

	return cache
}
