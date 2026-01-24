package internal_test

import (
	"context"
	"testing"
	"time"

	"github.com/JGpGH/golfu"
	"github.com/JGpGH/golfu/coldstorages"
	"github.com/JGpGH/golfu/element"
)

func BenchmarkCachedStorageSet2(b *testing.B) {
	tempDir := b.TempDir()
	cold := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := golfu.NewCachedStorage(ctx, cold, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, []element.Indexed[int]{
			element.NewIndexed("bench1", i),
			element.NewIndexed("bench2", i+1),
		})
	}
	b.StopTimer()
	cancel()
	time.Sleep(100 * time.Millisecond) // Wait for async writes to complete
}

func BenchmarkCachedStorageSet1(b *testing.B) {
	tempDir := b.TempDir()
	cold := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := golfu.NewCachedStorage(ctx, cold, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, []element.Indexed[int]{
			element.NewIndexed("bench1", i),
		})
	}
	b.StopTimer()
	cancel()
	time.Sleep(100 * time.Millisecond) // Wait for async writes to complete
}

func BenchmarkCachedStorageGetCacheHit1(b *testing.B) {
	tempDir := b.TempDir()
	cold := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	cache := golfu.NewCachedStorage(context.Background(), cold, 100)
	ctx := context.Background()

	// Pre-populate cache
	cache.Set(ctx, []element.Indexed[int]{
		element.NewIndexed("bench1", 1),
		element.NewIndexed("bench2", 2),
		element.NewIndexed("bench3", 3),
	})
	time.Sleep(100 * time.Millisecond) // Wait for async write to cold storage

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Get(ctx, "bench1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedStorageGets10Hits(b *testing.B) {
	tempDir := b.TempDir()
	cold := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	cache := golfu.NewCachedStorage(context.Background(), cold, 101)
	ctx := context.Background()

	// Pre-populate
	items := make([]element.Indexed[int], 100)
	indexes := make([]string, 100)
	for i := 0; i < 100; i++ {
		idx := "item" + string(rune(i))
		items[i] = element.NewIndexed(idx, i)
		indexes[i] = idx
	}
	cache.Set(ctx, items)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Gets(ctx, indexes[:10])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedStorageGetsMixedHitMiss10(b *testing.B) {
	tempDir := b.TempDir()
	cold := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	cache := golfu.NewCachedStorage(context.Background(), cold, 50)
	ctx := context.Background()

	// Pre-populate cold storage with more items than cache can hold
	items := make([]element.Indexed[int], 100)
	for i := 0; i < 100; i++ {
		items[i] = element.NewIndexed("item"+string(rune(i)), i)
	}
	cold.Set(ctx, items)

	// Warm up cache with first 50
	cache.Set(ctx, items[:50])
	time.Sleep(100 * time.Millisecond)

	// Query mix of cached (0-49) and cold (50-59)
	queryIndexes := make([]string, 10)
	for i := 0; i < 5; i++ {
		queryIndexes[i] = "item" + string(rune(i)) // Cache hits
	}
	for i := 5; i < 10; i++ {
		queryIndexes[i] = "item" + string(rune(i+45)) // Cache misses
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Gets(ctx, queryIndexes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorageSet2(b *testing.B) {
	tempDir := b.TempDir()
	fs := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	ctx := context.Background()

	items := []element.Indexed[int]{
		element.NewIndexed("bench1", 1),
		element.NewIndexed("bench2", 2),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := fs.Set(ctx, items)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorageSet1(b *testing.B) {
	tempDir := b.TempDir()
	fs := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	ctx := context.Background()

	items := []element.Indexed[int]{
		element.NewIndexed("bench1", 1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := fs.Set(ctx, items)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorageGet1(b *testing.B) {
	tempDir := b.TempDir()
	fs := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	ctx := context.Background()

	// Pre-populate
	fs.Set(ctx, []element.Indexed[int]{
		element.NewIndexed("bench1", 1),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fs.Get(ctx, "bench1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorageGets10(b *testing.B) {
	tempDir := b.TempDir()
	fs := coldstorages.NewFileStorage[element.Indexed[int]](tempDir)
	ctx := context.Background()

	// Pre-populate
	items := make([]element.Indexed[int], 100)
	indexes := make([]string, 100)
	for i := 0; i < 100; i++ {
		idx := "item" + string(rune(i))
		items[i] = element.NewIndexed(idx, i)
		indexes[i] = idx
	}
	fs.Set(ctx, items)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fs.Gets(ctx, indexes[:10])
		if err != nil {
			b.Fatal(err)
		}
	}
}
