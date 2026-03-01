//nolint:godot,gocritic,revive,goconst
package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RPMLimiter implements request-per-minute rate limiting using token bucket algorithm
type RPMLimiter struct {
	store    *RedisStore
	limiters sync.Map // Local rate limiters for fast path
	rpm      int
	burst    int
	warning  float64
}

// NewRPMLimiter creates a new RPM limiter
func NewRPMLimiter(store *RedisStore, rpm, burst int, warningThreshold float64) *RPMLimiter {
	if warningThreshold <= 0 || warningThreshold > 1 {
		warningThreshold = 0.9
	}
	return &RPMLimiter{
		store:   store,
		rpm:     rpm,
		burst:   burst,
		warning: warningThreshold,
	}
}

// Allow checks if a request is allowed (implements AccountLimiter interface)
func (l *RPMLimiter) Allow(ctx context.Context, key string, cost int64) (bool, error) {
	// For RPM limiter, cost is treated as number of requests
	if cost <= 0 {
		cost = 1
	}
	limiter := l.getLimiter(key)
	return limiter.AllowN(time.Now(), int(cost)), nil
}

// AllowN checks if N requests are allowed
func (l *RPMLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	limiter := l.getLimiter(key)
	return limiter.AllowN(time.Now(), n), nil
}

// Wait waits until a request is allowed or context is canceled
func (l *RPMLimiter) Wait(ctx context.Context, key string) error {
	limiter := l.getLimiter(key)
	return limiter.Wait(ctx)
}

// WaitN waits until N requests are allowed or context is canceled
func (l *RPMLimiter) WaitN(ctx context.Context, key string, n int) error {
	limiter := l.getLimiter(key)
	return limiter.WaitN(ctx, n)
}

// Reserve reserves a request
func (l *RPMLimiter) Reserve(ctx context.Context, key string) (*rate.Reservation, error) {
	limiter := l.getLimiter(key)
	return limiter.Reserve(), nil
}

// GetUsage returns the current usage information
func (l *RPMLimiter) GetUsage(ctx context.Context, key string) (*Usage, error) {
	redisKey := l.buildKey(key)

	// Get current request count in the minute window
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// Clean old entries and count
	err := l.store.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", windowStart.UnixNano()))
	if err != nil {
		return nil, err
	}

	count, err := l.store.ZCard(ctx, redisKey)
	if err != nil {
		count = 0
	}

	limit := int64(l.rpm)
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	percentUsed := float64(count) / float64(limit) * 100
	resetAt := now.Truncate(time.Minute).Add(time.Minute)

	usage := &Usage{
		Key:         key,
		Used:        count,
		Limit:       limit,
		Remaining:   remaining,
		ResetAt:     resetAt,
		Period:      PeriodMinute,
		PercentUsed: percentUsed,
	}

	if percentUsed >= 100 {
		usage.WarningLevel = "exceeded"
	} else if percentUsed >= l.warning*100 {
		usage.WarningLevel = "warning"
	}

	return usage, nil
}

// RecordRequest records a request for distributed rate limiting
func (l *RPMLimiter) RecordRequest(ctx context.Context, key string) error {
	redisKey := l.buildKey(key)
	now := time.Now().UnixNano()

	// Add entry with timestamp
	err := l.store.ZAdd(ctx, redisKey, float64(now), fmt.Sprintf("%d", now))
	if err != nil {
		return err
	}

	// Set TTL
	return l.store.Expire(ctx, redisKey, 2*time.Minute)
}

// Reset resets the rate limiter for a key
func (l *RPMLimiter) Reset(ctx context.Context, key string) error {
	redisKey := l.buildKey(key)
	if err := l.store.Del(ctx, redisKey); err != nil {
		return err
	}

	// Also remove local limiter
	l.limiters.Delete(key)
	return nil
}

