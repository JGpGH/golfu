package golfu

import (
	"context"

	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

func NewCachedStorage[T storage.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) storage.CachedStorage[T] {
	return internal.NewCachedStorage(ctx, cold, maxUnits)
}

func NewNoopTrash[T storage.Indexable]() storage.Trash[T] {
	return &internal.NoopTrash[T]{}
}
