package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Cache defines the interface for caching
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	Keys(pattern string) []string
}

// ResponseCache provides caching for AI responses
type ResponseCache struct {
	cache  Cache
	ttl    time.Duration
	prefix string
	stats  *Stats
}

// NewResponseCache creates a new response cache
func NewResponseCache(cache Cache, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		cache:  cache,
		ttl:    ttl,
		prefix: "ai-response:",
		stats:  GlobalStatsCollector.GetStats("response"),
	}
}

// CachedResponse represents a cached AI response
type CachedResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
	CreatedAt  time.Time         `json:"created_at"`
	HitCount   int64             `json:"hit_count"`
	HitModels  map[string]int64  `json:"hit_models,omitempty"`
	Provider   string            `json:"provider"`
	Model      string            `json:"model"`
	Prompt     string            `json:"prompt"`    // 便于缓存管理页面展示用户消息
	TaskType   string            `json:"task_type"` // 便于按任务类型过滤
}

// GenerateKey creates a cache key from request parameters
func (c *ResponseCache) GenerateKey(provider, model string, request interface{}) (string, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	return c.prefix + provider + ":" + model + ":" + hashStr, nil
}

// Get retrieves a cached response
func (c *ResponseCache) Get(ctx context.Context, key string) (*CachedResponse, error) {
	start := time.Now()
	var cached CachedResponse
	err := c.cache.Get(ctx, key, &cached)
	if err != nil {
		if err == ErrNotFound {
			c.stats.RecordMiss(time.Since(start))
		} else {
			c.stats.RecordError()
		}
		return nil, err
	}
	c.stats.RecordHit(time.Since(start))
	return &cached, nil
}

// Set stores a response in cache
func (c *ResponseCache) Set(ctx context.Context, key string, response *CachedResponse) error {
	return c.cache.Set(ctx, key, response, c.ttl)
}

// SetWithTTL stores a response in cache with a custom TTL
func (c *ResponseCache) SetWithTTL(ctx context.Context, key string, response *CachedResponse, ttl time.Duration) error {
	return c.cache.Set(ctx, key, response, ttl)
}

// SetDefaultTTL updates the default TTL for response cache
func (c *ResponseCache) SetDefaultTTL(ttl time.Duration) {
	if ttl > 0 {
		c.ttl = ttl
	}
}

// GetStats returns response cache statistics
func (c *ResponseCache) GetStats() StatsSnapshot {
	return c.stats.Snapshot()
}

// Delete removes a cached response
func (c *ResponseCache) Delete(ctx context.Context, key string) error {
	return c.cache.Delete(ctx, key)
}

// IsCacheable checks if a request is cacheable
func (c *ResponseCache) IsCacheable(request interface{}) bool {
	// Non-streaming requests with deterministic parameters are cacheable
	type cacheable interface {
		IsStream() bool
	}

	if req, ok := request.(cacheable); ok {
		return !req.IsStream()
	}

	return true
}
