package listop

import (
	"container/list"
	"sync"
	"sync/atomic"

	"github.com/JGpGH/golfu/element"
)

// readCountTracker over an Element with value of type T
type readCountTracker[T element.Indexable] struct {
	Element        *list.Element
	ReadWriteCount *atomic.Uint32
	index          string
}

func (c readCountTracker[T]) Index() string {
	return c.index
}

func (c readCountTracker[T]) Value() T {
	return c.Element.Value.(T)
}

type IndexedList[T element.Indexable] struct {
	indexed    *LRMap[readCountTracker[T]]
	sorted     list.List
	sortedLock sync.RWMutex
}

func NewIndexedList[T element.Indexable]() IndexedList[T] {
	return IndexedList[T]{
		indexed: NewLRMap[readCountTracker[T]](),
		sorted:  list.List{},
	}
}

func (l *IndexedList[T]) readWriteCount(e *list.Element) uint32 {
	c, _ := l.indexed.Get(e.Value.(T).Index())
	return c.ReadWriteCount.Load()
}

func (l *IndexedList[T]) Remove(indexes []string) {
	l.sortedLock.Lock()
	defer l.sortedLock.Unlock()
	trackers := l.indexed.Gets(indexes)
	for _, c := range trackers {
		l.sorted.Remove(c.Element)
	}
	l.indexed.Remove(indexes)
}

func (l *IndexedList[T]) Gets(indexes []string) map[string]T {
	trackers := l.indexed.Gets(indexes)
	res := make(map[string]T, len(trackers))
	for k, c := range trackers {
		c.ReadWriteCount.Add(1)
		res[k] = c.Value()
	}
	return res
}

func (l *IndexedList[T]) Get(index string) *T {
	if c, ok := l.indexed.Get(index); ok {
		c.ReadWriteCount.Add(1)
		val := c.Value()
		return &val
	}
	return nil
}

// returns the read write count for the given indexes; does not affect the count
func (l *IndexedList[T]) ReadWriteCounts(indexes []string) map[string]uint32 {
	trackers := l.indexed.Gets(indexes)
	res := make(map[string]uint32, len(trackers))
	for k, c := range trackers {
		res[k] = c.ReadWriteCount.Load()
	}
	return res
}

// returns read write counts in order that they appear in the current buffer's list
func (l *IndexedList[T]) OrderedReadWriteCounts() []uint32 {
	l.sortedLock.RLock()
	defer l.sortedLock.RUnlock()
	var result []uint32
	for e := l.sorted.Front(); e != nil; e = e.Next() {
		result = append(result, l.readWriteCount(e))
	}
	return result
}

func (l *IndexedList[T]) Set(values []T) {
	l.sortedLock.Lock()
	defer l.sortedLock.Unlock()
	var newTrackers []readCountTracker[T]
	for _, v := range values {
		if c, ok := l.indexed.Get(v.Index()); ok {
			c.Element.Value = v
			c.ReadWriteCount.Add(1)
		} else {
			c := readCountTracker[T]{
				Element:        l.sorted.PushBack(v),
				ReadWriteCount: &atomic.Uint32{},
				index:          v.Index(),
			}
			c.ReadWriteCount.Add(1)
			newTrackers = append(newTrackers, c)
		}
	}
	if len(newTrackers) > 0 {
		l.indexed.Set(newTrackers)
	}
}

func (l *IndexedList[T]) PopWhere(predicate func(T) bool, amount int) []T {
	var result []T
	var toRemove []string
	l.sortedLock.Lock()
	for e := l.sorted.Front(); e != nil && amount > 0; {
		next := e.Next()
		asT := e.Value.(T)
		if predicate(asT) {
			result = append(result, asT)
			toRemove = append(toRemove, asT.Index())
			l.sorted.Remove(e)
			amount--
		}
		e = next
	}
	l.sortedLock.Unlock()
	if len(toRemove) > 0 {
		l.indexed.Remove(toRemove)
	}
	return result
}

func (l *IndexedList[T]) Len() int {
	l.sortedLock.RLock()
	defer l.sortedLock.RUnlock()
	return l.sorted.Len()
}

func (l *IndexedList[T]) Pop(amount int) []T {
	var result []T
	var toRemove []string
	l.sortedLock.Lock()
	for e := l.sorted.Front(); e != nil && amount > 0; {
		next := e.Next()
		asT := e.Value.(T)
		result = append(result, asT)
		toRemove = append(toRemove, asT.Index())
		l.sorted.Remove(e)
		amount--
		e = next
	}
	l.sortedLock.Unlock()
	if len(toRemove) > 0 {
		l.indexed.Remove(toRemove)
	}
	return result
}

// Insertion sort from the least read to the most read
func (l *IndexedList[T]) SortByReadCount() {
	l.sortedLock.Lock()
	defer l.sortedLock.Unlock()

	if l.sorted.Len() < 2 {
		return
	}

	for e := l.sorted.Front().Next(); e != nil; {
		next := e.Next()
		if l.readWriteCount(e) < l.readWriteCount(e.Prev()) {
			l.sorted.MoveBefore(e, e.Prev())
			for e.Prev() != nil && l.readWriteCount(e) < l.readWriteCount(e.Prev()) {
				l.sorted.MoveBefore(e, e.Prev())
			}
		}
		e = next
	}
}

func (l *IndexedList[T]) HalveReadCounts() {
	l.sortedLock.RLock()
	defer l.sortedLock.RUnlock()
	for e := l.sorted.Front(); e != nil; e = e.Next() {
		c, _ := l.indexed.Get(e.Value.(T).Index())
		for {
			old := c.ReadWriteCount.Load()
			if c.ReadWriteCount.CompareAndSwap(old, old/2) {
				break
			}
		}
	}
}
