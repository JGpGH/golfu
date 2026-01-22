package storage

import (
	"context"

	"github.com/JGpGH/golfu/element"
)

type ColdStorage[T element.Indexable] interface {
	Set(ctx context.Context, values []T) error
	Trash(ctx context.Context, values []T) error
	Get(ctx context.Context, index string) (*T, error)
	Gets(ctx context.Context, indexes []string) (map[string]T, error)
}
