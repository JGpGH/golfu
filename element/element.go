package element

import "sync/atomic"

type Indexable interface {
	Index() string
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

type Evictable interface {
	CanBeEvicted() bool
}

type EvictableIndexed[T any] struct {
	Indexed[T]
	canBeEvicted atomic.Bool
}

func (t *EvictableIndexed[T]) CanBeEvicted() bool {
	return t.canBeEvicted.Load()
}

func (t *EvictableIndexed[T]) SetCanBeEvicted(value bool) {
	t.canBeEvicted.Store(value)
}

func NewEvictableIndexed[T any](index string, value T, canBeEvicted bool) *EvictableIndexed[T] {
	ei := &EvictableIndexed[T]{
		Indexed:      Indexed[T]{index: index, Value: value},
		canBeEvicted: atomic.Bool{},
	}
	ei.canBeEvicted.Store(canBeEvicted)
	return ei
}
