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

type Deletable interface {
	CanBeDeleted() bool
}

type DeletableIndexed[T any] struct {
	Indexed[T]
	canBeDeleted atomic.Bool
}

func NewDeletableIndexed[T any](index string, value T, canBeDeleted bool) *DeletableIndexed[T] {
	di := &DeletableIndexed[T]{
		Indexed[T]{index: index, Value: value},
		atomic.Bool{},
	}
	di.canBeDeleted.Store(canBeDeleted)
	return di
}

func (d *DeletableIndexed[T]) CanBeDeleted() bool {
	return d.canBeDeleted.Load()
}

func (d *DeletableIndexed[T]) SetCanBeDeleted(value bool) {
	d.canBeDeleted.Store(value)
}
