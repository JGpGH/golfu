package coldstorages

import (
	"context"
	"sync"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// InMemoryStorage provides a thread-safe in-memory storage implementation.
type InMemoryStorage[T element.Indexable] struct {
	mu   sync.RWMutex
	data map[string]T
}

// NewInMemoryStorage creates a new in-memory storage instance.
func NewInMemoryStorage[T element.Indexable]() *InMemoryStorage[T] {
	return &InMemoryStorage[T]{
		data: make(map[string]T),
	}
}

// Set stores multiple elements in memory.
func (ims *InMemoryStorage[T]) Set(ctx context.Context, values []T) error {
	ctx, span := otel.GetTracerProvider().Tracer("InMemoryStorage").Start(ctx, "InMemoryStorage.Set")
	defer span.End()

	ims.mu.Lock()
	defer ims.mu.Unlock()

	for _, element := range values {
		if ctx.Err() != nil {
			span.RecordError(ctx.Err())
			return ctx.Err()
		}
		index := element.Index()
		if index == "" {
			err := errors.ErrInvalidIndex
			span.RecordError(err)
			return err
		}
		ims.data[index] = element
	}
	return nil
}

// Get retrieves a single element by index.
func (ims *InMemoryStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("InMemoryStorage").Start(ctx, "InMemoryStorage.Get")
	defer span.End()
	span.SetAttributes(attribute.String("index", index))

	if ctx.Err() != nil {
		span.RecordError(ctx.Err())
		return nil, ctx.Err()
	}

	if index == "" {
		err := errors.ErrInvalidIndex
		span.RecordError(err)
		return nil, err
	}

	ims.mu.RLock()
	defer ims.mu.RUnlock()

	element, ok := ims.data[index]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return &element, nil
}

// Gets retrieves multiple elements by their indexes.
func (ims *InMemoryStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	ctx, span := otel.GetTracerProvider().Tracer("InMemoryStorage").Start(ctx, "InMemoryStorage.Gets")
	defer span.End()

	if ctx.Err() != nil {
		span.RecordError(ctx.Err())
		return nil, ctx.Err()
	}

	ims.mu.RLock()
	defer ims.mu.RUnlock()

	result := make(map[string]T)
	for _, index := range indexes {
		if ctx.Err() != nil {
			span.RecordError(ctx.Err())
			return nil, ctx.Err()
		}
		if index == "" {
			continue
		}
		if element, ok := ims.data[index]; ok {
			result[index] = element
		}
	}
	return result, nil
}

// Delete removes elements by their indexes.
func (ims *InMemoryStorage[T]) Delete(ctx context.Context, indexes []string) error {
	ctx, span := otel.GetTracerProvider().Tracer("InMemoryStorage").Start(ctx, "InMemoryStorage.Delete")
	defer span.End()

	ims.mu.Lock()
	defer ims.mu.Unlock()

	for _, index := range indexes {
		if ctx.Err() != nil {
			span.RecordError(ctx.Err())
			return ctx.Err()
		}
		delete(ims.data, index)
	}
	return nil
}

// Clear removes all elements from storage.
func (ims *InMemoryStorage[T]) Clear() {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	ims.data = make(map[string]T)
}

// Size returns the number of elements in storage.
func (ims *InMemoryStorage[T]) Size() int {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return len(ims.data)
}

// Has checks if an element exists by index.
func (ims *InMemoryStorage[T]) Has(index string) bool {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	_, ok := ims.data[index]
	return ok
}

// Keys returns all indexes currently in storage.
func (ims *InMemoryStorage[T]) Keys() []string {
	ims.mu.RLock()
	defer ims.mu.RUnlock()

	keys := make([]string, 0, len(ims.data))
	for key := range ims.data {
		keys = append(keys, key)
	}
	return keys
}

// Trash removes elements by treating them like Delete (for ColdStorage interface).
func (ims *InMemoryStorage[T]) Trash(ctx context.Context, values []T) error {
	indexes := make([]string, len(values))
	for i, v := range values {
		indexes[i] = v.Index()
	}
	return ims.Delete(ctx, indexes)
}
