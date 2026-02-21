package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotCacheable = errors.New("request is not cacheable")
)

// RequestCacheConfig holds configuration for request caching
type RequestCacheConfig struct {
	DefaultTTL   time.Duration
	MaxBodySize  int64
	SkipStream   bool
	KeyTemplate  string
}

// DefaultRequestCacheConfig returns default configuration
func DefaultRequestCacheConfig() RequestCacheConfig {
	return RequestCacheConfig{
		DefaultTTL:  30 * time.Minute,
		MaxBodySize: 1024 * 1024, // 1MB
		SkipStream:  true,
		KeyTemplate: "{provider}:{model}:{hash}",
	}
}

// CachedRequest represents a cached AI request response
type CachedRequest struct {
	Key         string          `json:"key"`
	Provider    string          `json:"provider"`
	Model       string          `json:"model"`
	Prompt      string          `json:"prompt"`
	Parameters  json.RawMessage `json:"parameters"`
	Response    json.RawMessage `json:"response"`
	TokensUsed  TokenUsage      `json:"tokens_used"`
	CreatedAt   time.Time       `json:"created_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
	HitCount    int64           `json:"hit_count"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

// RequestCache handles caching of AI request responses
type RequestCache struct {
	cache  Cache
	stats  *Stats
	config RequestCacheConfig
}

// NewRequestCache creates a new request cache
func NewRequestCache(cache Cache, config RequestCacheConfig) *RequestCache {
	return &RequestCache{
		cache:  cache,
		stats:  GlobalStatsCollector.GetStats("request"),
		config: config,
	}
}

// CacheKey generates a cache key from request parameters
// Key = sha256(Prompt + Model + Parameters)
func (c *RequestCache) CacheKey(provider, model, prompt string, params interface{}) (string, error) {
	// Build key components
	keyData := map[string]interface{}{
		"provider": provider,
		"model":    model,
		"prompt":   prompt,
		"params":   params,
	}

	data, err := json.Marshal(keyData)
	if err != nil {
		return "", err
	}

	// Generate SHA256 hash (more secure than MD5)
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	return "req:" + provider + ":" + model + ":" + hashStr, nil
}

// Get retrieves a cached response
func (c *RequestCache) Get(ctx context.Context, key string) (*CachedRequest, error) {
	start := time.Now()

	var cached CachedRequest
	err := c.cache.Get(ctx, key, &cached)

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
	c.stats.RecordRequestSaved()
	c.stats.RecordTokensSaved(cached.TokensUsed.TotalTokens)

	// Update hit count
	cached.HitCount++

	return &cached, nil
}

// Set stores a response in cache
func (c *RequestCache) Set(ctx context.Context, req *CachedRequest) error {
	if !c.IsCacheable(req) {
		return ErrNotCacheable
	}

	ttl := c.config.DefaultTTL
	if req.ExpiresAt.After(time.Now()) {
		ttl = time.Until(req.ExpiresAt)
	}

	req.CreatedAt = time.Now()
	req.ExpiresAt = time.Now().Add(ttl)

	return c.cache.Set(ctx, req.Key, req, ttl)
}

// IsCacheable checks if a request can be cached
func (c *RequestCache) IsCacheable(req *CachedRequest) bool {
	// Don't cache empty responses
	if len(req.Response) == 0 {
		return false
	}

	// Don't cache if body is too large
	if c.config.MaxBodySize > 0 && int64(len(req.Response)) > c.config.MaxBodySize {
		return false
	}

	return true
}

// Invalidate invalidates cached requests by pattern
func (c *RequestCache) Invalidate(ctx context.Context, provider, model string) error {
	pattern := "req:" + provider + ":" + model + ":*"

	if rc, ok := c.cache.(*RedisCache); ok {
		return rc.DeleteByPattern(ctx, pattern)
	}

	// For memory cache, we need to handle differently
	// This is a limitation of in-memory cache
	return nil
}

// GetStats returns cache statistics
func (c *RequestCache) GetStats() StatsSnapshot {
	return c.stats.Snapshot()
}

// GetTokenSavings returns total tokens saved
func (c *RequestCache) GetTokenSavings() int64 {
	return c.stats.Snapshot().TokensSaved
}

// RequestCacheMiddleware provides middleware for caching AI requests
type RequestCacheMiddleware struct {
	cache       *RequestCache
	skipFunc    func(provider, model string) bool
	keyFunc     func(provider, model, prompt string, params interface{}) (string, error)
	serialize   func(response interface{}) ([]byte, error)
	deserialize func(data []byte, dest interface{}) error
}

// NewRequestCacheMiddleware creates a new cache middleware
func NewRequestCacheMiddleware(cache *RequestCache) *RequestCacheMiddleware {
	return &RequestCacheMiddleware{
		cache: cache,
		skipFunc: func(provider, model string) bool {
			return false // Don't skip any by default
		},
		keyFunc: cache.CacheKey,
		serialize: func(response interface{}) ([]byte, error) {
			return json.Marshal(response)
		},
		deserialize: func(data []byte, dest interface{}) error {
			return json.Unmarshal(data, dest)
		},
	}
}

// WithSkipFunc sets a custom skip function
func (m *RequestCacheMiddleware) WithSkipFunc(fn func(provider, model string) bool) *RequestCacheMiddleware {
	m.skipFunc = fn
	return m
}

// GetOrCreate attempts to get from cache, or creates and caches the result
func (m *RequestCacheMiddleware) GetOrCreate(
	ctx context.Context,
	provider, model, prompt string,
	params interface{},
	create func() ([]byte, TokenUsage, error),
) ([]byte, bool, error) {
	// Check if we should skip caching for this request
	if m.skipFunc(provider, model) {
		response, _, err := create()
		return response, false, err
	}

	// Generate cache key
	key, err := m.keyFunc(provider, model, prompt, params)
	if err != nil {
		response, _, err := create()
		return response, false, err
	}

	// Try to get from cache
	cached, err := m.cache.Get(ctx, key)
	if err == nil {
		return cached.Response, true, nil
	}

	// Cache miss - create new response
	response, tokens, err := create()
	if err != nil {
		return nil, false, err
	}

	// Store in cache
	cachedReq := &CachedRequest{
		Key:        key,
		Provider:   provider,
		Model:      model,
		Prompt:     prompt,
		Parameters: mustMarshal(params),
		Response:   response,
		TokensUsed: tokens,
	}

	if err := m.cache.Set(ctx, cachedReq); err != nil {
		// Log error but don't fail the request
		// In production, this should be logged
	}

	return response, false, nil
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
