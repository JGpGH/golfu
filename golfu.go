package golfu

import (
	"context"

	"github.com/JGpGH/golfu/element"
	lfuErr "github.com/JGpGH/golfu/errors"
	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

var (
	ErrNotFound     = lfuErr.ErrNotFound
	ErrInvalidIndex = lfuErr.ErrInvalidIndex
)

func NewCachedStorage[T element.Indexable](ctx context.Context, cold storage.ColdStorage[T], maxUnits int) storage.CachedStorage[T] {
	return internal.NewCachedStorage(ctx, cold, maxUnits)
}
