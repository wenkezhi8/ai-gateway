//nolint:errcheck,revive // Stub methods and setup calls are intentionally lightweight in tests.
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyVectorStore struct{}

func (d *dummyVectorStore) EnsureIndex(ctx context.Context) error  { return nil }
func (d *dummyVectorStore) RebuildIndex(ctx context.Context) error { return nil }
func (d *dummyVectorStore) GetExact(ctx context.Context, cacheKey string) (*VectorCacheDocument, error) {
	return nil, nil
}
func (d *dummyVectorStore) VectorSearch(ctx context.Context, intent string, vector []float64, topK int, minSimilarity float64) ([]VectorSearchHit, error) {
	return nil, nil
}
func (d *dummyVectorStore) Upsert(ctx context.Context, doc *VectorCacheDocument) error { return nil }
func (d *dummyVectorStore) Delete(ctx context.Context, cacheKey string) error          { return nil }
func (d *dummyVectorStore) TouchTTL(ctx context.Context, cacheKey string, ttlSec int64) error {
	return nil
}
func (d *dummyVectorStore) Stats(ctx context.Context) (VectorStoreStats, error) {
	return VectorStoreStats{}, nil
}

func TestDefaultManagerConfig(t *testing.T) {
	cfg := DefaultManagerConfig()

	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, 30*time.Minute, cfg.ResponseTTL)
	assert.False(t, cfg.UseRedis)
}

func TestNewManager_MemoryCache(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.UseRedis = false

	mgr, err := NewManager(cfg)
	require.NoError(t, err)
	require.NotNil(t, mgr)

	assert.NotNil(t, mgr.RequestCache)
	assert.NotNil(t, mgr.ContextCache)
	assert.NotNil(t, mgr.RouteCache)
	assert.NotNil(t, mgr.UsageCache)
	assert.NotNil(t, mgr.ResponseCache)
}

func TestNewManagerWithCache(t *testing.T) {
	cache := NewMemoryCache()
	mgr := NewManagerWithCache(cache)

	require.NotNil(t, mgr)
	assert.NotNil(t, mgr.RequestCache)
	assert.Equal(t, cache, mgr.Cache())
}

func TestManager_GetSet(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()
	key := "test-key"
	value := map[string]string{"hello": "world"}

	err = mgr.Set(ctx, key, value, time.Minute)
	require.NoError(t, err)

	var result map[string]string
	err = mgr.Get(ctx, key, &result)
	require.NoError(t, err)
	assert.Equal(t, "world", result["hello"])
}

func TestManager_Delete(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()
	key := "delete-test"

	mgr.Set(ctx, key, "value", time.Minute)

	err = mgr.Delete(ctx, key)
	require.NoError(t, err)

	var result string
	err = mgr.Get(ctx, key, &result)
	assert.Error(t, err)
}

func TestManager_Exists(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()

	exists, err := mgr.Exists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	mgr.Set(ctx, "exists-test", "value", time.Minute)

	exists, err = mgr.Exists(ctx, "exists-test")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestManager_DeleteByPattern(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()

	mgr.Set(ctx, "prefix:a", "1", time.Minute)
	mgr.Set(ctx, "prefix:b", "2", time.Minute)
	mgr.Set(ctx, "other:c", "3", time.Minute)

	err = mgr.DeleteByPattern(ctx, "prefix:*")
	require.NoError(t, err)
}

func TestManager_GetAllStats(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	stats := mgr.GetAllStats()
	assert.NotNil(t, stats)
}

func TestManager_GetRequestCacheStats(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	stats := mgr.GetRequestCacheStats()
	assert.NotNil(t, stats)
}

func TestManager_GetContextCacheStats(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	stats := mgr.GetContextCacheStats()
	assert.NotNil(t, stats)
}

func TestManager_GetRouteCacheStats(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	stats := mgr.GetRouteCacheStats()
	assert.NotNil(t, stats)
}

func TestManager_GetUsageCacheStats(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	stats := mgr.GetUsageCacheStats()
	assert.NotNil(t, stats)
}

func TestManager_GetTokenSavings(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	savings := mgr.GetTokenSavings()
	assert.GreaterOrEqual(t, savings, int64(0))
}

func TestManager_HealthCheck(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	err = mgr.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestManager_Summary(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	summary := mgr.Summary()
	assert.NotNil(t, summary)
	assert.Greater(t, len(summary), 0)
}

func TestManager_InvalidateAll(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()

	mgr.Set(ctx, "req:test", "value", time.Minute)
	mgr.Set(ctx, "ai-response:test", "value", time.Minute)

	err = mgr.InvalidateAll(ctx)
	require.NoError(t, err)
}

func TestManager_InvalidateProvider(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()

	err = mgr.InvalidateProvider(ctx, "openai")
	require.NoError(t, err)
}

func TestManager_InvalidateModel(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	ctx := context.Background()

	err = mgr.InvalidateModel(ctx, "openai", "gpt-4")
	require.NoError(t, err)
}

func TestManager_SemanticCache(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	assert.Nil(t, mgr.GetSemanticCache())

	sc := &SemanticCache{}
	mgr.SetSemanticCache(sc)

	assert.Equal(t, sc, mgr.GetSemanticCache())
}

func TestManager_VectorStore(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	assert.Nil(t, mgr.GetVectorStore())

	vs := &dummyVectorStore{}
	mgr.SetVectorStore(vs)
	assert.Equal(t, vs, mgr.GetVectorStore())
}

func TestManager_Close(t *testing.T) {
	mgr, err := NewManager(DefaultManagerConfig())
	require.NoError(t, err)

	err = mgr.Close()
	require.NoError(t, err)
}

func TestManager_Cache_ReturnsUnderlying(t *testing.T) {
	cache := NewMemoryCache()
	mgr := NewManagerWithCache(cache)

	assert.Equal(t, cache, mgr.Cache())
}
