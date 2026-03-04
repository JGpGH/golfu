package lockmap_test

import (
	"sync"
	"testing"
	"time"

	"github.com/JGpGH/golfu/lockmap"
)

func TestNew(t *testing.T) {
	lm := lockmap.New()
	if lm == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLockMap_Lock_BasicUsage(t *testing.T) {
	lm := lockmap.New()

	// Acquire lock
	unlock := lm.Lock("test1")
	if unlock == nil {
		t.Fatal("Lock returned nil unlock function")
	}

	// Release lock
	unlock()
}

func TestLockMap_Rlock_BasicUsage(t *testing.T) {
	lm := lockmap.New()

	// Acquire read lock
	unlock := lm.Rlock("test1")
	if unlock == nil {
		t.Fatal("Rlock returned nil unlock function")
	}

	// Release lock
	unlock()
}

func TestLockMap_Lock_ExclusiveAccess(t *testing.T) {
	lm := lockmap.New()

	var counter int
	var wg sync.WaitGroup

	// Lock the resource
	unlock := lm.Lock("exclusive")

	// Try to acquire lock in another goroutine
	wg.Add(1)
	acquired := make(chan bool, 1)
	go func() {
		defer wg.Done()
		unlock2 := lm.Lock("exclusive")
		acquired <- true
		counter++
		unlock2()
	}()

	// Verify lock is not acquired while first lock is held
	select {
	case <-acquired:
		t.Fatal("Lock was acquired while another lock was held")
	case <-time.After(10 * time.Millisecond):
		// Expected - lock not acquired
	}

	// Release first lock
	unlock()

	// Wait for second goroutine to acquire and complete
	select {
	case <-acquired:
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Second lock was not acquired after first was released")
	}

	wg.Wait()

	if counter != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter)
	}
}

func TestLockMap_Rlock_ConcurrentReads(t *testing.T) {
	lm := lockmap.New()

	const numReaders = 100
	var wg sync.WaitGroup
	started := make(chan bool, numReaders)
	proceed := make(chan bool)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started <- true
			<-proceed
			unlock := lm.Rlock("concurrent_read")
			unlock()
		}()
	}

	// Wait for all goroutines to start
	for i := 0; i < numReaders; i++ {
		<-started
	}

	// Signal all to proceed at once
	close(proceed)

	// Wait for all to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		// Success - all readers completed concurrently
	case <-time.After(2 * time.Second):
		t.Fatal("Concurrent reads timed out")
	}
}

func TestLockMap_Lock_BlocksRlock(t *testing.T) {
	lm := lockmap.New()

	// Acquire write lock
	unlock := lm.Lock("resource")

	acquired := make(chan bool, 1)
	go func() {
		unlock2 := lm.Rlock("resource")
		acquired <- true
		unlock2()
	}()

	// Verify Rlock is blocked
	select {
	case <-acquired:
		t.Fatal("Rlock was acquired while Lock was held")
	case <-time.After(100 * time.Millisecond):
		// Expected - Rlock is blocked
	}

	// Release write lock
	unlock()

	// Read lock should now be acquired
	select {
	case <-acquired:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Rlock was not acquired after Lock was released")
	}
}

func TestLockMap_Rlock_BlocksLock(t *testing.T) {
	lm := lockmap.New()

	// Acquire read lock
	unlock := lm.Rlock("resource")

	acquired := make(chan bool, 1)
	go func() {
		unlock2 := lm.Lock("resource")
		acquired <- true
		unlock2()
	}()

	// Verify Lock is blocked
	select {
	case <-acquired:
		t.Fatal("Lock was acquired while Rlock was held")
	case <-time.After(100 * time.Millisecond):
		// Expected - Lock is blocked
	}

	// Release read lock
	unlock()

	// Write lock should now be acquired
	select {
	case <-acquired:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Lock was not acquired after Rlock was released")
	}
}

func TestLockMap_MultipleIndexes(t *testing.T) {
	lm := lockmap.New()

	const numIndexes = 10
	const opsPerIndex = 20
	var wg sync.WaitGroup

	for i := 0; i < numIndexes; i++ {
		for j := 0; j < opsPerIndex; j++ {
			wg.Add(1)
			go func(index string) {
				defer wg.Done()
				unlock := lm.Rlock(index)
				unlock()
			}(string(rune('A' + i)))
		}
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Operations on multiple indexes timed out")
	}
}

