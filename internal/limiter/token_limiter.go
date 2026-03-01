//nolint:godot,exhaustive
package limiter

import (
	"context"
	"fmt"
	"time"
)

// TokenLimiter implements token-based rate limiting using sliding window algorithm
type TokenLimiter struct {
	store   *RedisStore
	period  Period
	limit   int64
	warning float64
}

// NewTokenLimiter creates a new token limiter
func NewTokenLimiter(store *RedisStore, period Period, limit int64, warningThreshold float64) *TokenLimiter {
	if warningThreshold <= 0 || warningThreshold > 1 {
		warningThreshold = 0.9
	}
	return &TokenLimiter{
		store:   store,
		period:  period,
		limit:   limit,
		warning: warningThreshold,
	}
}

// Allow checks if the request is allowed (does not consume tokens)
func (l *TokenLimiter) Allow(ctx context.Context, key string, cost int64) (bool, error) {
	usage, err := l.GetUsage(ctx, key)
	if err != nil {
		return false, err
	}
	return usage.Remaining >= cost, nil
}

// Consume consumes tokens from the limit
func (l *TokenLimiter) Consume(ctx context.Context, key string, cost int64) error {
	allowed, err := l.Allow(ctx, key, cost)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrQuotaExceeded
	}

	redisKey := l.buildKey(key)
	windowStart := l.getWindowStart()

	// Use Lua script for atomic operation
	script := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local cost = tonumber(ARGV[2])
		local ttl = tonumber(ARGV[3])

		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Get current count
		local current = redis.call('ZCARD', key)

		-- Add new entry with current timestamp as score
		local now = redis.call('TIME')
		local timestamp = tonumber(now[1]) + tonumber(now[2]) / 1000000
		redis.call('ZADD', key, timestamp, timestamp .. '-' .. math.random())

		-- Set TTL
		redis.call('EXPIRE', key, ttl)

		return current + 1
	`

	_, err = l.store.Eval(ctx, script, []string{redisKey},
		fmt.Sprintf("%d", windowStart.UnixNano()),
		fmt.Sprintf("%d", cost),
		fmt.Sprintf("%.0f", l.getTTL().Seconds()),
	)
	if err != nil {
		return fmt.Errorf("failed to consume tokens: %w", err)
	}

	// Increment by cost
	_, err = l.store.IncrBy(ctx, redisKey+":tokens", cost)
	if err != nil {
		return err
	}

	return nil
}

// GetUsage returns the current usage information
func (l *TokenLimiter) GetUsage(ctx context.Context, key string) (*Usage, error) {
	redisKey := l.buildKey(key)
	windowStart := l.getWindowStart()

	// Clean old entries
	err := l.store.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", windowStart.UnixNano()))
	if err != nil {
		return nil, err
	}

	// Get current token count
	used, err := l.store.Get(ctx, redisKey+":tokens")
	if err != nil {
		used = 0
	}

	remaining := l.limit - used
	if remaining < 0 {
		remaining = 0
	}

	percentUsed := float64(used) / float64(l.limit) * 100
	resetAt := l.getResetTime()

	usage := &Usage{
		Key:         key,
		Used:        used,
		Limit:       l.limit,
		Remaining:   remaining,
		ResetAt:     resetAt,
		Period:      l.period,
		PercentUsed: percentUsed,
	}

	// Set warning level
	if percentUsed >= 100 {
		usage.WarningLevel = "exceeded"
	} else if percentUsed >= l.warning*100 {
		usage.WarningLevel = "warning"
	}

	return usage, nil
}

// Reset resets the usage for a key
func (l *TokenLimiter) Reset(ctx context.Context, key string) error {
	redisKey := l.buildKey(key)
	if err := l.store.Del(ctx, redisKey); err != nil {
		return err
	}
	return l.store.Del(ctx, redisKey+":tokens")
}

// buildKey creates the Redis key for a given identifier
func (l *TokenLimiter) buildKey(key string) string {
	return fmt.Sprintf("token_limiter:%s:%s", l.period, key)
}

// getWindowStart returns the start of the current sliding window
func (l *TokenLimiter) getWindowStart() time.Time {
	now := time.Now()
	switch l.period {
	case PeriodMinute:
		return now.Add(-time.Minute)
	case PeriodHour:
		return now.Add(-time.Hour)
	case PeriodDay:
		return now.Add(-24 * time.Hour)
	case PeriodMonth:
		return now.AddDate(0, -1, 0)
	default:
		return now.Add(-time.Hour)
	}
}

// getResetTime returns when the current window resets
func (l *TokenLimiter) getResetTime() time.Time {
	now := time.Now()
	switch l.period {
	case PeriodMinute:
		return now.Truncate(time.Minute).Add(time.Minute)
	case PeriodHour:
		return now.Truncate(time.Hour).Add(time.Hour)
	case PeriodDay:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	case PeriodMonth:
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	default:
		return now.Add(time.Hour)
	}
}

// getTTL returns the TTL for keys
func (l *TokenLimiter) getTTL() time.Duration {
	d, err := PeriodDuration(l.period)
	if err != nil {
		d = time.Hour
	}
	return d + time.Minute // Add buffer
}
