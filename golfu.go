package golfu

import (
	"context"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/listop"
	"github.com/JGpGH/golfu/storage"
	"go.opentelemetry.io/otel"
)

func NewCachedStorage[T element.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) storage.CachedStorage[T] {
	cache := &internal.CachedStorage[T]{
		InMemory: listop.NewIndexedList[T](),
		Cold:     cold,
		MaxUnits: maxUnits,
		Ctx:      ctx,
		Tracer:   otel.GetTracerProvider().Tracer(internal.TracerName),
		ToCold:   make(chan []T, maxUnits),
	}

	cache.Start(ctx)

	return cache
}
