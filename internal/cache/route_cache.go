package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrRouteNotFound = errors.New("route not found")
)

// RouteCacheConfig holds configuration for route caching.
type RouteCacheConfig struct {
	DefaultTTL    time.Duration // Default cache TTL
	MaxEntries    int           // Maximum cached routes
	EnableWarmup  bool          // Enable cache warmup on start
	ConfigVersion string        // Current config version for invalidation
}

// DefaultRouteCacheConfig returns default configuration.
func DefaultRouteCacheConfig() RouteCacheConfig {
	return RouteCacheConfig{
		DefaultTTL:    5 * time.Minute,
		MaxEntries:    10000,
		EnableWarmup:  false,
		ConfigVersion: "1",
	}
}

// RouteDecision represents a cached routing decision.
type RouteDecision struct {
	Key           string          `json:"key"`
	Model         string          `json:"model"`
	Provider      string          `json:"provider"`
	Endpoint      string          `json:"endpoint"`
	Priority      int             `json:"priority"`
	Weight        int             `json:"weight"`
	Parameters    json.RawMessage `json:"parameters,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	ConfigVersion string          `json:"config_version"`
	HitCount      int64           `json:"hit_count"`
}

// RouteCache caches routing decisions for hot models.
type RouteCache struct {
	cache         Cache
	stats         *Stats
	config        RouteCacheConfig
	hotModels     map[string]int64 // model -> access count
	mu            sync.RWMutex
	configVersion string
}

// NewRouteCache creates a new route cache.
func NewRouteCache(cache Cache, config RouteCacheConfig) *RouteCache {
	return &RouteCache{
		cache:         cache,
		stats:         GlobalStatsCollector.GetStats("route"),
		config:        config,
		hotModels:     make(map[string]int64),
		configVersion: config.ConfigVersion,
	}
}

// routeKey generates the cache key for a route decision.
func (c *RouteCache) routeKey(model string, params interface{}) (string, error) {
	// For simple cases, just use model name.
	if params == nil {
		return "route:" + model, nil
	}

	// For complex routing with parameters, include them in key.
	data, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	// Create a simple hash of parameters.
	key := "route:" + model + ":" + string(data)
	if len(key) > 200 {
		// Truncate long keys.
		key = key[:200]
	}
	return key, nil
}

// Get retrieves a cached routing decision.
func (c *RouteCache) Get(ctx context.Context, model string, params interface{}) (*RouteDecision, error) {
	start := time.Now()

	key, err := c.routeKey(model, params)
	if err != nil {
		return nil, err
	}

	var decision RouteDecision
	err = c.cache.Get(ctx, key, &decision)

	latency := time.Since(start)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(latency)
			return nil, ErrRouteNotFound
		}
		c.stats.RecordError()
		return nil, err
	}

	// Check if config version has changed (invalidate stale entries).
	if decision.ConfigVersion != c.configVersion {
		if delErr := c.cache.Delete(ctx, key); delErr != nil {
			c.stats.RecordError()
		}
		c.stats.RecordMiss(latency)
		return nil, ErrRouteNotFound
	}

	c.stats.RecordHit(latency)

	// Track hot model access.
	c.mu.Lock()
	c.hotModels[model]++
	c.mu.Unlock()

	// Update hit count.
	decision.HitCount++

	return &decision, nil
}

// Set stores a routing decision in cache.
func (c *RouteCache) Set(ctx context.Context, model string, params interface{}, decision *RouteDecision) error {
	key, err := c.routeKey(model, params)
	if err != nil {
		return err
	}

	decision.Key = key
	decision.Model = model
	decision.CreatedAt = time.Now()
	decision.ExpiresAt = time.Now().Add(c.config.DefaultTTL)
	decision.ConfigVersion = c.configVersion

	return c.cache.Set(ctx, key, decision, c.config.DefaultTTL)
}

// Invalidate invalidates all cached routes for a model.
func (c *RouteCache) Invalidate(ctx context.Context, model string) error {
	// For Redis, we can use pattern matching.
	if rc, ok := c.cache.(*RedisCache); ok {
		return rc.DeleteByPattern(ctx, "route:"+model+"*")
	}

	// For memory cache, delete exact key.
	key := "route:" + model
	return c.cache.Delete(ctx, key)
}

// InvalidateAll invalidates all cached routes (called on config change).
func (c *RouteCache) InvalidateAll(ctx context.Context) error {
	// Increment config version.
	c.mu.Lock()
	c.configVersion = time.Now().Format("20060102150405")
	c.mu.Unlock()

	// For Redis, delete all route keys.
	if rc, ok := c.cache.(*RedisCache); ok {
		return rc.DeleteByPattern(ctx, "route:*")
	}

	// For memory cache, we can't easily delete all matching keys.
	// The version check in Get() will handle invalidation.
	return nil
}

// GetHotModels returns the most frequently accessed models.
func (c *RouteCache) GetHotModels(limit int) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Convert map to slice for sorting.
	type modelCount struct {
		model string
		count int64
	}

	models := make([]modelCount, 0, len(c.hotModels))
	for m, c := range c.hotModels {
		models = append(models, modelCount{m, c})
	}

	// Sort by count descending using standard library (O(n log n)).
	sort.Slice(models, func(i, j int) bool {
		return models[i].count > models[j].count
	})

	// Return top N.
	result := make([]string, 0, limit)
	for i := 0; i < limit && i < len(models); i++ {
		result = append(result, models[i].model)
	}

	return result
}

// GetStats returns cache statistics.
func (c *RouteCache) GetStats() StatsSnapshot {
	return c.stats.Snapshot()
}

// SetDefaultTTL updates the default TTL for route cache.
func (c *RouteCache) SetDefaultTTL(ttl time.Duration) {
	if ttl > 0 {
		c.config.DefaultTTL = ttl
	}
}

// UpdateConfig updates the cache configuration and invalidates if version changed.
func (c *RouteCache) UpdateConfig(config RouteCacheConfig) error {
	c.mu.Lock()
	shouldInvalidate := config.ConfigVersion != c.configVersion
	if shouldInvalidate {
		c.configVersion = config.ConfigVersion
	}
	c.config = config
	c.mu.Unlock()

	if shouldInvalidate {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return c.invalidateRouteKeys(ctx)
	}
	return nil
}

func (c *RouteCache) invalidateRouteKeys(ctx context.Context) error {
	if rc, ok := c.cache.(*RedisCache); ok {
		return rc.DeleteByPattern(ctx, "route:*")
	}
	return nil
}

// PreloadRoute preloads a routing decision into cache (for warmup).
func (c *RouteCache) PreloadRoute(ctx context.Context, model string, decision *RouteDecision) error {
	return c.Set(ctx, model, nil, decision)
}

// GetRouteStats returns routing statistics for a model.
func (c *RouteCache) GetRouteStats(model string) (hitCount, accessCount int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	accessCount = c.hotModels[model]
	return 0, accessCount
}

// ClearHotModels clears the hot model tracking.
func (c *RouteCache) ClearHotModels() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hotModels = make(map[string]int64)
}
