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
}

func (c *readCountTracker[T]) Index() string {
	return c.Element.Value.(T).Index()
}

func (c *readCountTracker[T]) Value() T {
	return c.Element.Value.(T)
}

type IndexedList[T element.Indexable] struct {
	indexed map[string]readCountTracker[T]
	sorted  list.List
	lock    sync.RWMutex
}

func NewIndexedList[T element.Indexable]() IndexedList[T] {
	return IndexedList[T]{
		indexed: map[string]readCountTracker[T]{},
		sorted:  list.List{},
		lock:    sync.RWMutex{},
	}
}

func (l *IndexedList[T]) readWriteCount(e *list.Element) uint32 {
	return l.indexed[e.Value.(T).Index()].ReadWriteCount.Load()
}

func (l *IndexedList[T]) Remove(indexes []string) int {
	l.lock.Lock()
	defer l.lock.Unlock()
	count := 0
	for _, index := range indexes {
		if c, ok := l.indexed[index]; ok {
			l.sorted.Remove(c.Element)
			delete(l.indexed, index)
			count++
		}
	}
	return count
}

func (l *IndexedList[T]) Gets(indexes []string) map[string]T {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res := make(map[string]T)
	for _, index := range indexes {
		if c, ok := l.indexed[index]; ok {
			c.ReadWriteCount.Add(1)
			res[index] = c.Value()
		}
	}
	return res
}

func (l *IndexedList[T]) Get(index string) *T {
	l.lock.RLock()
	defer l.lock.RUnlock()
	if c, ok := l.indexed[index]; ok {
		c.ReadWriteCount.Add(1)
		val := c.Value()
		return &val
	}
	return nil
}

// returns the read write count for the given indexes; does not affect the count
func (l *IndexedList[T]) ReadWriteCounts(indexes []string) map[string]uint32 {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res := make(map[string]uint32)
	for _, index := range indexes {
		if c, ok := l.indexed[index]; ok {
			res[index] = c.ReadWriteCount.Load()
		}
	}
	return res
}

// returns read write counts in order that they appear in the current buffer's list
func (l *IndexedList[T]) OrderedReadWriteCounts() []uint32 {
	l.lock.RLock()
	defer l.lock.RUnlock()
	var result []uint32
	for e := l.sorted.Front(); e != nil; e = e.Next() {
		result = append(result, l.readWriteCount(e))
	}
	return result
}

func (l *IndexedList[T]) Set(values []T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	for _, v := range values {
		if c, ok := l.indexed[v.Index()]; ok {
			c.Element.Value = v
			c.ReadWriteCount.Add(1)
		} else {
			c := readCountTracker[T]{
				Element:        l.sorted.PushBack(v),
				ReadWriteCount: &atomic.Uint32{},
			}
			c.ReadWriteCount.Add(1)
			l.indexed[v.Index()] = c
		}
	}
}

func (l *IndexedList[T]) PopWhere(predicate func(T) bool, amount int) []T {
	l.lock.Lock()
	defer l.lock.Unlock()
	var result []T
	for e := l.sorted.Front(); e != nil && amount > 0; {
		next := e.Next()
		asT := e.Value.(T)
		if predicate(asT) {
			result = append(result, asT)
			delete(l.indexed, asT.Index())
			l.sorted.Remove(e)
			amount--
		}
		e = next
	}
	return result
}

func (l *IndexedList[T]) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.sorted.Len()
}

func (l *IndexedList[T]) Pop(amount int) []T {
	l.lock.Lock()
	defer l.lock.Unlock()
	var result []T
	for e := l.sorted.Front(); e != nil && amount > 0; {
		next := e.Next()
		asT := e.Value.(T)
		result = append(result, asT)
		delete(l.indexed, asT.Index())
		l.sorted.Remove(e)
		amount--
		e = next
	}
	return result
}

// Insertion sort from the least read to the most read
func (l *IndexedList[T]) SortByReadCount() {
	l.lock.Lock()
	defer l.lock.Unlock()

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

func (l *IndexedList[T]) ClearReadCounts() {
	l.lock.Lock()
	defer l.lock.Unlock()
	for e := l.sorted.Front(); e != nil; e = e.Next() {
		l.indexed[e.Value.(T).Index()].ReadWriteCount.Store(0)
	}
}
