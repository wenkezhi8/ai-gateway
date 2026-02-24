package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRouteCacheConfig(t *testing.T) {
	cfg := DefaultRouteCacheConfig()

	assert.Equal(t, 5*time.Minute, cfg.DefaultTTL)
	assert.Equal(t, 10000, cfg.MaxEntries)
	assert.False(t, cfg.EnableWarmup)
	assert.Equal(t, "1", cfg.ConfigVersion)
}

func TestNewRouteCache(t *testing.T) {
	cache := NewMemoryCache()
	cfg := DefaultRouteCacheConfig()

	rc := NewRouteCache(cache, cfg)

	require.NotNil(t, rc)
	assert.NotNil(t, rc.cache)
	assert.NotNil(t, rc.stats)
	assert.NotNil(t, rc.hotModels)
}

func TestRouteCache_Set_Get(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	decision := &RouteDecision{
		Provider: "openai",
		Endpoint: "https://api.openai.com",
		Priority: 1,
		Weight:   100,
	}

	err := rc.Set(ctx, "gpt-4", nil, decision)
	require.NoError(t, err)

	retrieved, err := rc.Get(ctx, "gpt-4", nil)
	require.NoError(t, err)
	assert.Equal(t, "openai", retrieved.Provider)
	assert.Equal(t, "gpt-4", retrieved.Model)
	assert.Equal(t, int64(1), retrieved.HitCount)
}

func TestRouteCache_Get_NotFound(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	_, err := rc.Get(ctx, "nonexistent", nil)

	assert.Error(t, err)
	assert.Equal(t, ErrRouteNotFound, err)
}

func TestRouteCache_Invalidate(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	rc.Set(ctx, "gpt-4", nil, &RouteDecision{Provider: "openai"})

	err := rc.Invalidate(ctx, "gpt-4")
	require.NoError(t, err)

	_, err = rc.Get(ctx, "gpt-4", nil)
	assert.Error(t, err)
}

func TestRouteCache_InvalidateAll(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	rc.Set(ctx, "gpt-4", nil, &RouteDecision{Provider: "openai"})
	rc.Set(ctx, "claude-3", nil, &RouteDecision{Provider: "anthropic"})

	err := rc.InvalidateAll(ctx)
	require.NoError(t, err)
}

func TestRouteCache_GetHotModels(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	rc.Set(ctx, "gpt-4", nil, &RouteDecision{Provider: "openai"})
	rc.Set(ctx, "claude-3", nil, &RouteDecision{Provider: "anthropic"})

	rc.Get(ctx, "gpt-4", nil)
	rc.Get(ctx, "gpt-4", nil)
	rc.Get(ctx, "claude-3", nil)

	hot := rc.GetHotModels(5)
	assert.NotEmpty(t, hot)
	assert.Contains(t, hot, "gpt-4")
}

func TestRouteCache_GetStats(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	stats := rc.GetStats()
	assert.NotNil(t, stats)
}

func TestRouteCache_UpdateConfig(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	newCfg := DefaultRouteCacheConfig()
	newCfg.DefaultTTL = 10 * time.Minute
	newCfg.ConfigVersion = "2"

	err := rc.UpdateConfig(newCfg)
	require.NoError(t, err)

	assert.Equal(t, "2", rc.configVersion)
}

func TestRouteCache_PreloadRoute(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	decision := &RouteDecision{Provider: "openai"}

	err := rc.PreloadRoute(ctx, "gpt-4", decision)
	require.NoError(t, err)

	retrieved, err := rc.Get(ctx, "gpt-4", nil)
	require.NoError(t, err)
	assert.Equal(t, "openai", retrieved.Provider)
}

func TestRouteCache_GetRouteStats(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	rc.Set(ctx, "gpt-4", nil, &RouteDecision{Provider: "openai"})
	rc.Get(ctx, "gpt-4", nil)
	rc.Get(ctx, "gpt-4", nil)

	_, accessCount := rc.GetRouteStats("gpt-4")
	assert.Equal(t, int64(2), accessCount)
}

func TestRouteCache_ClearHotModels(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	rc.Set(ctx, "gpt-4", nil, &RouteDecision{Provider: "openai"})
	rc.Get(ctx, "gpt-4", nil)

	rc.ClearHotModels()

	_, accessCount := rc.GetRouteStats("gpt-4")
	assert.Equal(t, int64(0), accessCount)
}

func TestRouteCache_RouteKey(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	key, err := rc.routeKey("gpt-4", nil)
	require.NoError(t, err)
	assert.Equal(t, "route:gpt-4", key)

	key, err = rc.routeKey("gpt-4", map[string]string{"user": "test"})
	require.NoError(t, err)
	assert.Contains(t, key, "gpt-4")
}

func TestRouteCache_ConfigVersionInvalidation(t *testing.T) {
	cfg := DefaultRouteCacheConfig()
	cfg.ConfigVersion = "1"
	rc := NewRouteCache(NewMemoryCache(), cfg)

	ctx := context.Background()
	rc.Set(ctx, "gpt-4", nil, &RouteDecision{Provider: "openai"})

	rc.mu.Lock()
	rc.configVersion = "2"
	rc.mu.Unlock()

	_, err := rc.Get(ctx, "gpt-4", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrRouteNotFound, err)
}

func TestRouteCache_ParamsKey(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	params := map[string]interface{}{
		"user_id": "user-1",
		"model":   "gpt-4",
	}

	key, err := rc.routeKey("gpt-4", params)
	require.NoError(t, err)
	assert.Contains(t, key, "gpt-4")
}

func TestRouteCache_SetWithParams(t *testing.T) {
	rc := NewRouteCache(NewMemoryCache(), DefaultRouteCacheConfig())

	ctx := context.Background()
	params := map[string]string{"user": "test"}
	decision := &RouteDecision{Provider: "openai"}

	err := rc.Set(ctx, "gpt-4", params, decision)
	require.NoError(t, err)

	retrieved, err := rc.Get(ctx, "gpt-4", params)
	require.NoError(t, err)
	assert.Equal(t, "openai", retrieved.Provider)
}
