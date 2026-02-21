package cache

import (
	"context"
	"encoding/json"
	"time"
)

// UsageCacheConfig holds configuration for usage statistics caching
type UsageCacheConfig struct {
	DefaultTTL     time.Duration // Default cache TTL for usage data
	RealtimeTTL    time.Duration // TTL for realtime data (shorter)
	HistoricalTTL  time.Duration // TTL for historical data (longer)
	AggregationTTL time.Duration // TTL for aggregated data
}

// DefaultUsageCacheConfig returns default configuration
func DefaultUsageCacheConfig() UsageCacheConfig {
	return UsageCacheConfig{
		DefaultTTL:     5 * time.Minute,
		RealtimeTTL:    30 * time.Second,
		HistoricalTTL:  1 * time.Hour,
		AggregationTTL: 15 * time.Minute,
	}
}

// DashboardData represents cached dashboard statistics
type DashboardData struct {
	Key          string          `json:"key"`
	Period       string          `json:"period"`       // hour, day, week, month
	GeneratedAt  time.Time       `json:"generated_at"`
	ExpiresAt    time.Time       `json:"expires_at"`

	// Usage metrics
	TotalRequests   int64            `json:"total_requests"`
	TotalTokens     int64            `json:"total_tokens"`
	TotalErrors     int64            `json:"total_errors"`
	AverageLatency  int64            `json:"average_latency_ms"`

	// Provider breakdown
	ProviderStats   map[string]ProviderUsage `json:"provider_stats"`

	// Model breakdown
	ModelStats      map[string]ModelUsage    `json:"model_stats"`

	// User breakdown (top users)
	UserStats       []UserUsage              `json:"user_stats,omitempty"`

	// Cost tracking
	EstimatedCost   float64                  `json:"estimated_cost"`

	// Cache performance
	CacheHits       int64                    `json:"cache_hits"`
	CacheMisses     int64                    `json:"cache_misses"`
	TokensSaved     int64                    `json:"tokens_saved"`
}

// ProviderUsage represents usage stats for a provider
type ProviderUsage struct {
	Provider       string  `json:"provider"`
	Requests       int64   `json:"requests"`
	Tokens         int64   `json:"tokens"`
	Errors         int64   `json:"errors"`
	AvgLatency     int64   `json:"avg_latency_ms"`
	EstimatedCost  float64 `json:"estimated_cost"`
}

