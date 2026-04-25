package utils

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func ConnectToRedis(addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	return client
}