func TestLockMap_Rlock_RefCounting(t *testing.T) {
	lm := lockmap.New()

	const numOverlappingReads = 50
	var wg sync.WaitGroup
	started := make(chan bool, numOverlappingReads)
	proceed := make(chan bool)

	for i := 0; i < numOverlappingReads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started <- true
			<-proceed
			unlock := lm.Rlock("refcount")
			unlock()
		}()
	}

	// Wait for all to start
	for i := 0; i < numOverlappingReads; i++ {
		<-started
	}

	// Signal all to proceed
	close(proceed)

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Overlapping reads timed out")
	}

	// Verify we can acquire lock after all reads complete
	unlock := lm.Lock("refcount")
	unlock()
}

func TestLockMap_WriteDuringReads(t *testing.T) {
	lm := lockmap.New()

	const numReads = 10
	var wg sync.WaitGroup
	readStarted := make(chan bool, numReads)
	readsHoldingLock := make(chan bool)
	writeAcquired := make(chan bool, 1)

	// Start multiple reads
	for i := 0; i < numReads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := lm.Rlock("mixed")
			readStarted <- true
			<-readsHoldingLock // Wait for signal to release
			unlock()
		}()
	}

	// Wait for all reads to acquire locks
	for i := 0; i < numReads; i++ {
		<-readStarted
	}

	// Start a write - should wait for reads to complete
	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock := lm.Lock("mixed")
		writeAcquired <- true
		unlock()
	}()

	// Write should not have been acquired yet
	select {
	case <-writeAcquired:
		t.Fatal("Write lock acquired while reads were in progress")
	case <-time.After(100 * time.Millisecond):
		// Expected - write is blocked
	}

	// Release all reads
	close(readsHoldingLock)

	// Wait for all operations to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Mixed read/write operations timed out")
	}
}

func TestLockMap_ConcurrentWrites(t *testing.T) {
	lm := lockmap.New()

	const numWrites = 50
	var wg sync.WaitGroup
	var counter int
	var mu sync.Mutex

	for i := 0; i < numWrites; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := lm.Lock("write_resource")
			// Critical section - increment counter
			mu.Lock()
			counter++
			mu.Unlock()
			unlock()
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Concurrent writes timed out")
	}

	if counter != numWrites {
		t.Errorf("Expected counter to be %d, got %d", numWrites, counter)
	}
}

func TestLockMap_Lock_Cleanup(t *testing.T) {
	lm := lockmap.New()

	// Acquire and release lock multiple times
	for i := 0; i < 100; i++ {
		unlock := lm.Lock("cleanup_test")
		unlock()
	}

	if lm.Len() != 0 {
		t.Errorf("Expected map to be empty after all locks released, got len %d", lm.Len())
	}

	// Should be able to acquire lock again without issues
	unlock := lm.Lock("cleanup_test")
	unlock()
}

func TestLockMap_Rlock_Cleanup(t *testing.T) {
	lm := lockmap.New()

	// Acquire and release read lock multiple times
	for i := 0; i < 100; i++ {
		unlock := lm.Rlock("cleanup_test")
		unlock()
	}

	if lm.Len() != 0 {
		t.Errorf("Expected map to be empty after all rlocks released, got len %d", lm.Len())
	}

	// Should be able to acquire lock again without issues
	unlock := lm.Rlock("cleanup_test")
	unlock()
}

func TestLockMap_SpecialCharactersInIndex(t *testing.T) {
	lm := lockmap.New()

	specialIndexes := []string{
		"test/with/slashes",
		"test:with:colons",
		"test with spaces",
		"test@with@at",
		"test#with#hash",
		"test?with?question",
		"test&with&ampersand",
		"",
	}

	for _, index := range specialIndexes {
		unlock := lm.Lock(index)
		unlock()

		unlock = lm.Rlock(index)
		unlock()
	}
}

func TestLockMap_CleanupAfterConcurrentUse(t *testing.T) {
	lm := lockmap.New()

	const numKeys = 20
	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := string(rune('A' + id%numKeys))
			if id%2 == 0 {
				unlock := lm.Lock(key)
				unlock()
			} else {
				unlock := lm.Rlock(key)
				unlock()
			}
		}(i)
	}

	wg.Wait()

	if lm.Len() != 0 {
		t.Errorf("Expected map to be empty after all concurrent ops, got len %d", lm.Len())
	}
}

func TestLockMap_StressTest(t *testing.T) {
	lm := lockmap.New()

	const numGoroutines = 100
	const numOperations = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				index := string(rune('A' + (id+j)%10))
				if j%2 == 0 {
					unlock := lm.Lock(index)
					unlock()
				} else {
					unlock := lm.Rlock(index)
					unlock()
				}
			}
		}(i)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Stress test timed out")
	}

	if lm.Len() != 0 {
		t.Errorf("Expected map to be empty after stress test, got len %d", lm.Len())
	}
}
