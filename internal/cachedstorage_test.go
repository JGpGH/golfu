package internal_test

import (
	"context"
	"testing"
	"time"

	"github.com/JGpGH/golfu/internal"
	"github.com/JGpGH/golfu/storage"
)

type TestColdStorage[T storage.Indexable] struct {
	setted  chan T
	deleted chan T
	inner   map[string]T
}

func NewTestColdStorage[T storage.Indexable]() *TestColdStorage[T] {
	return &TestColdStorage[T]{
		setted:  make(chan T, 100),
		deleted: make(chan T, 100),
		inner:   make(map[string]T),
	}
}

func (tcs *TestColdStorage[T]) Set(ctx context.Context, values []storage.Trashable[T]) error {
	for _, in := range values {
		tcs.setted <- in.Value()
	}
	return nil
}

func (tcs *TestColdStorage[T]) Get(ctx context.Context, indexes []string) (map[string]T, error) {
	res := make(map[string]T)
	for _, key := range indexes {
		res[key] = tcs.inner[key]
	}
	return tcs.inner, nil
}

func (tcs *TestColdStorage[T]) Trash(ctx context.Context, values []T) error {
	for _, in := range values {
		tcs.deleted <- in
	}
	return nil
}

func (tcs *TestColdStorage[T]) CollectSetted(ctx context.Context, max int) []T {
	res := make([]T, 0)
	for {
		select {
		case <-ctx.Done():
			return res
		case in := <-tcs.setted:
			res = append(res, in)
			if len(res) >= max {
				return res
			}
		}
	}
}

func (tcs *TestColdStorage[T]) CollectDeleted(ctx context.Context, max int) []T {
	res := make([]T, 0)
	for {
		select {
		case <-ctx.Done():
			return res
		case in := <-tcs.deleted:
			res = append(res, in)
			if len(res) >= max {
				return res
			}
		}
	}
}

func TestStorageSetThenGet(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	}, nil)
	res, err := cache.Get(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	if res["1"].Value != 1 || res["2"].Value != 2 || res["3"].Value != 3 {
		t.Error("Get failed")
	}
}

func TestStorageEvictsSortedByRead(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
		storage.NewIndexed("4", 0),
	}, nil)
	_, err := cache.Get(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}

	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("5", 1),
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 1)
	if ctx.Err() != nil {
		t.Error(ctx.Err())
	}
	cancel()
	if len(r) != 1 {
		t.Errorf("Eviction failed expected 1 got %d", len(r))
	}
	for _, in := range r {
		if in.Value != 0 {
			t.Errorf("Eviction of higher ranked values detected: %d", in.Value)
		}
	}
}

func TestStorageEvictsSortedByRead2(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 0),
		storage.NewIndexed("4", 0),
	}, nil)
	_, err := cache.Get(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Error(err)
	}

	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("5", 1),
		storage.NewIndexed("6", 3),
	}, &storage.SetOptions{CanBeTrashed: false})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	r := cold.CollectDeleted(ctx, 5)
	cancel()
	if len(r) != 2 {
		t.Errorf("Eviction failed expected 2 got %d", len(r))
	}
	for _, in := range r {
		if in.Value != 0 {
			t.Errorf("Eviction of higher ranked values detected: %d with index %s", in.Value, in.Index())
		}
	}
}

func TestStorageEvictsOldest(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	}, nil)
	_, err := cache.Get(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("5", 0),
		storage.NewIndexed("6", 0),
	}, nil)
	_, err = cache.Get(context.Background(), []string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get(context.Background(), []string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 1)
	cancel()
	if len(r) != 1 {
		t.Error("Eviction failed")
	}
	for _, in := range r {
		if in.Value == 0 {
			t.Error("Eviction of newer values detected")
		}
	}
}

func TestStorageEvictsUntil20PercentUnderMax(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
		storage.NewIndexed("4", 4),
		storage.NewIndexed("5", 5),
		storage.NewIndexed("6", 6),
		storage.NewIndexed("7", 7),
		storage.NewIndexed("8", 8),
		storage.NewIndexed("9", 9),
		storage.NewIndexed("10", 10),
	}, nil)
	cache.Set(context.Background(), []storage.Indexed[int]{
		storage.NewIndexed("11", 11),
	}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 3)
	cancel()
	if len(r) != 3 {
		t.Error("Eviction of 20% under max failed")
	}
}
