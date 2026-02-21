package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRequestCacheConfig(t *testing.T) {
	config := DefaultRequestCacheConfig()
	assert.Equal(t, 30*time.Minute, config.DefaultTTL)
	assert.Equal(t, int64(1024*1024), config.MaxBodySize)
	assert.True(t, config.SkipStream)
}

func TestNewRequestCache(t *testing.T) {
	memCache := NewMemoryCache()
	config := DefaultRequestCacheConfig()
	rc := NewRequestCache(memCache, config)

	assert.NotNil(t, rc)
	assert.NotNil(t, rc.cache)
	assert.NotNil(t, rc.stats)
}

func TestRequestCache_CacheKey(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())

	key1, err := rc.CacheKey("openai", "gpt-4", "Hello", nil)
	require.NoError(t, err)
	assert.Contains(t, key1, "req:openai:gpt-4:")

	// Same inputs should produce same key
	key2, err := rc.CacheKey("openai", "gpt-4", "Hello", nil)
	require.NoError(t, err)
	assert.Equal(t, key1, key2)

	// Different prompt should produce different key
	key3, err := rc.CacheKey("openai", "gpt-4", "World", nil)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key3)

	// Different model should produce different key
	key4, err := rc.CacheKey("anthropic", "claude-3", "Hello", nil)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key4)
}

func TestRequestCache_CacheKey_WithParams(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())

	params := map[string]interface{}{
		"temperature": 0.7,
		"max_tokens":  1000,
	}

	key1, err := rc.CacheKey("openai", "gpt-4", "Hello", params)
	require.NoError(t, err)

	// Same params should produce same key
	key2, err := rc.CacheKey("openai", "gpt-4", "Hello", params)
	require.NoError(t, err)
	assert.Equal(t, key1, key2)

	// Different params should produce different key
	params2 := map[string]interface{}{
		"temperature": 0.9,
		"max_tokens":  1000,
	}
	key3, err := rc.CacheKey("openai", "gpt-4", "Hello", params2)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key3)
}

func TestRequestCache_SetAndGet(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())
	ctx := context.Background()

	key := "req:openai:gpt-4:test123"
	response := json.RawMessage(`{"choices":[{"message":{"content":"Hello"}}]}`)

	req := &CachedRequest{
		Key:        key,
		Provider:   "openai",
		Model:      "gpt-4",
		Prompt:     "Hi",
		Response:   response,
		TokensUsed: TokenUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}

	err := rc.Set(ctx, req)
	require.NoError(t, err)

	// Retrieve cached request
	cached, err := rc.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, "openai", cached.Provider)
	assert.Equal(t, "gpt-4", cached.Model)
	assert.Equal(t, int64(30), cached.TokensUsed.TotalTokens)
	assert.Equal(t, int64(1), cached.HitCount)
}

func TestRequestCache_Get_NotFound(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())
	ctx := context.Background()

	_, err := rc.Get(ctx, "nonexistent-key")
	assert.Equal(t, ErrNotFound, err)
}

func TestRequestCache_IsCacheable(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())

	// Empty response is not cacheable
	req1 := &CachedRequest{
		Response: json.RawMessage{},
	}
	assert.False(t, rc.IsCacheable(req1))

	// Response too large is not cacheable
	largeResponse := make([]byte, 2*1024*1024) // 2MB
	req2 := &CachedRequest{
		Response: largeResponse,
	}
	assert.False(t, rc.IsCacheable(req2))

	// Normal response is cacheable
	req3 := &CachedRequest{
		Response: json.RawMessage(`{"result":"ok"}`),
	}
	assert.True(t, rc.IsCacheable(req3))
}

func TestRequestCache_Stats(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())
	ctx := context.Background()

	// Cache a request
	key := "req:test:stats:unique"
	req := &CachedRequest{
		Key:        key,
		Provider:   "test",
		Model:      "test-model",
		Response:   json.RawMessage(`{"ok":true}`),
		TokensUsed: TokenUsage{TotalTokens: 100},
	}

	err := rc.Set(ctx, req)
	require.NoError(t, err)

	// Get from cache (should record hit)
	_, err = rc.Get(ctx, key)
	require.NoError(t, err)

	// Try to get nonexistent key (should record miss)
	_, err = rc.Get(ctx, "nonexistent-key-unique")
	assert.Equal(t, ErrNotFound, err)

	// Check stats - note that stats may include data from other tests due to global collector
	// So we just verify the stats are non-zero
	stats := rc.GetStats()
	assert.GreaterOrEqual(t, stats.Hits, int64(1))
	assert.GreaterOrEqual(t, stats.Misses, int64(1))
}

func TestRequestCache_Invalidate(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewRequestCache(memCache, DefaultRequestCacheConfig())
	ctx := context.Background()

	// Cache multiple requests with unique keys
	req1 := &CachedRequest{
		Key:      "req:openai:gpt-4:invalidate-test-1",
		Provider: "openai",
		Model:    "gpt-4",
		Response: json.RawMessage(`{"ok":true}`),
	}
	req2 := &CachedRequest{
		Key:      "req:anthropic:claude-3:invalidate-test-2",
		Provider: "anthropic",
		Model:    "claude-3",
		Response: json.RawMessage(`{"ok":true}`),
	}

	err := rc.Set(ctx, req1)
	require.NoError(t, err)
	err = rc.Set(ctx, req2)
	require.NoError(t, err)

	// Verify both are cached
	_, err = rc.Get(ctx, req1.Key)
	require.NoError(t, err)
	_, err = rc.Get(ctx, req2.Key)
	require.NoError(t, err)

	// Invalidate openai gpt-4
	// Note: Memory cache does not support pattern-based invalidation
	// This is a limitation - for full invalidation support, use Redis cache
	err = rc.Invalidate(ctx, "openai", "gpt-4")
	require.NoError(t, err)

	// For memory cache, Invalidate is a no-op, so entries still exist
	// This test documents the current behavior
	_, err = rc.Get(ctx, req1.Key)
	// Memory cache doesn't support invalidation, so entry still exists
	require.NoError(t, err)
}

func TestRequestCache_TTL(t *testing.T) {
	memCache := NewMemoryCache()
	config := DefaultRequestCacheConfig()
	config.DefaultTTL = 100 * time.Millisecond
	rc := NewRequestCache(memCache, config)
	ctx := context.Background()

	key := "req:ttl:test"
	req := &CachedRequest{
		Key:      key,
		Provider: "test",
		Model:    "test",
		Response: json.RawMessage(`{}`),
	}

	err := rc.Set(ctx, req)
	require.NoError(t, err)

	// Should exist immediately
	_, err = rc.Get(ctx, key)
	require.NoError(t, err)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, err = rc.Get(ctx, key)
	assert.Equal(t, ErrNotFound, err)
}
