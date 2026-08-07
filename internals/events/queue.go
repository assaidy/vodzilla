package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Publish(ctx context.Context, rdb *redis.Client, eventName string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal %q payload: %w", eventName, err)
	}
	if err := rdb.LPush(ctx, queueName(eventName), body).Err(); err != nil {
		return fmt.Errorf("failed to publish %q event: %w", eventName, err)
	}
	return nil
}

func Consume(ctx context.Context, rdb *redis.Client, eventName string, payload any) (bool, error) {
	result, err := rdb.BRPop(ctx, 5*time.Second, queueName(eventName)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) || ctx.Err() != nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to pop from %q queue: %w", eventName, err)
	}
	if err := json.Unmarshal([]byte(result[1]), payload); err != nil {
		return false, fmt.Errorf("failed to unmarshal %q event payload: %w", eventName, err)
	}
	return true, nil
}
