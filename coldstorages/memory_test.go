package coldstorages_test

import (
	"context"
	"sync"
	"testing"

	"github.com/JGpGH/golfu/coldstorages"
	"github.com/JGpGH/golfu/errors"
)

type memTestElement struct {
	ID    string
	Value string
	Count int
}

func (t memTestElement) Index() string {
	return t.ID
}

type memTestElementPointer struct {
	ID    string
	Value string
	Count int
}

func (t *memTestElementPointer) Index() string {
	return t.ID
}

func TestNewInMemoryStorage(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	if storage == nil {
		t.Fatal("NewInMemoryStorage returned nil")
	}
	if storage.Size() != 0 {
		t.Errorf("Expected size 0, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Set_Single(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element := memTestElement{ID: "test1", Value: "value1", Count: 42}
	err := storage.Set(ctx, []memTestElement{element})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := storage.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil")
	}
	if retrieved.ID != element.ID || retrieved.Value != element.Value || retrieved.Count != element.Count {
		t.Errorf("Element mismatch: got %+v, want %+v", retrieved, element)
	}
}

func TestInMemoryStorage_Set_Multiple(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}

	err := storage.Set(ctx, elements)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	for _, element := range elements {
		retrieved, err := storage.Get(ctx, element.ID)
		if err != nil {
			t.Errorf("Get failed for %s: %v", element.ID, err)
		}
		if retrieved == nil {
			t.Errorf("Get returned nil for %s", element.ID)
			continue
		}
		if retrieved.ID != element.ID || retrieved.Value != element.Value || retrieved.Count != element.Count {
			t.Errorf("Element mismatch for %s: got %+v, want %+v", element.ID, retrieved, element)
		}
	}

	if storage.Size() != 3 {
		t.Errorf("Expected size 3, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Set_Overwrite(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element1 := memTestElement{ID: "test1", Value: "value1", Count: 1}
	element2 := memTestElement{ID: "test1", Value: "value2", Count: 2}

	err := storage.Set(ctx, []memTestElement{element1})
	if err != nil {
		t.Fatalf("First Set failed: %v", err)
	}

	err = storage.Set(ctx, []memTestElement{element2})
	if err != nil {
		t.Fatalf("Second Set failed: %v", err)
	}

	retrieved, err := storage.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Value != element2.Value || retrieved.Count != element2.Count {
		t.Errorf("Expected element2 values, got %+v", retrieved)
	}

	if storage.Size() != 1 {
		t.Errorf("Expected size 1 after overwrite, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Set_EmptyIndex(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element := memTestElement{ID: "", Value: "empty", Count: 1}
	err := storage.Set(ctx, []memTestElement{element})
	if err != errors.ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex, got %v", err)
	}

	if storage.Size() != 0 {
		t.Errorf("Expected size 0 after failed Set, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Set_CancelledContext(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	element := memTestElement{ID: "test1", Value: "value1", Count: 1}
	err := storage.Set(ctx, []memTestElement{element})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestInMemoryStorage_Set_EmptySlice(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	err := storage.Set(ctx, []memTestElement{})
	if err != nil {
		t.Errorf("Set failed with empty slice: %v", err)
	}
}

func TestInMemoryStorage_Get_Existing(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element := memTestElement{ID: "test1", Value: "value1", Count: 42}
	storage.Set(ctx, []memTestElement{element})

	retrieved, err := storage.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil")
	}
	if retrieved.ID != element.ID {
		t.Errorf("Expected ID %s, got %s", element.ID, retrieved.ID)
	}
}

func TestInMemoryStorage_Get_NotFound(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	retrieved, err := storage.Get(ctx, "nonexistent")
	if err != errors.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil element, got %+v", retrieved)
	}
}

func TestInMemoryStorage_Get_EmptyIndex(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	retrieved, err := storage.Get(ctx, "")
	if err != errors.ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex, got %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil element, got %+v", retrieved)
	}
}

func TestInMemoryStorage_Get_CancelledContext(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	retrieved, err := storage.Get(ctx, "test1")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil element, got %+v", retrieved)
	}
}

func TestInMemoryStorage_Gets_Multiple(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}
	storage.Set(ctx, elements)

	indexes := []string{"test1", "test2", "test3"}
	results, err := storage.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, element := range elements {
		result, ok := results[element.ID]
		if !ok {
			t.Errorf("Expected element %s in results", element.ID)
			continue
		}
		if result.ID != element.ID || result.Value != element.Value || result.Count != element.Count {
			t.Errorf("Element mismatch for %s: got %+v, want %+v", element.ID, result, element)
		}
	}
}

func TestInMemoryStorage_Gets_PartialResults(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
	}
	storage.Set(ctx, elements)

	indexes := []string{"test1", "test2", "nonexistent"}
	results, err := storage.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if _, ok := results["nonexistent"]; ok {
		t.Error("Did not expect nonexistent in results")
	}
}

