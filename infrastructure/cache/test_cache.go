package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

type ITestCache interface {
	Set(ctx context.Context, key string, value interface{})
	Get(ctx context.Context, key string) (interface{}, error)
}

type TestCache struct {
	RedisClient *redis.Client
}

func NewTestCache(redisClient *redis.Client) ITestCache {
	return &TestCache{RedisClient: redisClient}
}

func (c *TestCache) Set(ctx context.Context, key string, value interface{}) {
	err := c.RedisClient.Set(ctx, key, value, time.Second*30)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while save redis")
	}
}

func (c *TestCache) Get(ctx context.Context, key string) (interface{}, error) {
	return c.RedisClient.Get(ctx, key).Result()
}
