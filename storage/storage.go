package storage

type Indexable interface {
	Index() string
}

type Readonly[T any] interface {
	Read() T
}

func Collect[T any](src []Readonly[T]) []T {
	var result []T
	for _, r := range src {
		result = append(result, r.Read())
	}
	return result
}

type ColdStorage[T Indexable] interface {
	Set([]Readonly[T]) error
	Get([]string) (map[string]T, error)
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
	Trash([]T)
}

type CachedStorage[T Indexable] interface {
	Set([]T)
	Get([]string) (map[string]T, error)
	Sync() chan error
}
