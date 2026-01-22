package storage

import (
	"context"

	"github.com/JGpGH/golfu/element"
)

type CachedStorage[T element.Indexable] interface {
	Set(ctx context.Context, values []T) error
	Get(ctx context.Context, index string) (*T, error)
	Gets(ctx context.Context, indexes []string) (map[string]T, error)
}
