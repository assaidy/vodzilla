package utils

import (
	"sync"

	"github.com/google/uuid"
)

type KeyedRWMutex struct {
	m sync.Map
}

func (km *KeyedRWMutex) Lock(key uuid.UUID) {
	mu, _ := km.m.LoadOrStore(key, new(sync.RWMutex))
	mu.(*sync.RWMutex).Lock()
}

func (km *KeyedRWMutex) Unlock(key uuid.UUID) {
	if mu, ok := km.m.Load(key); ok {
		mu.(*sync.RWMutex).Unlock()
	}
}

func (km *KeyedRWMutex) RLock(key uuid.UUID) {
	mu, _ := km.m.LoadOrStore(key, new(sync.RWMutex))
	mu.(*sync.RWMutex).RLock()
}

func (km *KeyedRWMutex) RUnlock(key uuid.UUID) {
	if mu, ok := km.m.Load(key); ok {
		mu.(*sync.RWMutex).RUnlock()
	}
}
