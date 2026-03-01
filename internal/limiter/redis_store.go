//nolint:godot,gocritic,revive
package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis key prefixes for scheduler
const (
	keyPrefixSession    = "sched:session:"
	keyPrefixResponse   = "sched:response:"
	keyPrefixConcurrent = "sched:concurrent:"
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

// --- Session Sticky Methods ---

// SetSessionAccount binds a session hash to an account ID
func (s *RedisStore) SetSessionAccount(ctx context.Context, provider, sessionHash, accountID string, ttl time.Duration) error {
	key := keyPrefixSession + provider + ":" + sessionHash
	return s.client.Set(ctx, key, accountID, ttl).Err()
}

// GetSessionAccount gets the account ID bound to a session hash
func (s *RedisStore) GetSessionAccount(ctx context.Context, provider, sessionHash string) (string, error) {
	key := keyPrefixSession + provider + ":" + sessionHash
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// DeleteSession deletes a session binding
func (s *RedisStore) DeleteSession(ctx context.Context, provider, sessionHash string) error {
	key := keyPrefixSession + provider + ":" + sessionHash
	return s.client.Del(ctx, key).Err()
}

// RefreshSessionTTL refreshes the TTL of a session binding
func (s *RedisStore) RefreshSessionTTL(ctx context.Context, provider, sessionHash string, ttl time.Duration) error {
	key := keyPrefixSession + provider + ":" + sessionHash
	return s.client.Expire(ctx, key, ttl).Err()
}

// --- Response Binding Methods ---

// SetResponseAccount binds a response ID to an account ID
func (s *RedisStore) SetResponseAccount(ctx context.Context, responseID, accountID string, ttl time.Duration) error {
	key := keyPrefixResponse + responseID
	return s.client.Set(ctx, key, accountID, ttl).Err()
}

// GetResponseAccount gets the account ID bound to a response ID
func (s *RedisStore) GetResponseAccount(ctx context.Context, responseID string) (string, error) {
	key := keyPrefixResponse + responseID
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// --- Concurrency Control Methods ---

// IncrConcurrent increments the concurrent count for an account
// Returns the new count after increment
func (s *RedisStore) IncrConcurrent(ctx context.Context, accountID string) (int64, error) {
	key := keyPrefixConcurrent + accountID
	return s.client.Incr(ctx, key).Result()
}

// DecrConcurrent decrements the concurrent count for an account
// Returns the new count after decrement
func (s *RedisStore) DecrConcurrent(ctx context.Context, accountID string) (int64, error) {
	key := keyPrefixConcurrent + accountID
	return s.client.Decr(ctx, key).Result()
}

// GetConcurrent gets the current concurrent count for an account
func (s *RedisStore) GetConcurrent(ctx context.Context, accountID string) (int64, error) {
	key := keyPrefixConcurrent + accountID
	val, err := s.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// SetConcurrentWithExpire sets the concurrent count with an expiration
// This is used to initialize the counter with a TTL to prevent orphaned locks
func (s *RedisStore) SetConcurrentWithExpire(ctx context.Context, accountID string, value int64, ttl time.Duration) error {
	key := keyPrefixConcurrent + accountID
	return s.client.Set(ctx, key, value, ttl).Err()
}

// TryAcquireSlot atomically tries to acquire a concurrency slot
// Returns (currentCount, acquired, error)
func (s *RedisStore) TryAcquireSlot(ctx context.Context, accountID string, maxConcurrency int) (int64, bool, error) {
	key := keyPrefixConcurrent + accountID

	// Use Lua script for atomic check-and-increment
	script := `
		local current = redis.call('GET', KEYS[1])
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end
		if current < tonumber(ARGV[1]) then
			local newval = redis.call('INCR', KEYS[1])
			redis.call('EXPIRE', KEYS[1], 3600)
			return {newval, 1}
		else
			return {current, 0}
		end
	`

	result, err := s.client.Eval(ctx, script, []string{key}, maxConcurrency).Result()
	if err != nil {
		return 0, false, err
	}

	// Parse result [currentCount, acquired]
	if arr, ok := result.([]interface{}); ok && len(arr) >= 2 {
		count, ok := arr[0].(int64)
		if !ok {
			return 0, false, nil
		}
		acquired, ok := arr[1].(int64)
		if !ok {
			return count, false, nil
		}
		return count, acquired == 1, nil
	}

	return 0, false, nil
}

// ReleaseSlot atomically releases a concurrency slot
func (s *RedisStore) ReleaseSlot(ctx context.Context, accountID string) error {
	key := keyPrefixConcurrent + accountID

	// Use Lua script for atomic decrement (but not below 0)
	script := `
		local current = redis.call('GET', KEYS[1])
		if current == false or tonumber(current) <= 0 then
			return 0
		else
			return redis.call('DECR', KEYS[1])
		end
	`

	_, err := s.client.Eval(ctx, script, []string{key}).Result()
	return err
}
