package internal_test

import (
	"context"
	"testing"
	"time"

	"github.com/JGpGH/golfu/element"
	"github.com/JGpGH/golfu/internal"
)

type TestColdStorage[T element.Indexable] struct {
	setted  chan T
	deleted chan T
	inner   map[string]T
}

func NewTestColdStorage[T element.Indexable]() *TestColdStorage[T] {
	return &TestColdStorage[T]{
		setted:  make(chan T, 100),
		deleted: make(chan T, 100),
		inner:   make(map[string]T),
	}
}

func (tcs *TestColdStorage[T]) Set(ctx context.Context, values []T) error {
	for _, in := range values {
		tcs.setted <- in
	}
	return nil
}

func (tcs *TestColdStorage[T]) Gets(ctx context.Context, indexes []string) (map[string]T, error) {
	res := make(map[string]T)
	for _, key := range indexes {
		res[key] = tcs.inner[key]
	}
	return res, nil
}

func (tcs *TestColdStorage[T]) Get(ctx context.Context, index string) (*T, error) {
	res, ok := tcs.inner[index]
	if !ok {
		return nil, nil
	}
	return &res, nil
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
	cold := NewTestColdStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 1),
		element.NewIndexed("2", 2),
		element.NewIndexed("3", 3),
	})
	res, err := cache.Gets(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	if res["1"].Value != 1 || res["2"].Value != 2 || res["3"].Value != 3 {
		t.Error("Get failed")
	}
}

func TestStorageEvictsSortedByRead(t *testing.T) {
	cold := NewTestColdStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 1),
		element.NewIndexed("2", 2),
		element.NewIndexed("3", 3),
		element.NewIndexed("4", 0),
	})
	_, err := cache.Gets(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}

	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("5", 1),
	})

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
	cold := NewTestColdStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 1),
		element.NewIndexed("2", 2),
		element.NewIndexed("3", 1),
		element.NewIndexed("4", 0),
	})
	_, err := cache.Gets(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Gets(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Error(err)
	}

	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("5", 0),
		element.NewIndexed("6", 0),
	})

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
	cold := NewTestColdStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 1),
		element.NewIndexed("2", 2),
		element.NewIndexed("3", 3),
	})
	_, err := cache.Gets(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Gets(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("5", 0),
		element.NewIndexed("6", 0),
	})
	_, err = cache.Gets(context.Background(), []string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Gets(context.Background(), []string{"5", "6"})
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
	cold := NewTestColdStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 1),
		element.NewIndexed("2", 2),
		element.NewIndexed("3", 3),
		element.NewIndexed("4", 4),
		element.NewIndexed("5", 5),
		element.NewIndexed("6", 6),
		element.NewIndexed("7", 7),
		element.NewIndexed("8", 8),
		element.NewIndexed("9", 9),
		element.NewIndexed("10", 10),
	})
	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("11", 11),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 3)
	cancel()
	if len(r) != 3 {
		t.Error("Eviction of 20% under max failed")
	}
}

type TrashableIndexed struct {
	element.Indexed[int]
	canBeTrashed bool
}

func (u *TrashableIndexed) Index() string {
	return u.Indexed.Index()
}

func (u *TrashableIndexed) CanBeTrashed() bool {
	return u.canBeTrashed
}

func TestStorageDoesNotTrashUntrashable(t *testing.T) {
	cold := NewTestColdStorage[*TrashableIndexed]()
	cache := internal.NewCachedStorage(context.Background(), cold, 4)
	cache.Set(context.Background(), []*TrashableIndexed{
		{Indexed: element.NewIndexed("1", 1), canBeTrashed: false},
		{Indexed: element.NewIndexed("2", 2), canBeTrashed: false},
		{Indexed: element.NewIndexed("3", 3), canBeTrashed: true},
		{Indexed: element.NewIndexed("4", 4), canBeTrashed: false},
	})
	cache.Set(context.Background(), []*TrashableIndexed{
		{Indexed: element.NewIndexed("5", 5), canBeTrashed: false},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 1)
	cancel()
	if len(r) != 1 {
		t.Errorf("Expected 1 trashed items got %d", len(r))
	}
	if r[0].Index() != "3" {
		t.Errorf("Wrong item trashed expected 3 got %s", r[0].Index())
	}
}
