package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewCache(ctx context.Context, addr, username, password string) (*redis.Client, error) {
	rds := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       0,
	})

	_, err := rds.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	fmt.Println("Redis connected")
	return rds, nil
}
