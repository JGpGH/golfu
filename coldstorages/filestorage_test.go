package coldstorages_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JGpGH/golfu/coldstorages"
	"github.com/JGpGH/golfu/errors"
)

type testElement struct {
	ID    string
	Value string
	Count int
}

func (t testElement) Index() string {
	return t.ID
}

type testElementPointer struct {
	ID    string
	Value string
	Count int
}

func (t *testElementPointer) Index() string {
	return t.ID
}

func setupTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := filepath.Join(os.TempDir(), "golfu_test_"+time.Now().Format("20060102150405.000000"))
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

func TestNewFileStorage(t *testing.T) {
	tmpDir := setupTestDir(t)
	path := filepath.Join(tmpDir, "new_storage")

	fs := coldstorages.NewFileStorage[testElement](path)
	if fs == nil {
		t.Fatal("NewFileStorage returned nil")
	}

	// Verify directory was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected directory to be created at %s", path)
	}
}

func TestFileStorage_Set_Single(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	element := testElement{ID: "test1", Value: "value1", Count: 42}

	err := fs.Set(ctx, []testElement{element})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify element can be retrieved
	retrieved, err := fs.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil element")
	}
	if retrieved.ID != element.ID {
		t.Errorf("Expected ID %s, got %s", element.ID, retrieved.ID)
	}
	if retrieved.Value != element.Value {
		t.Errorf("Expected Value %s, got %s", element.Value, retrieved.Value)
	}
	if retrieved.Count != element.Count {
		t.Errorf("Expected Count %d, got %d", element.Count, retrieved.Count)
	}
}

func TestFileStorage_Set_Multiple(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	elements := []testElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}

	err := fs.Set(ctx, elements)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify all elements can be retrieved
	for _, element := range elements {
		retrieved, err := fs.Get(ctx, element.ID)
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
}

func TestFileStorage_Set_Overwrite(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	element1 := testElement{ID: "test1", Value: "value1", Count: 1}
	element2 := testElement{ID: "test1", Value: "value2", Count: 2}

	// Set first element
	err := fs.Set(ctx, []testElement{element1})
	if err != nil {
		t.Fatalf("First Set failed: %v", err)
	}

	// Overwrite with second element
	err = fs.Set(ctx, []testElement{element2})
	if err != nil {
		t.Fatalf("Second Set failed: %v", err)
	}

	// Verify second element is retrieved
	retrieved, err := fs.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Value != element2.Value {
		t.Errorf("Expected Value %s, got %s", element2.Value, retrieved.Value)
	}
	if retrieved.Count != element2.Count {
		t.Errorf("Expected Count %d, got %d", element2.Count, retrieved.Count)
	}
}

func TestFileStorage_Set_CancelledContext(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	element := testElement{ID: "test1", Value: "value1", Count: 1}
	err := fs.Set(ctx, []testElement{element})
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestFileStorage_Get_Existing(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	element := testElement{ID: "test1", Value: "value1", Count: 42}

	err := fs.Set(ctx, []testElement{element})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := fs.Get(ctx, "test1")
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

func TestFileStorage_Get_NotFound(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	retrieved, err := fs.Get(ctx, "nonexistent")
	if err != errors.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil element, got %+v", retrieved)
	}
}

func TestFileStorage_Get_CancelledContext(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	retrieved, err := fs.Get(ctx, "test1")
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil element, got %+v", retrieved)
	}
}

func TestFileStorage_Gets_Multiple(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	elements := []testElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
		{ID: "test3", Value: "value3", Count: 3},
	}

	err := fs.Set(ctx, elements)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	indexes := []string{"test1", "test2", "test3"}
	results, err := fs.Gets(ctx, indexes)
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

func TestFileStorage_Gets_PartialResults(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	elements := []testElement{
		{ID: "test1", Value: "value1", Count: 1},
		{ID: "test2", Value: "value2", Count: 2},
	}

	err := fs.Set(ctx, elements)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Request some existing and some non-existing
	indexes := []string{"test1", "test2", "nonexistent"}
	results, err := fs.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results (ignoring nonexistent), got %d", len(results))
	}

	if _, ok := results["test1"]; !ok {
		t.Error("Expected test1 in results")
	}
	if _, ok := results["test2"]; !ok {
		t.Error("Expected test2 in results")
	}
	if _, ok := results["nonexistent"]; ok {
		t.Error("Did not expect nonexistent in results")
	}
}

