package limiter

import (
	"context"
	"time"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	Allow(key string) bool
	Wait(ctx context.Context, key string) error
	Reset(key string)
}

// EnhancedLimiter defines the enhanced interface for rate limiting with cost
type EnhancedLimiter interface {
	Allow(ctx context.Context, key string, cost int64) (bool, error)
	GetUsage(ctx context.Context, key string) (*Usage, error)
}

// Store defines the interface for usage storage
type Store interface {
	Get(ctx context.Context, key string) (int64, error)
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// UsageRecord represents a usage record
type UsageRecord struct {
	UserID     string
	Provider   string
	Endpoint   string
	Tokens     int64
	Timestamp  time.Time
}

// LegacyUsageTracker tracks API usage per user/provider (legacy compatibility)
type LegacyUsageTracker struct {
	store Store
}

// NewLegacyUsageTracker creates a new usage tracker
func NewLegacyUsageTracker(store Store) *LegacyUsageTracker {
	return &LegacyUsageTracker{
		store: store,
	}
}

// IncrementUsage increments the usage counter for a user
func (t *LegacyUsageTracker) IncrementUsage(ctx context.Context, userID, provider string, tokens int64) error {
	key := t.buildKey(userID, provider)
	val, err := t.store.Incr(ctx, key)
	if err != nil {
		return err
	}

	// Set expiration if this is a new key (first increment)
	if val == 1 {
		// Expire at the end of the day
		now := time.Now()
		endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		t.store.Expire(ctx, key, time.Until(endOfDay))
	}

	return nil
}

// GetUsage gets the current usage for a user
func (t *LegacyUsageTracker) GetUsage(ctx context.Context, userID, provider string) (int64, error) {
	key := t.buildKey(userID, provider)
	val, err := t.store.Get(ctx, key)
	if err != nil {
		// If key not found, usage is 0
		return 0, nil
	}
	return val, nil
}

func (t *LegacyUsageTracker) buildKey(userID, provider string) string {
	now := time.Now()
	return "usage:" + provider + ":" + userID + ":" + now.Format("2006-01-02")
}