// getLimiter gets or creates a rate limiter for a key
func (l *RPMLimiter) getLimiter(key string) *rate.Limiter {
	if limiter, ok := l.limiters.Load(key); ok {
		if typedLimiter, ok := limiter.(*rate.Limiter); ok {
			return typedLimiter
		}
	}

	// Create new limiter: rate = rpm/60 (requests per second)
	limiter := rate.NewLimiter(rate.Limit(l.rpm)/60, l.burst)
	actual, _ := l.limiters.LoadOrStore(key, limiter)
	if typedLimiter, ok := actual.(*rate.Limiter); ok {
		return typedLimiter
	}
	return limiter
}

// buildKey creates the Redis key for a given identifier
func (l *RPMLimiter) buildKey(key string) string {
	return fmt.Sprintf("rpm_limiter:%s", key)
}

// CleanupLimiters removes old limiters to prevent memory leaks
func (l *RPMLimiter) CleanupLimiters(maxAge time.Duration) {
	// This could be enhanced to track last access time
	// For now, we rely on the sync.Map's internal cleanup
}

// ConcurrentLimiter implements concurrent request limiting
type ConcurrentLimiter struct {
	store     *RedisStore
	maxConcur int
	warning   float64
}

// NewConcurrentLimiter creates a new concurrent request limiter
func NewConcurrentLimiter(store *RedisStore, maxConcurrent int, warningThreshold float64) *ConcurrentLimiter {
	if warningThreshold <= 0 || warningThreshold > 1 {
		warningThreshold = 0.9
	}
	return &ConcurrentLimiter{
		store:     store,
		maxConcur: maxConcurrent,
		warning:   warningThreshold,
	}
}

// Acquire acquires a concurrent slot
func (l *ConcurrentLimiter) Acquire(ctx context.Context, key string, requestID string) (bool, error) {
	redisKey := l.buildKey(key)

	// Get current count
	count, err := l.store.Get(ctx, redisKey)
	if err != nil {
		return false, err
	}

	if count >= int64(l.maxConcur) {
		return false, nil
	}

	// Increment atomically
	newCount, err := l.store.Incr(ctx, redisKey)
	if err != nil {
		return false, err
	}

	// Check if we exceeded after increment (race condition)
	if newCount > int64(l.maxConcur) {
		// Rollback
		if _, err := l.store.Decr(ctx, redisKey); err != nil {
			return false, err
		}
		return false, nil
	}

	// Track this request
	if err := l.store.ZAdd(ctx, redisKey+":requests", float64(time.Now().UnixNano()), requestID); err != nil {
		return false, err
	}

	return true, nil
}

// Release releases a concurrent slot
func (l *ConcurrentLimiter) Release(ctx context.Context, key string, requestID string) error {
	redisKey := l.buildKey(key)

	// Decrement count
	_, err := l.store.Decr(ctx, redisKey)
	if err != nil {
		return err
	}

	// Remove from tracking set
	return l.store.Del(ctx, redisKey+":requests:"+requestID)
}

// GetUsage returns current concurrent usage
func (l *ConcurrentLimiter) GetUsage(ctx context.Context, key string) (*Usage, error) {
	redisKey := l.buildKey(key)

	count, err := l.store.Get(ctx, redisKey)
	if err != nil {
		count = 0
	}

	limit := int64(l.maxConcur)
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	percentUsed := float64(count) / float64(limit) * 100

	usage := &Usage{
		Key:         key,
		Used:        count,
		Limit:       limit,
		Remaining:   remaining,
		ResetAt:     time.Time{}, // Concurrent limits don't reset
		Period:      "",
		PercentUsed: percentUsed,
	}

	if percentUsed >= 100 {
		usage.WarningLevel = "exceeded"
	} else if percentUsed >= l.warning*100 {
		usage.WarningLevel = "warning"
	}

	return usage, nil
}

// buildKey creates the Redis key for a given identifier
func (l *ConcurrentLimiter) buildKey(key string) string {
	return fmt.Sprintf("concurrent_limiter:%s", key)
}
