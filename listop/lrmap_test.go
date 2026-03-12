package listop

import (
	"fmt"
	"sync"
	"testing"

	"github.com/JGpGH/golfu/element"
)

type testItem = element.Indexed[int]

func item(key string, val int) testItem {
	return element.NewIndexed(key, val)
}

func TestLRMapSetAndGet(t *testing.T) {
	lr := NewLRMap[testItem]()

	lr.Set([]testItem{item("a", 1), item("b", 2)})

	v, ok := lr.Get("a")
	if !ok || v.Value != 1 {
		t.Fatalf("expected a=1, got %v, %v", v, ok)
	}
	v, ok = lr.Get("b")
	if !ok || v.Value != 2 {
		t.Fatalf("expected b=2, got %v, %v", v, ok)
	}
	_, ok = lr.Get("c")
	if ok {
		t.Fatal("expected c to be missing")
	}
}

func TestLRMapOverwrite(t *testing.T) {
	lr := NewLRMap[testItem]()

	lr.Set([]testItem{item("a", 1)})
	lr.Set([]testItem{item("a", 42)})

	v, ok := lr.Get("a")
	if !ok || v.Value != 42 {
		t.Fatalf("expected a=42, got %v", v)
	}
}

func TestLRMapRemove(t *testing.T) {
	lr := NewLRMap[testItem]()

	lr.Set([]testItem{item("a", 1), item("b", 2), item("c", 3)})
	lr.Remove([]string{"a", "c"})

	if _, ok := lr.Get("a"); ok {
		t.Fatal("expected a to be removed")
	}
	if _, ok := lr.Get("c"); ok {
		t.Fatal("expected c to be removed")
	}
	v, ok := lr.Get("b")
	if !ok || v.Value != 2 {
		t.Fatal("expected b=2 to remain")
	}
}

func TestLRMapGets(t *testing.T) {
	lr := NewLRMap[testItem]()

	lr.Set([]testItem{item("a", 1), item("b", 2), item("c", 3)})

	result := lr.Gets([]string{"a", "c", "missing"})
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result["a"].Value != 1 || result["c"].Value != 3 {
		t.Fatalf("unexpected values: %v", result)
	}
}

func TestLRMapLen(t *testing.T) {
	lr := NewLRMap[testItem]()

	if lr.Len() != 0 {
		t.Fatal("expected empty map")
	}

	lr.Set([]testItem{item("a", 1), item("b", 2)})
	if lr.Len() != 2 {
		t.Fatalf("expected len=2, got %d", lr.Len())
	}

	lr.Remove([]string{"a"})
	if lr.Len() != 1 {
		t.Fatalf("expected len=1, got %d", lr.Len())
	}
}

func TestLRMapConcurrentReadsAndWrites(t *testing.T) {
	lr := NewLRMap[testItem]()
	const writers = 4
	const readers = 8
	const ops = 1000

	var wg sync.WaitGroup

	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ops {
				key := fmt.Sprintf("w%d-k%d", w, i)
				lr.Set([]testItem{item(key, i)})
			}
		}()
	}

	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ops {
				key := fmt.Sprintf("w0-k%d", i)
				lr.Get(key)
			}
		}()
	}

	wg.Wait()

	// Verify all writes landed
	for w := range writers {
		for i := range ops {
			key := fmt.Sprintf("w%d-k%d", w, i)
			v, ok := lr.Get(key)
			if !ok || v.Value != i {
				t.Fatalf("missing or wrong value for %s: got %v, %v", key, v, ok)
			}
		}
	}
}

func TestLRMapConcurrentRemoves(t *testing.T) {
	lr := NewLRMap[testItem]()
	const n = 500

	items := make([]testItem, n)
	for i := range n {
		items[i] = item(fmt.Sprintf("k%d", i), i)
	}
	lr.Set(items)

	var wg sync.WaitGroup

	// Writers removing odd keys
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i < n; i += 2 {
			lr.Remove([]string{fmt.Sprintf("k%d", i)})
		}
	}()

	// Readers concurrently
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				lr.Get("k0")
				lr.Gets([]string{"k0", "k2", "k4"})
				lr.Len()
			}
		}()
	}

	wg.Wait()

	// Even keys remain
	for i := 0; i < n; i += 2 {
		if _, ok := lr.Get(fmt.Sprintf("k%d", i)); !ok {
			t.Fatalf("expected even key k%d to remain", i)
		}
	}
	// Odd keys removed
	for i := 1; i < n; i += 2 {
		if _, ok := lr.Get(fmt.Sprintf("k%d", i)); ok {
			t.Fatalf("expected odd key k%d to be removed", i)
		}
	}
}

func TestLRMapBothSidesConsistent(t *testing.T) {
	lr := NewLRMap[testItem]()

	// Alternating sets and removes to exercise both sides
	for i := range 100 {
		lr.Set([]testItem{item(fmt.Sprintf("k%d", i), i)})
	}
	for i := range 50 {
		lr.Remove([]string{fmt.Sprintf("k%d", i)})
	}

	// Every read should see the same state regardless of which side it hits
	for i := range 100 {
		key := fmt.Sprintf("k%d", i)
		v, ok := lr.Get(key)
		if i < 50 {
			if ok {
				t.Fatalf("expected %s to be removed", key)
			}
		} else {
			if !ok || v.Value != i {
				t.Fatalf("expected %s=%d, got %v, %v", key, i, v, ok)
			}
		}
	}
}
