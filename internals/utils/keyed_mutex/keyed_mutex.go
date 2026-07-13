package keyed_mutex

import (
	"sync"
	"time"
)

// This works only for a single-process setup.
// For distributed environments, use [distributedlock] instead.
func New() *KeyedRWMutex {
	km := &KeyedRWMutex{
		muMap: make(map[string]*mutex),
	}

	return km
}

type KeyedRWMutex struct {
	muMap map[string]*mutex
	mu    sync.Mutex
}

type mutex struct {
	mu      sync.RWMutex
	lastUse time.Time
	refs    int
}

func (me *KeyedRWMutex) Lock(key string) {
	me.mu.Lock()
	keyedMu, ok := me.muMap[key]
	if !ok {
		keyedMu = new(mutex)
		me.muMap[key] = keyedMu
	}
	keyedMu.refs++
	me.mu.Unlock()

	keyedMu.mu.Lock()
}

func (me *KeyedRWMutex) Unlock(key string) {
	me.mu.Lock()
	defer me.mu.Unlock()

	if keyedMu, ok := me.muMap[key]; ok {
		keyedMu.mu.Unlock()
		if keyedMu.refs > 0 {
			keyedMu.refs--
		}
		if keyedMu.refs == 0 {
			keyedMu.lastUse = time.Now()
		}
	}
}

func (me *KeyedRWMutex) RLock(key string) {
	me.mu.Lock()
	keyedMu, ok := me.muMap[key]
	if !ok {
		keyedMu = new(mutex)
		me.muMap[key] = keyedMu
	}
	keyedMu.refs++
	me.mu.Unlock()

	keyedMu.mu.RLock()
}

func (me *KeyedRWMutex) RUnlock(key string) {
	me.mu.Lock()
	defer me.mu.Unlock()

	if keyedMu, ok := me.muMap[key]; ok {
		keyedMu.mu.RUnlock()
		if keyedMu.refs > 0 {
			keyedMu.refs--
		}
		if keyedMu.refs == 0 {
			keyedMu.lastUse = time.Now()
		}
	}
}

func (me *KeyedRWMutex) ClearUnused(threshold time.Duration) {
	me.mu.Lock()
	defer me.mu.Unlock()

	for key, keyedMu := range me.muMap {
		if time.Since(keyedMu.lastUse) >= threshold && keyedMu.refs == 0 {
			delete(me.muMap, key)
		}
	}
}
