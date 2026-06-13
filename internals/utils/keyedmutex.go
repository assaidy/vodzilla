package utils

import (
	"sync"
	"time"
)

// TODO: this works only for a single-process setup.
// impl a cross-process lock using a storage such as redis or postgres temp table.
func NewKeyedMutex() *KeyedMutex {
	km := &KeyedMutex{
		muMap: make(map[string]*mutex),
	}

	return km
}

type KeyedMutex struct {
	muMap map[string]*mutex
	mu    sync.Mutex
}

type mutex struct {
	mu      sync.RWMutex
	lastUse time.Time
	refs    int
}

func (me *KeyedMutex) Lock(key string) {
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

func (me *KeyedMutex) Unlock(key string) {
	me.mu.Lock()

	if keyedMu, ok := me.muMap[key]; ok {
		keyedMu.mu.Unlock()
		if keyedMu.refs > 0 {
			keyedMu.refs--
		}
		if keyedMu.refs == 0 {
			keyedMu.lastUse = time.Now()
		}
	}

	me.mu.Unlock()
}

func (me *KeyedMutex) RLock(key string) {
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

func (me *KeyedMutex) RUnlock(key string) {
	me.mu.Lock()

	if keyedMu, ok := me.muMap[key]; ok {
		keyedMu.mu.RUnlock()
		if keyedMu.refs > 0 {
			keyedMu.refs--
		}
		if keyedMu.refs == 0 {
			keyedMu.lastUse = time.Now()
		}
	}

	me.mu.Unlock()
}

func (me *KeyedMutex) ClearUnused(threshold time.Duration) {
	me.mu.Lock()
	defer me.mu.Unlock()

	for key, keyedMu := range me.muMap {
		if time.Since(keyedMu.lastUse) >= threshold && keyedMu.refs == 0 {
			delete(me.muMap, key)
		}
	}
}
