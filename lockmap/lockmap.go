package lockmap

import (
	"sync"
	"sync/atomic"
)

type resource struct {
	refCount atomic.Int32
	lock     sync.RWMutex
}

type LockMap struct {
	lockMap map[string]*resource
	mu      sync.Mutex
}

func New() *LockMap {
	return &LockMap{
		lockMap: make(map[string]*resource),
	}
}

func (fs *LockMap) Rlock(index string) func() {
	fs.mu.Lock()
	res, exists := fs.lockMap[index]
	if !exists {
		fs.lockMap[index] = &resource{}
		res = fs.lockMap[index]
	} else {
		res.refCount.Add(1)
	}
	fs.mu.Unlock()
	res.lock.RLock()
	return func() {
		if res.refCount.Add(-1) == 0 {
			fs.mu.Lock()
			delete(fs.lockMap, index)
			fs.mu.Unlock()
		}
		res.lock.RUnlock()
	}
}

func (fs *LockMap) Lock(index string) func() {
	fs.mu.Lock()
	res, exists := fs.lockMap[index]
	if !exists {
		res = &resource{}
		fs.lockMap[index] = res
	}
	fs.mu.Unlock()
	res.lock.Lock()
	return func() {
		fs.mu.Lock()
		delete(fs.lockMap, index)
		fs.mu.Unlock()
		res.lock.Unlock()
	}
}
