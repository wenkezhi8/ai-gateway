package cache

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type redisCommandExecutor interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
}

// EnsureRedisStackCapabilities validates RediSearch and RedisJSON command support.
func EnsureRedisStackCapabilities(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return ensureRedisStackCapabilitiesWithExecutor(ctx, client)
}

func ensureRedisStackCapabilitiesWithExecutor(ctx context.Context, executor redisCommandExecutor) error {
	if executor == nil {
		return fmt.Errorf("redis command executor is nil")
	}

	if err := executor.Do(ctx, "FT._LIST").Err(); err != nil {
		if isUnknownRedisCommand(err) {
			return fmt.Errorf("RediSearch module is required")
		}
		return fmt.Errorf("check RediSearch capability failed: %w", err)
	}

	jsonErr := executor.Do(ctx, "JSON.GET", "aigw:redis-stack:capability", "$").Err()
	if jsonErr != nil && jsonErr != redis.Nil {
		if isUnknownRedisCommand(jsonErr) {
			return fmt.Errorf("RedisJSON module is required")
		}
		return fmt.Errorf("check RedisJSON capability failed: %w", jsonErr)
	}

	return nil
}

func isUnknownRedisCommand(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unknown command")
}
