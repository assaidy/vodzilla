package utils

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

func ConnectToPostgres(url string) *sql.DB {
	pool, err := sql.Open("postgres", url)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		panic(err)
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)
	pool.SetConnMaxIdleTime(1 * time.Minute)

	return pool
}
