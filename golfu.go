package golfu

import (
	"context"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

func NewCachedStorage[T element.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) storage.CachedStorage[T] {
	return internal.NewCachedStorage(ctx, cold, maxUnits)
}
