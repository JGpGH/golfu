package element

type Indexable interface {
	Index() string
}

type Indexed[T any] struct {
	index string
	Value T
}

type TrashableIndexed[T any] struct {
	Indexed[T]
	canBeTrashed bool
}

func (t *TrashableIndexed[T]) CanBeTrashed() bool {
	return t.canBeTrashed
}

type Trashable interface {
	CanBeTrashed() bool
}

func (i Indexed[T]) Index() string {
	return i.index
}

func NewIndexed[T any](index string, value T) Indexed[T] {
	return Indexed[T]{index: index, Value: value}
}
