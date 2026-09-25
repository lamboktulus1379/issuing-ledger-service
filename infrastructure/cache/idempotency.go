package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrIdempotencyConflict = errors.New("idempotency request is already in progress or completed")

type IdempotencyStore interface {
	AcquireLock(ctx context.Context, key string, ttl time.Duration) error
	Complete(ctx context.Context, key string, result []byte, ttl time.Duration) error
	GetResult(ctx context.Context, key string) ([]byte, error)
	ReleaseLock(ctx context.Context, key string) error
}

type redisIdempotencyStore struct {
	client *redis.Client
}

func NewIdempotencyStore(client *redis.Client) IdempotencyStore {
	return &redisIdempotencyStore{client: client}
}

const (
	idempotencyKeyPrefix  = "idemp:req:"
	idempotencyInProgress = "IN_PROGRESS"
)

// Completion is compare-and-set: a worker may publish a result only while the
// request still owns the IN_PROGRESS state. The check and replacement execute
// atomically inside Redis, so an expired lock cannot be overwritten by a stale worker.
var completeIdempotencyScript = redis.NewScript(`
	if redis.call("GET", KEYS[1]) ~= ARGV[1] then
		return 0
	end
	redis.call("SET", KEYS[1], ARGV[2], "PX", ARGV[3])
	return 1
`)

func (s *redisIdempotencyStore) AcquireLock(ctx context.Context, key string, ttl time.Duration) error {
	// SETNX and the initial TTL are a single Redis operation. Concurrent retries
	// therefore leave exactly one request holding the IN_PROGRESS marker.
	acquired, err := s.client.SetNX(ctx, idempotencyKey(key), idempotencyInProgress, ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire idempotency lock: %w", err)
	}
	if !acquired {
		return ErrIdempotencyConflict
	}
	return nil
}

func (s *redisIdempotencyStore) Complete(ctx context.Context, key string, result []byte, ttl time.Duration) error {
	updated, err := completeIdempotencyScript.Run(
		ctx,
		s.client,
		[]string{idempotencyKey(key)},
		idempotencyInProgress,
		string(result),
		ttl.Milliseconds(),
	).Int()
	if err != nil {
		return fmt.Errorf("failed to complete idempotency request: %w", err)
	}
	if updated == 0 {
		return ErrIdempotencyConflict
	}
	return nil
}

func (s *redisIdempotencyStore) GetResult(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.Get(ctx, idempotencyKey(key)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency result: %w", err)
	}
	if string(result) == idempotencyInProgress {
		return nil, ErrIdempotencyConflict
	}
	return result, nil
}

func (s *redisIdempotencyStore) ReleaseLock(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, idempotencyKey(key)).Err(); err != nil {
		return fmt.Errorf("failed to release idempotency lock: %w", err)
	}
	return nil
}

func idempotencyKey(key string) string {
	return idempotencyKeyPrefix + key
}
