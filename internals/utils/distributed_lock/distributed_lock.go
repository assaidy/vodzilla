package distributed_lock

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	lockExpiration = 5 * time.Second
	retryInterval  = 50 * time.Millisecond
)

type DistributedLock struct {
	Id     string
	redis  *redis.Client
	logger *slog.Logger
}

func New(redis *redis.Client, logger *slog.Logger) *DistributedLock {
	dl := &DistributedLock{
		Id:     uuid.New().String(),
		redis:  redis,
		logger: logger,
	}

	return dl
}

func lockEntry(key string) string {
	return fmt.Sprintf("lock:%s", key)
}

func rlockEntry(key string) string {
	return fmt.Sprintf("rlock:%s", key)
}

var tryLockScript = redis.NewScript(`
	local lock_key = KEYS[1]
	local reader_key = KEYS[2]
	local lock_id = ARGV[1]
	local ttl_ms = ARGV[2]
	local now = tonumber(ARGV[3])
	if redis.call("ZCOUNT", reader_key, "(" .. now, "+inf") > 0 then
		return 0
	end
	local ok = redis.call("SET", lock_key, lock_id, "NX", "PX", ttl_ms)
	if ok then
		return 1
	end
	return 0
`)

var renewLockScript = redis.NewScript(`
	local lock_key = KEYS[1]
	local lock_id = ARGV[1]
	local ttl_ms = ARGV[2]
	if redis.call("GET", lock_key) == lock_id then
		return redis.call("PEXPIRE", lock_key, ttl_ms)
	end
	return 0
`)

func (me *DistributedLock) TryLock(ctx context.Context, key string) (bool, error) {
	if n, err := tryLockScript.Run(
		ctx,
		me.redis,
		[]string{lockEntry(key), rlockEntry(key)},
		me.Id,
		lockExpiration.Milliseconds(),
		time.Now().UnixMilli(),
	).Int(); err != nil {
		return false, fmt.Errorf("try lock key=%q: %w", key, err)
	} else if n == 0 {
		return false, nil
	}

	go func() {
		ticker := time.NewTicker(lockExpiration / 3)
		defer ticker.Stop()

		for range ticker.C {
			if n, err := renewLockScript.Run(
				ctx,
				me.redis,
				[]string{lockEntry(key)},
				me.Id,
				lockExpiration.Milliseconds(),
			).Int(); err != nil {
				me.logger.Error("failed to renew lock", "key", key, "error", err)
				return
			} else if n == 0 {
				return
			}
		}
	}()

	return true, nil
}

func (me *DistributedLock) Lock(ctx context.Context, key string) error {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		if ok, err := me.TryLock(ctx, key); err != nil {
			return fmt.Errorf("lock key=%q: %w", key, err)
		} else if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("lock key=%q: %w", key, ctx.Err())
		case <-ticker.C:
		}
	}
}

var unlockScript = redis.NewScript(`
	local lock_key = KEYS[1]
	local lock_id = ARGV[1]
	if redis.call("GET", lock_key) == lock_id then
		return redis.call("DEL", lock_key)
	end
	return 0
`)

func (me *DistributedLock) Unlock(ctx context.Context, key string) error {
	_, err := unlockScript.Run(ctx, me.redis, []string{lockEntry(key)}, me.Id).Result()
	if err != nil {
		return fmt.Errorf("unlock key=%q: %w", key, err)
	}
	return nil
}

var tryRlockScript = redis.NewScript(`
	local writer_key = KEYS[1]
	local reader_key = KEYS[2]
	if redis.call("EXISTS", writer_key) == 1 then
		return 0
	end
	redis.call("ZREMRANGEBYSCORE", reader_key, "-inf", ARGV[2])
	redis.call("ZADD", reader_key, ARGV[3], ARGV[1])
	return 1
`)

var renewRLockScript = redis.NewScript(`
	local reader_key = KEYS[1]
	local reader_id = ARGV[1]
	local new_score = ARGV[2]
	if redis.call("ZSCORE", reader_key, reader_id) then
		return redis.call("ZADD", reader_key, new_score, reader_id)
	end
	return 0
`)

func (me *DistributedLock) TryRLock(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixMilli()
	if n, err := tryRlockScript.Run(
		ctx,
		me.redis,
		[]string{lockEntry(key), rlockEntry(key)},
		me.Id,
		now,
		now+lockExpiration.Milliseconds(),
	).Int(); err != nil {
		return false, fmt.Errorf("try rlock key=%q: %w", key, err)
	} else if n == 0 {
		return false, nil
	}

	go func() {
		ticker := time.NewTicker(lockExpiration / 3)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now().UnixMilli()
			if n, err := renewRLockScript.Run(
				ctx,
				me.redis,
				[]string{rlockEntry(key)},
				me.Id,
				now+lockExpiration.Milliseconds(),
			).Int(); err != nil {
				me.logger.Error("failed to renew rlock", "key", key, "error", err)
				return
			} else if n == 0 {
				return
			}
		}
	}()

	return true, nil
}

func (me *DistributedLock) RLock(ctx context.Context, key string) error {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		if ok, err := me.TryRLock(ctx, key); err != nil {
			return fmt.Errorf("rlock key=%q: %w", key, err)
		} else if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("rlock key=%q: %w", key, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (me *DistributedLock) RUnlock(ctx context.Context, key string) error {
	if err := me.redis.ZRem(ctx, rlockEntry(key), me.Id).Err(); err != nil {
		return fmt.Errorf("runlock key=%q: %w", key, err)
	}
	return nil
}
