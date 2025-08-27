package internal_test

import (
	"context"
	"strconv"
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

func (tcs *TestColdStorage[T]) Set(ins []storage.Readonly[T]) error {
	for _, in := range ins {
		tcs.setted <- in.Read()
	}
	return nil
}

func (tcs *TestColdStorage[T]) Get(keys []string) (map[string]T, error) {
	res := make(map[string]T)
	for _, key := range keys {
		res[key] = tcs.inner[key]
	}
	return tcs.inner, nil
}

func (tcs *TestColdStorage[T]) Trash(ins []T) {
	for _, in := range ins {
		tcs.deleted <- in
	}
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

func TestStorageGetThenSet(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, cold, 10)
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 3)
	cancel()
	res, err := cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	if res["1"].Value != 1 || res["2"].Value != 2 || res["3"].Value != 3 {
		t.Error("Get failed")
	}
}

func TestStorageEvictsSortedByRead(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, cold, 4)
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 3)
	cancel()
	_, err := cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("5", 0),
		storage.NewIndexed("6", 0),
	})
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 1)
	cancel()
	if len(r) != 1 {
		t.Error("Eviction failed")
	}
	for _, in := range r {
		if in.Value != 0 {
			t.Error("Eviction of higher ranked values detected")
		}
	}
}

func TestStorageEvictsOldest(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, cold, 4)
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
		storage.NewIndexed("2", 2),
		storage.NewIndexed("3", 3),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 3)
	cancel()
	_, err := cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get([]string{"1", "2", "3"})
	if err != nil {
		t.Error(err)
	}
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("5", 0),
		storage.NewIndexed("6", 0),
	})
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 2)
	cancel()
	_, err = cache.Get([]string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	_, err = cache.Get([]string{"5", "6"})
	if err != nil {
		t.Error(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
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
	cache := internal.NewCachedStorage(context.Background(), cold, cold, 10)
	cache.Set([]storage.Indexed[int]{
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
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cold.CollectSetted(ctx, 10)
	cancel()
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("11", 11),
	})
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	r := cold.CollectDeleted(ctx, 3)
	cancel()
	if len(r) != 3 {
		t.Error("Eviction of 20% under max failed")
	}
}

func TestStorageSync(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cache := internal.NewCachedStorage(ctx, cold, cold, 10)
	cache.Set([]storage.Indexed[int]{
		storage.NewIndexed("1", 1),
	})
	// Sync should block until the write is complete and return nil error
	err := <-cache.Sync()
	if err != nil {
		t.Errorf("Sync returned error: %v", err)
	}
	// Get should return the cached value
	res, err := cache.Get([]string{"1"})
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if res["1"].Value != 1 {
		t.Error("Get failed")
	}
}

func TestStorageSync_ContextCancel(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	ctx, cancel := context.WithCancel(context.Background())
	cache := internal.NewCachedStorage(ctx, cold, cold, 10)
	cancel() // cancel immediately
	cache.Set([]storage.Indexed[int]{storage.NewIndexed("2", 2)})
	select {
	case err := <-cache.Sync():
		if err == nil {
			t.Error("Sync should return error when context is canceled")
		}
	case <-time.After(1 * time.Second):
		t.Error("Sync did not return in time after context cancel")
	}
}

func TestStorageSync_MultipleSets(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, cold, 10)
	cache.Set([]storage.Indexed[int]{storage.NewIndexed("3", 3)})
	cache.Set([]storage.Indexed[int]{storage.NewIndexed("4", 4)})
	for i := 0; i < 2; i++ {
		select {
		case err := <-cache.Sync():
			if err != nil {
				t.Errorf("Sync returned error on write %d: %v", i+1, err)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Sync did not return in time for write %d", i+1)
		}
	}
}

// slowColdStorage is a ColdStorage that adds delay to Set for sync blocking tests
type slowColdStorage struct {
	TestColdStorage[storage.Indexed[int]]
	delay time.Duration
}

func (s *slowColdStorage) Set(ins []storage.Readonly[storage.Indexed[int]]) error {
	time.Sleep(s.delay)
	return s.TestColdStorage.Set(ins)
}

func TestStorageSync_BlocksUntilWrite(t *testing.T) {
	slow := &slowColdStorage{TestColdStorage: *NewTestColdStorage[storage.Indexed[int]](), delay: 300 * time.Millisecond}
	cache := internal.NewCachedStorage(context.Background(), slow, slow, 10)
	cache.Set([]storage.Indexed[int]{storage.NewIndexed("5", 5)})
	done := make(chan struct{})
	go func() {
		<-cache.Sync()
		close(done)
	}()
	select {
	case <-done:
		t.Error("Sync returned before Set completed (should block)")
	case <-time.After(150 * time.Millisecond):
		// expected: still blocked
	}
	// Now let enough time pass for Set to finish
	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Error("Sync did not return after Set completed")
	}
}

func TestStorageSync_ConcurrentSetSync(t *testing.T) {
	cold := NewTestColdStorage[storage.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, cold, 10)
	done := make(chan error, 10)
	for i := 0; i < 5; i++ {
		go func(i int) {
			key := strconv.Itoa(i)
			cache.Set([]storage.Indexed[int]{storage.NewIndexed(key, i)})
			err := <-cache.Sync()
			done <- err
		}(i)
	}
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 5; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Sync returned error in concurrent test: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Sync did not return in time in concurrent test")
		}
	}
}