func TestInMemoryStorage_Gets_Empty(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	results, err := storage.Gets(ctx, []string{})
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestInMemoryStorage_Gets_AllNonexistent(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	results, err := storage.Gets(ctx, []string{"nonexistent1", "nonexistent2"})
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestInMemoryStorage_Gets_CancelledContext(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, err := storage.Gets(ctx, []string{"test1"})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if results != nil {
		t.Errorf("Expected nil results, got %+v", results)
	}
}

func TestInMemoryStorage_Gets_WithEmptyIndex(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
	}
	storage.Set(ctx, elements)

	// Empty index should be skipped
	indexes := []string{"test1", "", "test2"}
	results, err := storage.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results (empty index skipped), got %d", len(results))
	}
}

func TestInMemoryStorage_Delete_Single(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element := memTestElement{ID: "test1", Value: "value1", Count: 1}
	storage.Set(ctx, []memTestElement{element})

	err := storage.Delete(ctx, []string{"test1"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, err := storage.Get(ctx, "test1")
	if err != errors.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil after delete, got %+v", retrieved)
	}

	if storage.Size() != 0 {
		t.Errorf("Expected size 0 after delete, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Delete_Multiple(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}
	storage.Set(ctx, elements)

	err := storage.Delete(ctx, []string{"test1", "test3"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// test1 should be deleted
	if _, err := storage.Get(ctx, "test1"); err != errors.ErrNotFound {
		t.Errorf("Expected ErrNotFound for test1, got %v", err)
	}

	// test2 should still exist
	if _, err := storage.Get(ctx, "test2"); err != nil {
		t.Errorf("Expected test2 to still exist, got error %v", err)
	}

	// test3 should be deleted
	if _, err := storage.Get(ctx, "test3"); err != errors.ErrNotFound {
		t.Errorf("Expected ErrNotFound for test3, got %v", err)
	}

	if storage.Size() != 1 {
		t.Errorf("Expected size 1 after deleting 2 of 3, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Delete_Nonexistent(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	// Deleting nonexistent element should not error
	err := storage.Delete(ctx, []string{"nonexistent"})
	if err != nil {
		t.Errorf("Delete failed for nonexistent element: %v", err)
	}
}

func TestInMemoryStorage_Delete_CancelledContext(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := storage.Delete(ctx, []string{"test1"})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestInMemoryStorage_Clear(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}
	storage.Set(ctx, elements)

	if storage.Size() != 3 {
		t.Errorf("Expected size 3 before clear, got %d", storage.Size())
	}

	storage.Clear()

	if storage.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", storage.Size())
	}

	// Verify all elements are gone
	for _, element := range elements {
		if _, err := storage.Get(ctx, element.ID); err != errors.ErrNotFound {
			t.Errorf("Expected ErrNotFound for %s after clear, got %v", element.ID, err)
		}
	}
}

func TestInMemoryStorage_Size(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	if storage.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", storage.Size())
	}

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
	}
	storage.Set(ctx, elements)

	if storage.Size() != 2 {
		t.Errorf("Expected size 2 after Set, got %d", storage.Size())
	}

	storage.Delete(ctx, []string{"test1"})

	if storage.Size() != 1 {
		t.Errorf("Expected size 1 after Delete, got %d", storage.Size())
	}
}

func TestInMemoryStorage_Has(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element := memTestElement{ID: "test1", Value: "value1", Count: 1}
	storage.Set(ctx, []memTestElement{element})

	if !storage.Has("test1") {
		t.Error("Expected Has to return true for test1")
	}

	if storage.Has("nonexistent") {
		t.Error("Expected Has to return false for nonexistent")
	}

	storage.Delete(ctx, []string{"test1"})

	if storage.Has("test1") {
		t.Error("Expected Has to return false after delete")
	}
}

func TestInMemoryStorage_Keys(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	elements := []memTestElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}
	storage.Set(ctx, elements)

	keys := storage.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Convert to map for easy checking
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for _, element := range elements {
		if !keyMap[element.ID] {
			t.Errorf("Expected key %s in Keys(), not found", element.ID)
		}
	}
}

func TestInMemoryStorage_PointerTypes(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[*memTestElementPointer]()
	ctx := context.Background()

	element := &memTestElementPointer{ID: "test1", Value: "value1", Count: 42}
	err := storage.Set(ctx, []*memTestElementPointer{element})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := storage.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil")
	}
	if (*retrieved).ID != element.ID || (*retrieved).Value != element.Value || (*retrieved).Count != element.Count {
		t.Errorf("Element mismatch: got %+v, want %+v", *retrieved, element)
	}
}

func TestInMemoryStorage_LargeDataset(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	count := 1000
	elements := make([]memTestElement, count)
	for i := 0; i < count; i++ {
		elements[i] = memTestElement{
			ID:    string(rune('A'+(i%26))) + string(rune(i/26)),
			Value: "value" + string(rune(i)),
			Count: i,
		}
	}

	err := storage.Set(ctx, elements)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if storage.Size() != count {
		t.Errorf("Expected size %d, got %d", count, storage.Size())
	}

	// Retrieve all elements
	indexes := make([]string, count)
	for i := 0; i < count; i++ {
		indexes[i] = elements[i].ID
	}

	results, err := storage.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != count {
		t.Errorf("Expected %d results, got %d", count, len(results))
	}
}

func TestInMemoryStorage_ConcurrentReads(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	element := memTestElement{ID: "concurrent", Value: "test", Count: 1}
	storage.Set(ctx, []memTestElement{element})

	var wg sync.WaitGroup
	numReaders := 100

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retrieved, err := storage.Get(ctx, "concurrent")
			if err != nil {
				t.Errorf("Concurrent Get failed: %v", err)
			}
			if retrieved == nil {
				t.Error("Concurrent Get returned nil")
			}
		}()
	}

	wg.Wait()
}

