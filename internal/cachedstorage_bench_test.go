package internal_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/JGpGH/golfu"
	"github.com/JGpGH/golfu/coldstorages"
	"github.com/JGpGH/golfu/element"
)

func BenchmarkCachedStorageSet2(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := golfu.NewCachedStorage(b.Context(), cold, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(b.Context(), []element.Indexed[int]{
			element.NewIndexed("bench1", i),
			element.NewIndexed("bench2", i+1),
		})
	}
}

func BenchmarkCachedStorageSet1(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	cache := golfu.NewCachedStorage(b.Context(), cold, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(b.Context(), []element.Indexed[int]{
			element.NewIndexed("bench1", i),
		})
	}
}

func BenchmarkCachedStorageGetCacheHit1(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	ctx := b.Context()
	cache := golfu.NewCachedStorage(ctx, cold, 100)

	cache.Set(ctx, []element.Indexed[int]{
		element.NewIndexed("bench1", 1),
		element.NewIndexed("bench2", 2),
		element.NewIndexed("bench3", 3),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Get(ctx, "bench1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedStorageGets10Hits(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	ctx := b.Context()
	cache := golfu.NewCachedStorage(ctx, cold, 101)

	items := make([]element.Indexed[int], 100)
	indexes := make([]string, 100)
	for i := range 100 {
		idx := "item" + string(rune(i))
		items[i] = element.NewIndexed(idx, i)
		indexes[i] = idx
	}
	cache.Set(ctx, items)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Gets(ctx, indexes[:10])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedStorageGetsMixedHitMiss10(b *testing.B) {
	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	ctx := b.Context()
	cache := golfu.NewCachedStorage(ctx, cold, 50)

	items := make([]element.Indexed[int], 100)
	for i := range 100 {
		items[i] = element.NewIndexed("item"+string(rune(i)), i)
	}
	// Populate cold storage for cache-miss lookups
	cold.Set(ctx, items)

	// Warm up cache with first 50
	cache.Set(ctx, items[:50])

	queryIndexes := make([]string, 10)
	for i := range 5 {
		queryIndexes[i] = "item" + string(rune(i))
	}
	for i := 5; i < 10; i++ {
		queryIndexes[i] = "item" + string(rune(i+45))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Gets(ctx, queryIndexes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Parallel benchmark: 80% reads, 20% writes with Zipf-distributed key access.
// Simulates a realistic hot-key workload where a small set of keys gets most traffic.
func BenchmarkCachedStorageParallelMixed(b *testing.B) {
	const numKeys = 10000
	const cacheSize = 2000

	cold := coldstorages.NewInMemoryStorage[element.Indexed[int]]()
	ctx := b.Context()
	cache := golfu.NewCachedStorage(ctx, cold, cacheSize)

	// Pre-populate cold + warm cache with a subset
	items := make([]element.Indexed[int], numKeys)
	for i := range numKeys {
		items[i] = element.NewIndexed(fmt.Sprintf("k%d", i), i)
	}
	cold.Set(ctx, items)
	cache.Set(ctx, items[:cacheSize])

	// Pre-build key lists for Gets (batches of 5)
	keys := make([]string, numKeys)
	for i := range numKeys {
		keys[i] = fmt.Sprintf("k%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		zipf := rand.NewZipf(rng, 1.1, 1.0, numKeys-1)
		for pb.Next() {
			if rng.IntN(100) < 80 {
				// 80% reads: batch of 5 keys, Zipf-distributed
				batch := make([]string, 5)
				for j := range batch {
					batch[j] = keys[zipf.Uint64()]
				}
				cache.Gets(ctx, batch)
			} else {
				// 20% writes: single item, Zipf-distributed key
				idx := zipf.Uint64()
				cache.Set(ctx, []element.Indexed[int]{
					element.NewIndexed(keys[idx], int(idx)),
				})
			}
		}
	})
}

