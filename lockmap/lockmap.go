package lockmap

import (
	"sync"
)

type resource struct {
	lock sync.RWMutex
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

func (fs *LockMap) getOrCreate(index string) *resource {
	fs.mu.Lock()
	res, exists := fs.lockMap[index]
	if !exists {
		res = &resource{}
		fs.lockMap[index] = res
	}
	fs.mu.Unlock()
	return res
}

func (fs *LockMap) Rlock(index string) func() {
	res := fs.getOrCreate(index)
	res.lock.RLock()
	return func() {
		res.lock.RUnlock()
	}
}

func (fs *LockMap) Lock(index string) func() {
	res := fs.getOrCreate(index)
	res.lock.Lock()
	return func() {
		res.lock.Unlock()
	}
}
