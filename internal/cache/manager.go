package cache

import (
	"context"
	"encoding/json"
	"time"
)

// Manager provides a unified interface for all cache operations
type Manager struct {
	cache Cache

	// Specialized caches
	RequestCache  *RequestCache
	ContextCache  *ContextCache
	RouteCache    *RouteCache
	ResponseCache *ResponseCache
	UsageCache    *UsageCache

	// Statistics
	stats *StatsCollector
}

// ManagerConfig holds configuration for the cache manager
type ManagerConfig struct {
	Redis           RedisConfig
	ContextConfig   ContextCacheConfig
	RequestConfig   RequestCacheConfig
	RouteConfig     RouteCacheConfig
	UsageConfig     UsageCacheConfig
	ResponseTTL     time.Duration
	UseRedis        bool
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		ContextConfig:  DefaultContextCacheConfig(),
		RequestConfig:  DefaultRequestCacheConfig(),
		RouteConfig:    DefaultRouteCacheConfig(),
		UsageConfig:    DefaultUsageCacheConfig(),
		ResponseTTL:    30 * time.Minute,
		UseRedis:       false,
	}
}

// NewManager creates a new cache manager
func NewManager(cfg ManagerConfig) (*Manager, error) {
	var cache Cache
	var err error

	if cfg.UseRedis {
		cache, err = NewRedisCache(cfg.Redis)
		if err != nil {
			// Fall back to memory cache if Redis is unavailable
			cache = NewMemoryCache()
		}
	} else {
		cache = NewMemoryCache()
	}

	return &Manager{
		cache:         cache,
		RequestCache:  NewRequestCache(cache, cfg.RequestConfig),
		ContextCache:  NewContextCache(cache, cfg.ContextConfig),
		RouteCache:    NewRouteCache(cache, cfg.RouteConfig),
		UsageCache:    NewUsageCache(cache, cfg.UsageConfig),
		ResponseCache: NewResponseCache(cache, cfg.ResponseTTL),
		stats:         GlobalStatsCollector,
	}, nil
}

// NewManagerWithCache creates a manager with an existing cache
func NewManagerWithCache(cache Cache) *Manager {
	return &Manager{
		cache:         cache,
		RequestCache:  NewRequestCache(cache, DefaultRequestCacheConfig()),
		ContextCache:  NewContextCache(cache, DefaultContextCacheConfig()),
		RouteCache:    NewRouteCache(cache, DefaultRouteCacheConfig()),
		UsageCache:    NewUsageCache(cache, DefaultUsageCacheConfig()),
		ResponseCache: NewResponseCache(cache, 30*time.Minute),
		stats:         GlobalStatsCollector,
	}
}

// Get retrieves a value from the underlying cache
func (m *Manager) Get(ctx context.Context, key string, dest interface{}) error {
	return m.cache.Get(ctx, key, dest)
}

// Cache returns the underlying cache interface
func (m *Manager) Cache() Cache {
	return m.cache
}

// Set stores a value in the underlying cache
func (m *Manager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return m.cache.Set(ctx, key, value, ttl)
}

// Delete removes a value from the underlying cache
func (m *Manager) Delete(ctx context.Context, key string) error {
	return m.cache.Delete(ctx, key)
}

// DeleteByPattern removes all values matching a pattern
func (m *Manager) DeleteByPattern(ctx context.Context, pattern string) error {
	return m.cache.DeleteByPattern(ctx, pattern)
}

// Exists checks if a key exists
func (m *Manager) Exists(ctx context.Context, key string) (bool, error) {
	return m.cache.Exists(ctx, key)
}

// GetAllStats returns statistics for all caches
func (m *Manager) GetAllStats() map[string]StatsSnapshot {
	return m.stats.AllStats()
}

// GetRequestCacheStats returns request cache statistics
func (m *Manager) GetRequestCacheStats() StatsSnapshot {
	return m.RequestCache.GetStats()
}

// GetContextCacheStats returns context cache statistics
func (m *Manager) GetContextCacheStats() StatsSnapshot {
	return m.ContextCache.GetStats()
}

// GetRouteCacheStats returns route cache statistics
func (m *Manager) GetRouteCacheStats() StatsSnapshot {
	return m.RouteCache.GetStats()
}

// GetUsageCacheStats returns usage cache statistics
func (m *Manager) GetUsageCacheStats() StatsSnapshot {
	return m.UsageCache.GetStats()
}

// GetTokenSavings returns total tokens saved across all caches
func (m *Manager) GetTokenSavings() int64 {
	return m.RequestCache.GetTokenSavings() +
		m.ContextCache.GetTokenSavings()
}

// InvalidateAll invalidates all cached data
func (m *Manager) InvalidateAll(ctx context.Context) error {
	// Invalidate route cache
	if err := m.RouteCache.InvalidateAll(ctx); err != nil {
		return err
	}

	// Invalidate request cache patterns
	if err := m.cache.DeleteByPattern(ctx, "req:*"); err != nil {
		return err
	}

	// Invalidate response cache patterns
	if err := m.cache.DeleteByPattern(ctx, "ai-response:*"); err != nil {
		return err
	}

	return nil
}

// InvalidateProvider invalidates all cached data for a specific provider
func (m *Manager) InvalidateProvider(ctx context.Context, provider string) error {
	// Invalidate request cache for this provider
	if err := m.RequestCache.Invalidate(ctx, provider, "*"); err != nil {
		return err
	}

	// Invalidate route cache for this provider's models
	if err := m.cache.DeleteByPattern(ctx, "route:*"); err != nil {
		return err
	}

	// Invalidate usage cache for this provider
	if err := m.UsageCache.InvalidateProvider(ctx, provider); err != nil {
		return err
	}

	return nil
}

// InvalidateModel invalidates all cached data for a specific model
func (m *Manager) InvalidateModel(ctx context.Context, provider, model string) error {
	// Invalidate request cache
	if err := m.RequestCache.Invalidate(ctx, provider, model); err != nil {
		return err
	}

	// Invalidate route cache
	if err := m.RouteCache.Invalidate(ctx, model); err != nil {
		return err
	}

	// Invalidate usage cache for this model
	if err := m.UsageCache.InvalidateModel(ctx, model); err != nil {
		return err
	}

	return nil
}

// HealthCheck verifies cache connectivity
func (m *Manager) HealthCheck(ctx context.Context) error {
	// Try a simple set/get operation
	testKey := "health:check"
	testValue := map[string]interface{}{"timestamp": time.Now().Unix()}

	if err := m.cache.Set(ctx, testKey, testValue, time.Minute); err != nil {
		return err
	}

	var result map[string]interface{}
	if err := m.cache.Get(ctx, testKey, &result); err != nil {
		return err
	}

	// Cleanup
	m.cache.Delete(ctx, testKey)

	return nil
}

// Summary returns a summary of cache state
func (m *Manager) Summary() json.RawMessage {
	summary := map[string]interface{}{
		"stats":         m.GetAllStats(),
		"token_savings": m.GetTokenSavings(),
		"hot_models":    m.RouteCache.GetHotModels(10),
	}

	data, _ := json.Marshal(summary)
	return data
}

// Close closes any open connections
func (m *Manager) Close() error {
	if rc, ok := m.cache.(*RedisCache); ok {
		return rc.Close()
	}
	return nil
}