func TestInMemoryStorage_ConcurrentWrites(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	var wg sync.WaitGroup
	numWriters := 100

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			element := memTestElement{
				ID:    "test" + string(rune('A'+id%26)) + string(rune(id/26)),
				Value: "value",
				Count: id,
			}
			err := storage.Set(ctx, []memTestElement{element})
			if err != nil {
				t.Errorf("Concurrent Set failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if storage.Size() != numWriters {
		t.Errorf("Expected size %d after concurrent writes, got %d", numWriters, storage.Size())
	}
}

func TestInMemoryStorage_ConcurrentReadWrite(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	// Initial data
	for i := 0; i < 10; i++ {
		element := memTestElement{
			ID:    "test" + string(rune('0'+i)),
			Value: "value",
			Count: i,
		}
		storage.Set(ctx, []memTestElement{element})
	}

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "test" + string(rune('0'+(id%10)))
			storage.Get(ctx, key)
		}(i)
	}

	// Concurrent writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			element := memTestElement{
				ID:    "test" + string(rune('0'+(id%10))),
				Value: "updated",
				Count: id,
			}
			storage.Set(ctx, []memTestElement{element})
		}(i)
	}

	wg.Wait()

	// Verify storage is still consistent
	if storage.Size() != 10 {
		t.Errorf("Expected size 10 after concurrent read/write, got %d", storage.Size())
	}
}

func TestInMemoryStorage_SpecialCharactersInIndex(t *testing.T) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	specialIDs := []string{
		"test/with/slashes",
		"test:with:colons",
		"test with spaces",
		"test@with@at",
		"test#with#hash",
		"test?with?question",
		"test&with&ampersand",
		"test\nwith\nnewlines",
		"test\twith\ttabs",
	}

	for _, id := range specialIDs {
		element := memTestElement{ID: id, Value: "value_" + id, Count: 1}
		err := storage.Set(ctx, []memTestElement{element})
		if err != nil {
			t.Errorf("Set failed for ID %q: %v", id, err)
			continue
		}

		retrieved, err := storage.Get(ctx, id)
		if err != nil {
			t.Errorf("Get failed for ID %q: %v", id, err)
			continue
		}
		if retrieved == nil {
			t.Errorf("Get returned nil for ID %q", id)
			continue
		}
		if retrieved.ID != id {
			t.Errorf("ID mismatch for %q: got %q", id, retrieved.ID)
		}
	}
}

func BenchmarkInMemoryStorage_Set(b *testing.B) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		element := memTestElement{
			ID:    "bench" + string(rune(i)),
			Value: "value",
			Count: i,
		}
		storage.Set(ctx, []memTestElement{element})
	}
}

func BenchmarkInMemoryStorage_Get(b *testing.B) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	// Setup: add elements
	for i := 0; i < 1000; i++ {
		element := memTestElement{
			ID:    "bench" + string(rune(i)),
			Value: "value",
			Count: i,
		}
		storage.Set(ctx, []memTestElement{element})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.Get(ctx, "bench"+string(rune(i%1000)))
	}
}

func BenchmarkInMemoryStorage_Gets(b *testing.B) {
	storage := coldstorages.NewInMemoryStorage[memTestElement]()
	ctx := context.Background()

	// Setup: add elements
	for i := 0; i < 1000; i++ {
		element := memTestElement{
			ID:    "bench" + string(rune(i)),
			Value: "value",
			Count: i,
		}
		storage.Set(ctx, []memTestElement{element})
	}

	indexes := make([]string, 10)
	for i := 0; i < 10; i++ {
		indexes[i] = "bench" + string(rune(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.Gets(ctx, indexes)
	}
}
