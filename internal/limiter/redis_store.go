package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements the Store interface using Redis
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis store
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Get retrieves a value from Redis
func (s *RedisStore) Get(ctx context.Context, key string) (int64, error) {
	val, err := s.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// Set stores a value in Redis with expiration
func (s *RedisStore) Set(ctx context.Context, key string, value int64, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, expiration).Err()
}

// Incr increments a value in Redis
func (s *RedisStore) Incr(ctx context.Context, key string) (int64, error) {
	return s.client.Incr(ctx, key).Result()
}

// IncrBy increments a value by a specific amount
func (s *RedisStore) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return s.client.IncrBy(ctx, key, value).Result()
}

// Decr decrements a value in Redis
func (s *RedisStore) Decr(ctx context.Context, key string) (int64, error) {
	return s.client.Decr(ctx, key).Result()
}

// DecrBy decrements a value by a specific amount
func (s *RedisStore) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return s.client.DecrBy(ctx, key, value).Result()
}

// Expire sets an expiration on a key
func (s *RedisStore) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return s.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the time to live for a key
func (s *RedisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return s.client.TTL(ctx, key).Result()
}

// Del deletes a key
func (s *RedisStore) Del(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (s *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	val, err := s.client.Exists(ctx, key).Result()
	return val > 0, err
}

// ZAdd adds a member to a sorted set (for sliding window)
func (s *RedisStore) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return s.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRangeByScore returns members in a score range
func (s *RedisStore) ZRangeByScore(ctx context.Context, key string, min, max string, opts *redis.ZRangeBy) ([]string, error) {
	if opts == nil {
		opts = &redis.ZRangeBy{}
	}
	return s.client.ZRangeByScore(ctx, key, opts).Result()
}

// ZRemRangeByScore removes members in a score range
func (s *RedisStore) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	return s.client.ZRemRangeByScore(ctx, key, min, max).Err()
}

// ZCard returns the number of members in a sorted set
func (s *RedisStore) ZCard(ctx context.Context, key string) (int64, error) {
	return s.client.ZCard(ctx, key).Result()
}

// ZScore returns the score of a member
func (s *RedisStore) ZScore(ctx context.Context, key, member string) (float64, error) {
	return s.client.ZScore(ctx, key, member).Result()
}

// Eval executes a Lua script
func (s *RedisStore) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return s.client.Eval(ctx, script, keys, args...).Result()
}

// Ping checks the connection
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}
