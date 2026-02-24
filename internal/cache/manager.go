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

	// Semantic cache (optional)
	semanticCache *SemanticCache

	// Statistics
	stats *StatsCollector
}

// ManagerConfig holds configuration for the cache manager
type ManagerConfig struct {
	Redis         RedisConfig
	ContextConfig ContextCacheConfig
	RequestConfig RequestCacheConfig
	RouteConfig   RouteCacheConfig
	UsageConfig   UsageCacheConfig
	ResponseTTL   time.Duration
	UseRedis      bool
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		ContextConfig: DefaultContextCacheConfig(),
		RequestConfig: DefaultRequestCacheConfig(),
		RouteConfig:   DefaultRouteCacheConfig(),
		UsageConfig:   DefaultUsageCacheConfig(),
		ResponseTTL:   30 * time.Minute,
		UseRedis:      false,
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

// GetSemanticCache returns the semantic cache
func (m *Manager) GetSemanticCache() *SemanticCache {
	return m.semanticCache
}

// SetSemanticCache sets the semantic cache
func (m *Manager) SetSemanticCache(sc *SemanticCache) {
	m.semanticCache = sc
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

// CacheEntryInfo represents info about a cache entry
type CacheEntryInfo struct {
	Key       string     `json:"key"`
	Type      string     `json:"type"`
	Size      int        `json:"size"`
	Hits      int        `json:"hits"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	TTL       int        `json:"ttl"`
	Preview   string     `json:"preview"`
	Model     string     `json:"model,omitempty"`
	Provider  string     `json:"provider,omitempty"`
}

// CacheEntryDetail represents detailed cache entry data
type CacheEntryDetail struct {
	Key       string      `json:"key"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	Size      int         `json:"size"`
	Hits      int         `json:"hits"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt *time.Time  `json:"expires_at,omitempty"`
	TTL       int         `json:"ttl"`
}

// ListEntries returns a list of cache entries
func (m *Manager) ListEntries(cacheType string, search string) []*CacheEntryInfo {
	entries := make([]*CacheEntryInfo, 0)

	keys := m.cache.Keys(getKeyPattern(cacheType))
	for _, key := range keys {
		if search != "" && !containsIgnoreCase(key, search) {
			continue
		}

		entry := &CacheEntryInfo{
			Key:  key,
			Type: getCacheTypeFromKey(key),
		}

		if mc, ok := m.cache.(*MemoryCache); ok {
			if meta := mc.GetMeta(key); meta != nil {
				entry.Size = meta.Size
				entry.Hits = meta.Hits
				entry.CreatedAt = meta.CreatedAt
				entry.TTL = meta.TTL
				if meta.TTL > 0 {
					exp := meta.CreatedAt.Add(time.Duration(meta.TTL) * time.Second)
					entry.ExpiresAt = &exp
				}
				entry.Preview = meta.Preview
				entry.Model = meta.Model
				entry.Provider = meta.Provider
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// GetEntryDetail returns detailed information about a cache entry
func (m *Manager) GetEntryDetail(ctx context.Context, key string) (*CacheEntryDetail, error) {
	var value interface{}
	if err := m.cache.Get(ctx, key, &value); err != nil {
		return nil, err
	}

	detail := &CacheEntryDetail{
		Key:   key,
		Type:  getCacheTypeFromKey(key),
		Value: value,
	}

	if mc, ok := m.cache.(*MemoryCache); ok {
		if meta := mc.GetMeta(key); meta != nil {
			detail.Size = meta.Size
			detail.Hits = meta.Hits
			detail.CreatedAt = meta.CreatedAt
			detail.TTL = meta.TTL
			if meta.TTL > 0 {
				exp := meta.CreatedAt.Add(time.Duration(meta.TTL) * time.Second)
				detail.ExpiresAt = &exp
			}
		}
	}

	return detail, nil
}

func getKeyPattern(cacheType string) string {
	switch cacheType {
	case "request":
		return "req:*"
	case "context":
		return "ctx:*"
	case "route":
		return "route:*"
	case "usage":
		return "usage:*"
	case "response":
		return "ai-response:*"
	default:
		return "*"
	}
}

func getCacheTypeFromKey(key string) string {
	if len(key) >= 4 {
		prefix := key[:4]
		switch {
		case prefix == "req:":
			return "request"
		case prefix == "ctx:":
			return "context"
		case len(key) >= 6 && key[:6] == "route:":
			return "route"
		case len(key) >= 6 && key[:6] == "usage:":
			return "usage"
		case len(key) >= 12 && key[:12] == "ai-response:":
			return "response"
		}
	}
	return "other"
}

func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sLower[i] = c
	}
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		substrLower[i] = c
	}

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
