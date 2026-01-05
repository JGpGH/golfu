package storage

import (
	"context"
	"sync/atomic"
)

type Indexable interface {
	Index() string
}

type Trashable[T Indexable] interface {
	Value() T
	CanBeTrashed() bool
	SetCanBeTrashed()
	Index() string
}

type trashable[T Indexable] struct {
	value        T
	canBeTrashed atomic.Bool
}

func (t *trashable[T]) Value() T {
	return t.value
}

func (t *trashable[T]) Index() string {
	return t.value.Index()
}

func (t *trashable[T]) CanBeTrashed() bool {
	return t.canBeTrashed.Load()
}

func (t *trashable[T]) SetCanBeTrashed() {
	t.canBeTrashed.Store(true)
}

func NewTrashable[T Indexable](value T, canBeTrashed bool) Trashable[T] {
	t := &trashable[T]{value: value}
	t.canBeTrashed.Store(canBeTrashed)
	return t
}

func NewTrashables[T Indexable](values []T, canBeTrashed bool) []Trashable[T] {
	result := make([]Trashable[T], len(values))
	for k := range values {
		result = append(result, NewTrashable(values[k], canBeTrashed))
	}
	return result
}

type ColdStorage[T Indexable] interface {
	Set(ctx context.Context, values []Trashable[T]) error
	Get(ctx context.Context, indexes []string) (map[string]T, error)
}

type Indexed[T any] struct {
	index string
	Value T
}

func (i Indexed[T]) Index() string {
	return i.index
}

func NewIndexed[T any](index string, value T) Indexed[T] {
	return Indexed[T]{index: index, Value: value}
}

type Trash[T Indexable] interface {
	Trash(ctx context.Context, values []T) error
}

type SetOptions struct {
	CanBeTrashed bool
}

type CachedStorage[T Indexable] interface {
	Set(ctx context.Context, values []T, options *SetOptions)
	Get(ctx context.Context, indexes []string) (map[string]T, error)
}
