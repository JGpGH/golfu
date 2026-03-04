package lockmap

import (
	"sync"
)

type resource struct {
	lock sync.RWMutex
	refs int
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
	res.refs++
	fs.mu.Unlock()
	return res
}

func (fs *LockMap) release(index string) {
	fs.mu.Lock()
	res := fs.lockMap[index]
	res.refs--
	if res.refs == 0 {
		delete(fs.lockMap, index)
	}
	fs.mu.Unlock()
}

func (fs *LockMap) Len() int {
	fs.mu.Lock()
	n := len(fs.lockMap)
	fs.mu.Unlock()
	return n
}

func (fs *LockMap) Rlock(index string) func() {
	res := fs.getOrCreate(index)
	res.lock.RLock()
	return func() {
		res.lock.RUnlock()
		fs.release(index)
	}
}

func (fs *LockMap) Lock(index string) func() {
	res := fs.getOrCreate(index)
	res.lock.Lock()
	return func() {
		res.lock.Unlock()
		fs.release(index)
	}
}
