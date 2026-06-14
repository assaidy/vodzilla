package keyed_mutex

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
)

func TestMultipleRLocksOnSameKey(t *testing.T) {
	km := New()
	key := uuid.New().String()

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
	km := New()
	key := uuid.New().String()

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
	km := New()
	key := uuid.New().String()

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
	km := New()
	key := uuid.New().String()

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
	km := New()
	key := uuid.New().String()

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
	km := New()
	keyA, keyB := uuid.New().String(), uuid.New().String()

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
	km := New()
	key := uuid.New().String()

	km.Unlock(key)
	km.RUnlock(key)
}

func TestConcurrentReadWriteNoDeadlock(t *testing.T) {
	km := New()
	key := uuid.New().String()

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
	km := New()

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			key := uuid.New().String()
			km.Lock(key)
			km.Unlock(key)
			km.RLock(key)
			km.RUnlock(key)
		})
	}
	wg.Wait()
}

func TestRLockUpgradeNotAllowed(t *testing.T) {
	km := New()
	key := uuid.New().String()

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

func TestClearUnusedRemovesOldReleasedEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := New()
		key := uuid.New().String()

		km.Lock(key)
		km.Unlock(key)

		time.Sleep(30 * time.Minute)

		km.ClearUnused(0)

		if _, ok := km.muMap[key]; ok {
			t.Fatal("ClearUnused should remove old released entry")
		}
	})
}

func TestClearUnusedSkipsRecentlyUsed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := New()
		key := uuid.New().String()

		km.Lock(key)
		km.Unlock(key)

		time.Sleep(30 * time.Minute)

		km.ClearUnused(time.Hour)

		if _, ok := km.muMap[key]; !ok {
			t.Fatal("ClearUnused should keep recently used entry")
		}
	})
}

func TestClearUnusedSkipsCurrentlyLocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := New()
		key := uuid.New().String()

		km.Lock(key)

		km.ClearUnused(time.Hour)

		if _, ok := km.muMap[key]; !ok {
			t.Fatal("ClearUnused should keep currently locked entry")
		}

		km.ClearUnused(0)

		if _, ok := km.muMap[key]; !ok {
			t.Fatal("ClearUnused should keep currently locked entry even with zero threshold")
		}

		km.Unlock(key)
	})
}

func TestClearUnusedSkipsCurrentlyRLocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := New()
		key := uuid.New().String()

		km.RLock(key)

		km.ClearUnused(time.Hour)

		if _, ok := km.muMap[key]; !ok {
			t.Fatal("ClearUnused should keep currently read-locked entry")
		}

		km.ClearUnused(0)

		if _, ok := km.muMap[key]; !ok {
			t.Fatal("ClearUnused should keep currently read-locked entry even with zero threshold")
		}

		km.RUnlock(key)
	})
}

func TestClearUnusedMixedEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := New()
		oldKey := uuid.New().String()
		recentKey := uuid.New().String()
		lockedKey := uuid.New().String()

		km.Lock(oldKey)
		km.Unlock(oldKey)

		time.Sleep(30 * time.Minute)

		km.Lock(recentKey)
		km.Unlock(recentKey)

		km.Lock(lockedKey)

		km.ClearUnused(0)

		_, oldOk := km.muMap[oldKey]
		_, recentOk := km.muMap[recentKey]
		_, lockedOk := km.muMap[lockedKey]

		if oldOk {
			t.Fatal("ClearUnused should remove old released entry")
		}
		if recentOk {
			t.Fatal("ClearUnused should remove recently released entry with zero threshold")
		}
		if !lockedOk {
			t.Fatal("ClearUnused should keep currently locked entry")
		}

		km.Unlock(lockedKey)
	})
}

func TestClearUnusedSkipsEntryLockedByAnotherGoroutine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		km := New()
		key := uuid.New().String()

		km.Lock(key)

		done := make(chan struct{})
		go func() {
			km.ClearUnused(0)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("ClearUnused should not block on locked entry")
		}

		if _, ok := km.muMap[key]; !ok {
			t.Fatal("ClearUnused should keep entry locked by another goroutine")
		}

		km.Unlock(key)
	})
}
