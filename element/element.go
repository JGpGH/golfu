package element

type Indexable interface {
	Index() string
}

type Indexed[T any] struct {
	index string
	Value T
}

type EvictableIndexed[T any] struct {
	Indexed[T]
	canBeEvicted bool
}

func (t *EvictableIndexed[T]) CanBeEvicted() bool {
	return t.canBeEvicted
}

type Evictable interface {
	CanBeEvicted() bool
}

func (i Indexed[T]) Index() string {
	return i.index
}

func NewIndexed[T any](index string, value T) Indexed[T] {
	return Indexed[T]{index: index, Value: value}
}

func NewEvictableIndexed[T any](index string, value T, canBeEvicted bool) *EvictableIndexed[T] {
	return &EvictableIndexed[T]{
		Indexed:      Indexed[T]{index: index, Value: value},
		canBeEvicted: canBeEvicted,
	}
}
