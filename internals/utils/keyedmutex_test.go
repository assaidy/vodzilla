package utils

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMultipleRLocksOnSameKey(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			km.RLock(key)
			km.RUnlock(key)
		})
	}
	wg.Wait()
}

func TestLockBlocksOtherLock(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	km.Lock(key)

	locked := make(chan struct{})
	go func() {
		km.Lock(key)
		close(locked)
	}()

	select {
	case <-locked:
		t.Fatal("second Lock should have blocked")
	case <-time.After(10 * time.Millisecond):
	}

	km.Unlock(key)

	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("second Lock should have acquired after unlock")
	}
	km.Unlock(key)
}

func TestLockBlocksRLock(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	km.Lock(key)

	rlocked := make(chan struct{})
	go func() {
		km.RLock(key)
		close(rlocked)
	}()

	select {
	case <-rlocked:
		t.Fatal("RLock should have blocked while Lock is held")
	case <-time.After(10 * time.Millisecond):
	}

	km.Unlock(key)

	select {
	case <-rlocked:
	case <-time.After(time.Second):
		t.Fatal("RLock should have acquired after Unlock")
	}
	km.RUnlock(key)
}

func TestRLockBlocksLock(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	km.RLock(key)

	locked := make(chan struct{})
	go func() {
		km.Lock(key)
		close(locked)
	}()

	select {
	case <-locked:
		t.Fatal("Lock should have blocked while RLock is held")
	case <-time.After(10 * time.Millisecond):
	}

	km.RUnlock(key)

	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("Lock should have acquired after RUnlock")
	}
	km.Unlock(key)
}

func TestMultipleRLocksConcurrent(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	km.RLock(key)

	started := make(chan struct{})
	done := make(chan struct{})

	go func() {
		km.RLock(key)
		close(started)
		km.RUnlock(key)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("second RLock should not block while first RLock is held")
	}

	km.RUnlock(key)
	<-done
}

func TestDifferentKeysDontInterfere(t *testing.T) {
	var km KeyedRWMutex
	keyA, keyB := uuid.New(), uuid.New()

	km.Lock(keyA)

	locked := make(chan struct{})
	go func() {
		km.Lock(keyB)
		close(locked)
	}()

	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("Lock on keyB should not block while keyA is locked")
	}
	km.Unlock(keyB)
	km.Unlock(keyA)
}

func TestUnlockWithoutLockDoesNotPanic(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	km.Unlock(key)
	km.RUnlock(key)
}

func TestConcurrentReadWriteNoDeadlock(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			km.RLock(key)
			time.Sleep(time.Microsecond)
			km.RUnlock(key)
		})
	}

	for range 5 {
		wg.Go(func() {
			km.Lock(key)
			time.Sleep(time.Microsecond)
			km.Unlock(key)
		})
	}

	wg.Wait()
}

func TestManyKeys(t *testing.T) {
	var km KeyedRWMutex

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			key := uuid.New()
			km.Lock(key)
			km.Unlock(key)
			km.RLock(key)
			km.RUnlock(key)
		})
	}
	wg.Wait()
}

func TestRLockUpgradeNotAllowed(t *testing.T) {
	var km KeyedRWMutex
	key := uuid.New()

	km.RLock(key)

	locked := make(chan struct{})
	go func() {
		km.Lock(key)
		close(locked)
	}()

	select {
	case <-locked:
		t.Fatal("Lock should block when RLock is held (no upgrade)")
	case <-time.After(10 * time.Millisecond):
	}
	km.RUnlock(key)

	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("Lock should acquire after RUnlock")
	}
	km.Unlock(key)
}