func TestFileStorage_Gets_Empty(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	indexes := []string{}
	results, err := fs.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestFileStorage_Gets_AllNonexistent(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	indexes := []string{"nonexistent1", "nonexistent2"}
	results, err := fs.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestFileStorage_Gets_CancelledContext(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	indexes := []string{"test1"}
	results, err := fs.Gets(ctx, indexes)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if results != nil {
		t.Errorf("Expected nil results, got %+v", results)
	}
}

func TestFileStorage_SpecialCharactersInIndex(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	specialIDs := []string{
		"test/with/slashes",
		"test:with:colons",
		"test with spaces",
		"test@with@at",
		"test#with#hash",
		"test?with?question",
		"test&with&ampersand",
	}

	for _, id := range specialIDs {
		element := testElement{ID: id, Value: "value_" + id, Count: 1}
		err := fs.Set(ctx, []testElement{element})
		if err != nil {
			t.Errorf("Set failed for ID %s: %v", id, err)
			continue
		}

		retrieved, err := fs.Get(ctx, id)
		if err != nil {
			t.Errorf("Get failed for ID %s: %v", id, err)
			continue
		}
		if retrieved == nil {
			t.Errorf("Get returned nil for ID %s", id)
			continue
		}
		if retrieved.ID != id {
			t.Errorf("ID mismatch for %s: got %s", id, retrieved.ID)
		}
	}
}

func TestFileStorage_PointerTypes(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[*testElementPointer](tmpDir)

	ctx := t.Context()
	element := &testElementPointer{ID: "test1", Value: "value1", Count: 42}

	err := fs.Set(ctx, []*testElementPointer{element})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := fs.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil")
	}
	if (*retrieved).ID != element.ID {
		t.Errorf("Expected ID %s, got %s", element.ID, (*retrieved).ID)
	}
	if (*retrieved).Value != element.Value {
		t.Errorf("Expected Value %s, got %s", element.Value, (*retrieved).Value)
	}
	if (*retrieved).Count != element.Count {
		t.Errorf("Expected Count %d, got %d", element.Count, (*retrieved).Count)
	}
}

func TestFileStorage_LargeDataset(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	count := 100
	elements := make([]testElement, count)
	for i := 0; i < count; i++ {
		elements[i] = testElement{
			ID:    "test" + string(rune(i)),
			Value: "value" + string(rune(i)),
			Count: i,
		}
	}

	err := fs.Set(ctx, elements)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Retrieve all elements
	indexes := make([]string, count)
	for i := 0; i < count; i++ {
		indexes[i] = elements[i].ID
	}

	results, err := fs.Gets(ctx, indexes)
	if err != nil {
		t.Fatalf("Gets failed: %v", err)
	}

	if len(results) != count {
		t.Errorf("Expected %d results, got %d", count, len(results))
	}
}

func TestFileStorage_EmptyIndex(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	element := testElement{ID: "", Value: "empty_id", Count: 1}

	err := fs.Set(ctx, []testElement{element})
	if err == nil {
		t.Error("Expected error for empty index, got nil")
	}
}

func TestFileStorage_Set_EmptySlice(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	err := fs.Set(ctx, []testElement{})
	if err != nil {
		t.Errorf("Set failed with empty slice: %v", err)
	}
}

func TestFileStorage_ConcurrentAccess(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	element := testElement{ID: "concurrent", Value: "test", Count: 1}

	// Set initial value
	err := fs.Set(ctx, []testElement{element})
	if err != nil {
		t.Fatalf("Initial Set failed: %v", err)
	}

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := fs.Get(ctx, "concurrent")
			if err != nil {
				t.Errorf("Concurrent Get failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFileStorage_Persistence(t *testing.T) {
	tmpDir := setupTestDir(t)
	fs1 := coldstorages.NewFileStorage[testElement](tmpDir)

	ctx := t.Context()
	element := testElement{ID: "persistent", Value: "data", Count: 99}

	err := fs1.Set(ctx, []testElement{element})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Create a new FileStorage instance pointing to the same directory
	fs2 := coldstorages.NewFileStorage[testElement](tmpDir)

	retrieved, err := fs2.Get(ctx, "persistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil")
	}
	if retrieved.ID != element.ID || retrieved.Value != element.Value || retrieved.Count != element.Count {
		t.Errorf("Data mismatch: got %+v, want %+v", retrieved, element)
	}
}
