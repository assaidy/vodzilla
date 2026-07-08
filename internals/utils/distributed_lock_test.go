package utils

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testRedis *redis.Client

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:8.6-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("* Ready to accept connections tcp"),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}
	port, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		panic(err)
	}
	testRedis = redis.NewClient(&redis.Options{
		Addr: "localhost:" + port.Port(),
	})
	code := m.Run()
	testRedis.Close()
	redisC.Terminate(ctx)
	os.Exit(code)
}

func newDL() *DistributedLock {
	return NewDistributedLock(testRedis, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))
}

func TestTryLockUnlock(t *testing.T) {
	dl := newDL()
	key := "test_try_lock"
	ctx := context.Background()

	ok, err := dl.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected to acquire lock")
	}

	ok, err = dl.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected not to acquire lock when already held")
	}

	if err := dl.Unlock(ctx, key); err != nil {
		t.Fatal(err)
	}

	ok, err = dl.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected to acquire lock after unlock")
	}
	dl.Unlock(ctx, key)
}

func TestLockBlocksOtherLock(t *testing.T) {
	dl := newDL()
	key := "test_lock_blocks"
	ctx := context.Background()

	if err := dl.Lock(ctx, key); err != nil {
		t.Fatal(err)
	}

	gotLock := make(chan struct{})
	go func() {
		dl.Lock(ctx, key)
		close(gotLock)
	}()

	select {
	case <-gotLock:
		t.Fatal("second Lock should block")
	case <-time.After(200 * time.Millisecond):
	}

	dl.Unlock(ctx, key)

	select {
	case <-gotLock:
	case <-time.After(time.Second):
		t.Fatal("second Lock should acquire after unlock")
	}
	dl.Unlock(ctx, key)
}

func TestLockBlocksRLock(t *testing.T) {
	dl := newDL()
	key := "test_lock_blocks_rlock"
	ctx := context.Background()

	dl.Lock(ctx, key)

	gotRLock := make(chan struct{})
	go func() {
		dl.RLock(ctx, key)
		close(gotRLock)
	}()

	select {
	case <-gotRLock:
		t.Fatal("RLock should block while Lock is held")
	case <-time.After(200 * time.Millisecond):
	}

	dl.Unlock(ctx, key)

	select {
	case <-gotRLock:
	case <-time.After(time.Second):
		t.Fatal("RLock should acquire after Unlock")
	}
	dl.RUnlock(ctx, key)
}

func TestRLockBlocksLock(t *testing.T) {
	dl := newDL()
	key := "test_rlock_blocks_lock"
	ctx := context.Background()

	dl.RLock(ctx, key)

	gotLock := make(chan struct{})
	go func() {
		dl.Lock(ctx, key)
		close(gotLock)
	}()

	select {
	case <-gotLock:
		t.Fatal("Lock should block while RLock is held")
	case <-time.After(200 * time.Millisecond):
	}

	dl.RUnlock(ctx, key)

	select {
	case <-gotLock:
	case <-time.After(time.Second):
		t.Fatal("Lock should acquire after RUnlock")
	}
	dl.Unlock(ctx, key)
}

func TestMultipleRLocksConcurrent(t *testing.T) {
	dl := newDL()
	key := "test_multiple_rlocks"
	ctx := context.Background()

	dl.RLock(ctx, key)

	gotRLock := make(chan struct{})
	done := make(chan struct{})
	go func() {
		dl.RLock(ctx, key)
		close(gotRLock)
		dl.RUnlock(ctx, key)
		close(done)
	}()

	select {
	case <-gotRLock:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("second RLock should not block while first RLock is held")
	}

	dl.RUnlock(ctx, key)
	<-done
}

func TestDifferentKeysDontInterfere(t *testing.T) {
	dl := newDL()
	ctx := context.Background()

	dl.Lock(ctx, "key_a")

	gotLock := make(chan struct{})
	go func() {
		dl.Lock(ctx, "key_b")
		close(gotLock)
	}()

	select {
	case <-gotLock:
	case <-time.After(time.Second):
		t.Fatal("Lock on key_b should not block while key_a is locked")
	}
	dl.Unlock(ctx, "key_b")
	dl.Unlock(ctx, "key_a")
}

func TestUnlockWithoutLockDoesNotError(t *testing.T) {
	dl := newDL()
	ctx := context.Background()

	if err := dl.Unlock(ctx, "no_lock"); err != nil {
		t.Fatal(err)
	}
	if err := dl.RUnlock(ctx, "no_rlock"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentReadWriteNoDeadlock(t *testing.T) {
	dl := newDL()
	key := "test_concurrent_rw"
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			dl.RLock(ctx, key)
			time.Sleep(time.Microsecond)
			dl.RUnlock(ctx, key)
		})
	}

	for range 3 {
		wg.Go(func() {
			dl.Lock(ctx, key)
			time.Sleep(time.Microsecond)
			dl.Unlock(ctx, key)
		})
	}

	wg.Wait()
}

func TestTryLockFailsWithActiveReaders(t *testing.T) {
	dl := newDL()
	key := "test_trylock_readers"
	ctx := context.Background()

	dl.RLock(ctx, key)

	ok, err := dl.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("TryLock should fail while readers are active")
	}

	dl.RUnlock(ctx, key)

	ok, err = dl.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("TryLock should succeed after readers release")
	}
	dl.Unlock(ctx, key)
}

func TestTryRLockFailsWithActiveWriter(t *testing.T) {
	dl := newDL()
	key := "test_tryrlock_writer"
	ctx := context.Background()

	dl.Lock(ctx, key)

	ok, err := dl.TryRLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("TryRLock should fail while writer is active")
	}

	dl.Unlock(ctx, key)

	ok, err = dl.TryRLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("TryRLock should succeed after writer releases")
	}
	dl.RUnlock(ctx, key)
}

func TestContextCancellation(t *testing.T) {
	dl := newDL()
	key := "test_cancel"
	ctx := context.Background()

	dl.Lock(ctx, key)

	cancelCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := dl.Lock(cancelCtx, key)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}

	dl.Unlock(ctx, key)
}

func TestRenewalKeepsLockAlive(t *testing.T) {
	dl := newDL()
	key := "test_renewal"
	ctx := context.Background()

	dl.Lock(ctx, key)
	defer dl.Unlock(ctx, key)

	lockKey := "lock:" + key
	ttl1, _ := testRedis.TTL(ctx, lockKey).Result()
	time.Sleep(2 * time.Second)
	ttl2, _ := testRedis.TTL(ctx, lockKey).Result()

	if ttl2 < ttl1 {
		t.Fatalf("TTL should have been renewed: before=%v after=%v", ttl1, ttl2)
	}
}