// ModelUsage represents usage stats for a model
type ModelUsage struct {
	Model          string  `json:"model"`
	Provider       string  `json:"provider"`
	Requests       int64   `json:"requests"`
	Tokens         int64   `json:"tokens"`
	PromptTokens   int64   `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	EstimatedCost  float64 `json:"estimated_cost"`
}

// UserUsage represents usage stats for a user
type UserUsage struct {
	UserID         string  `json:"user_id"`
	Requests       int64   `json:"requests"`
	Tokens         int64   `json:"tokens"`
	QuotaUsed      int64   `json:"quota_used"`
	QuotaLimit     int64   `json:"quota_limit"`
}

// RealtimeMetrics represents real-time metrics for the dashboard
type RealtimeMetrics struct {
	Key             string    `json:"key"`
	GeneratedAt     time.Time `json:"generated_at"`

	RequestsPerMinute   float64 `json:"requests_per_minute"`
	TokensPerMinute     float64 `json:"tokens_per_minute"`
	ActiveConnections   int64   `json:"active_connections"`
	ErrorRate           float64 `json:"error_rate"`
	AverageLatency      int64   `json:"average_latency_ms"`

	TopModels          []ModelUsage `json:"top_models"`
	RecentErrors       []ErrorEntry `json:"recent_errors,omitempty"`
}

// ErrorEntry represents a recent error entry
type ErrorEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	ErrorType   string    `json:"error_type"`
	Message     string    `json:"message"`
}

// UsageCache handles caching of usage statistics and dashboard data
type UsageCache struct {
	cache  Cache
	stats  *Stats
	config UsageCacheConfig
}

// NewUsageCache creates a new usage cache
func NewUsageCache(cache Cache, config UsageCacheConfig) *UsageCache {
	return &UsageCache{
		cache:  cache,
		stats:  GlobalStatsCollector.GetStats("usage"),
		config: config,
	}
}

// usageKey generates a cache key for usage data
func (c *UsageCache) usageKey(prefix, period string, timestamp time.Time) string {
	// Truncate timestamp based on period for consistent keying
	var truncated time.Time
	switch period {
	case "hour":
		truncated = timestamp.Truncate(time.Hour)
	case "day":
		truncated = timestamp.Truncate(24 * time.Hour)
	case "week":
		truncated = timestamp.AddDate(0, 0, -int(timestamp.Weekday())).Truncate(24 * time.Hour)
	case "month":
		truncated = time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	default:
		truncated = timestamp.Truncate(time.Hour)
	}
	return prefix + ":" + period + ":" + truncated.Format("2006-01-02-15")
}

// GetDashboardData retrieves cached dashboard data
func (c *UsageCache) GetDashboardData(ctx context.Context, period string) (*DashboardData, error) {
	start := time.Now()

	key := c.usageKey("dashboard", period, time.Now())
	var data DashboardData
	err := c.cache.Get(ctx, key, &data)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, err
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)
	return &data, nil
}

// SetDashboardData caches dashboard data
func (c *UsageCache) SetDashboardData(ctx context.Context, period string, data *DashboardData) error {
	key := c.usageKey("dashboard", period, time.Now())

	data.Key = key
	data.GeneratedAt = time.Now()

	var ttl time.Duration
	switch period {
	case "hour":
		ttl = c.config.DefaultTTL
	case "day":
		ttl = c.config.AggregationTTL
	case "week", "month":
		ttl = c.config.HistoricalTTL
	default:
		ttl = c.config.DefaultTTL
	}

	data.ExpiresAt = time.Now().Add(ttl)
	return c.cache.Set(ctx, key, data, ttl)
}

// GetRealtimeMetrics retrieves cached real-time metrics
func (c *UsageCache) GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error) {
	start := time.Now()

	key := "metrics:realtime"
	var metrics RealtimeMetrics
	err := c.cache.Get(ctx, key, &metrics)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, err
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)
	return &metrics, nil
}

// SetRealtimeMetrics caches real-time metrics
func (c *UsageCache) SetRealtimeMetrics(ctx context.Context, metrics *RealtimeMetrics) error {
	key := "metrics:realtime"
	metrics.Key = key
	metrics.GeneratedAt = time.Now()
	return c.cache.Set(ctx, key, metrics, c.config.RealtimeTTL)
}

// GetProviderStats retrieves cached provider statistics
func (c *UsageCache) GetProviderStats(ctx context.Context, provider, period string) (*ProviderUsage, error) {
	start := time.Now()

	key := "provider:" + provider + ":" + period
	var stats ProviderUsage
	err := c.cache.Get(ctx, key, &stats)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, err
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)
	return &stats, nil
}

// SetProviderStats caches provider statistics
func (c *UsageCache) SetProviderStats(ctx context.Context, provider, period string, stats *ProviderUsage) error {
	key := "provider:" + provider + ":" + period
	return c.cache.Set(ctx, key, stats, c.config.AggregationTTL)
}

// GetModelStats retrieves cached model statistics
func (c *UsageCache) GetModelStats(ctx context.Context, model, period string) (*ModelUsage, error) {
	start := time.Now()

	key := "model:" + model + ":" + period
	var stats ModelUsage
	err := c.cache.Get(ctx, key, &stats)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, err
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)
	return &stats, nil
}

// SetModelStats caches model statistics
func (c *UsageCache) SetModelStats(ctx context.Context, model, period string, stats *ModelUsage) error {
	key := "model:" + model + ":" + period
	return c.cache.Set(ctx, key, stats, c.config.AggregationTTL)
}

// GetUserStats retrieves cached user statistics
func (c *UsageCache) GetUserStats(ctx context.Context, userID string) (*UserUsage, error) {
	start := time.Now()

	key := "user:" + userID
	var stats UserUsage
	err := c.cache.Get(ctx, key, &stats)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, err
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)
	return &stats, nil
}

// SetUserStats caches user statistics
func (c *UsageCache) SetUserStats(ctx context.Context, userID string, stats *UserUsage) error {
	key := "user:" + userID
	return c.cache.Set(ctx, key, stats, c.config.DefaultTTL)
}

// InvalidateUser invalidates cached data for a user
func (c *UsageCache) InvalidateUser(ctx context.Context, userID string) error {
	return c.cache.Delete(ctx, "user:"+userID)
}

// InvalidateProvider invalidates cached data for a provider
func (c *UsageCache) InvalidateProvider(ctx context.Context, provider string) error {
	return c.cache.DeleteByPattern(ctx, "provider:"+provider+":*")
}

// InvalidateModel invalidates cached data for a model
func (c *UsageCache) InvalidateModel(ctx context.Context, model string) error {
	return c.cache.DeleteByPattern(ctx, "model:"+model+":*")
}

// GetStats returns cache statistics
func (c *UsageCache) GetStats() StatsSnapshot {
	return c.stats.Snapshot()
}

// GetAggregatedStats retrieves aggregated statistics for a time range
func (c *UsageCache) GetAggregatedStats(ctx context.Context, period string, start, end time.Time) ([]DashboardData, error) {
	// Generate keys for each period in the range
	var results []DashboardData

	current := start
	for current.Before(end) || current.Equal(end) {
		key := c.usageKey("dashboard", period, current)
		var data DashboardData
		if err := c.cache.Get(ctx, key, &data); err == nil {
			results = append(results, data)
		}

		// Move to next period
		switch period {
		case "hour":
			current = current.Add(time.Hour)
		case "day":
			current = current.AddDate(0, 0, 1)
		case "week":
			current = current.AddDate(0, 0, 7)
		case "month":
			current = current.AddDate(0, 1, 0)
		default:
			current = current.Add(time.Hour)
		}
	}

	return results, nil
}

// CacheDashboardJSON caches pre-computed JSON for dashboard API responses
func (c *UsageCache) CacheDashboardJSON(ctx context.Context, key string, jsonData json.RawMessage, ttl time.Duration) error {
	fullKey := "dashboard:json:" + key
	return c.cache.Set(ctx, fullKey, jsonData, ttl)
}

// GetDashboardJSON retrieves cached JSON for dashboard API responses
func (c *UsageCache) GetDashboardJSON(ctx context.Context, key string) (json.RawMessage, error) {
	start := time.Now()

	fullKey := "dashboard:json:" + key
	var data json.RawMessage
	err := c.cache.Get(ctx, fullKey, &data)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, err
		}
		c.stats.RecordError()
		return nil, err
	}

	c.stats.RecordHit(latency)
	return data, nil
}
