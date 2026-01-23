package internal_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/JGpGH/golfu/coldstorages"
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

// Additional tests using InMemoryStorage for better coverage
func TestCachedStorage_Get_FromCache(t *testing.T) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)

	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 42),
	})
	time.Sleep(50 * time.Millisecond)

	result, err := cache.Get(context.Background(), "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	if result.Value != 42 {
		t.Errorf("Expected value 42, got %d", result.Value)
	}
}

func TestCachedStorage_Get_FromCold(t *testing.T) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)

	cold.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 99),
	})

	result, err := cache.Get(context.Background(), "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	if result.Value != 99 {
		t.Errorf("Expected value 99, got %d", result.Value)
	}
}

func TestCachedStorage_Gets_Mixed(t *testing.T) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10)

	cache.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("1", 1),
		element.NewIndexed("2", 2),
	})
	time.Sleep(50 * time.Millisecond)

	cold.Set(context.Background(), []element.Indexed[int]{
		element.NewIndexed("3", 3),
		element.NewIndexed("4", 4),
	})

	results, err := cache.Gets(context.Background(), []string{"1", "2", "3", "4", "5"})
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}
}

func TestCachedStorage_ConcurrentSets(t *testing.T) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 100)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := string(rune('A'+id)) + string(rune('0'+j))
				cache.Set(context.Background(), []element.Indexed[int]{
					element.NewIndexed(key, id*10+j),
				})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}

// Benchmarks
func BenchmarkCachedStorage_Set(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, []element.Indexed[int]{
			element.NewIndexed(string(rune(i)), i),
		})
	}
}

func BenchmarkCachedStorage_Get_CacheHit(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10000)
	ctx := context.Background()

	elements := make([]element.Indexed[int], 1000)
	for i := 0; i < 1000; i++ {
		elements[i] = element.NewIndexed(string(rune(i)), i)
	}
	cache.Set(ctx, elements)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, string(rune(i%1000)))
	}
}

func BenchmarkCachedStorage_Gets(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := internal.NewCachedStorage(context.Background(), cold, 10000)
	ctx := context.Background()

	elements := make([]element.Indexed[int], 1000)
	for i := 0; i < 1000; i++ {
		elements[i] = element.NewIndexed(string(rune(i)), i)
	}
	cache.Set(ctx, elements)
	time.Sleep(100 * time.Millisecond)

	indexes := make([]string, 10)
	for i := 0; i < 10; i++ {
		indexes[i] = string(rune(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Gets(ctx, indexes)
	}
}
