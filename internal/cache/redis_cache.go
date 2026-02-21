package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	ErrRedisNotAvailable = errors.New("redis not available")
)

// RedisCache implements Cache interface using Redis with MessagePack serialization
type RedisCache struct {
	client *redis.Client
	prefix string
}

// RedisConfig holds Redis cache configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	Prefix   string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "ai-gateway:"
	}

	return &RedisCache{
		client: client,
		prefix: prefix,
	}, nil
}

// Get retrieves a value from Redis and deserializes using MessagePack
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.prefix + key
	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		return err
	}

	return msgpack.Unmarshal(data, dest)
}

// Set stores a value in Redis using MessagePack serialization
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.prefix + key

	data, err := msgpack.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, fullKey, data, ttl).Err()
}

// Delete removes a key from Redis
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// Exists checks if a key exists in Redis
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key
	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteByPattern removes all keys matching a pattern
func (c *RedisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	fullPattern := c.prefix + pattern

	var cursor uint64
	for {
		var keys []string
		var err error

		keys, cursor, err = c.client.Scan(ctx, cursor, fullPattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// GetClient returns the underlying Redis client for advanced operations
func (c *RedisCache) GetClient() *redis.Client {
	return c.client
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// SetNX sets a value only if the key does not exist (atomic operation)
func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	fullKey := c.prefix + key

	data, err := msgpack.Marshal(value)
	if err != nil {
		return false, err
	}

	return c.client.SetNX(ctx, fullKey, data, ttl).Result()
}

// GetOrSet returns the cached value or sets and returns the computed value
func (c *RedisCache) GetOrSet(ctx context.Context, key string, dest interface{}, compute func() (interface{}, error), ttl time.Duration) error {
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != ErrNotFound {
		return err
	}

	// Compute new value
	value, err := compute()
	if err != nil {
		return err
	}

	// Cache the computed value
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	// Copy value to dest by marshaling and unmarshaling
	data, err := msgpack.Marshal(value)
	if err != nil {
		return err
	}

	return msgpack.Unmarshal(data, dest)
}
