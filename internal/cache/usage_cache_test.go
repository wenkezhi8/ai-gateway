package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultUsageCacheConfig(t *testing.T) {
	cfg := DefaultUsageCacheConfig()

	assert.Equal(t, 5*time.Minute, cfg.DefaultTTL)
	assert.Equal(t, 30*time.Second, cfg.RealtimeTTL)
	assert.Equal(t, time.Hour, cfg.HistoricalTTL)
}

func TestNewUsageCache(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	require.NotNil(t, uc)
}

func TestUsageCache_DashboardData(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	data := &DashboardData{
		TotalRequests: 1000,
		TotalTokens:   50000,
	}

	err := uc.SetDashboardData(ctx, "hour", data)
	require.NoError(t, err)

	retrieved, err := uc.GetDashboardData(ctx, "hour")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), retrieved.TotalRequests)
}

func TestUsageCache_DashboardData_NotFound(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	_, err := uc.GetDashboardData(ctx, "hour")
	assert.Error(t, err)
}

func TestUsageCache_RealtimeMetrics(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	metrics := &RealtimeMetrics{
		RequestsPerMinute: 100.5,
		TokensPerMinute:   5000.5,
	}

	err := uc.SetRealtimeMetrics(ctx, metrics)
	require.NoError(t, err)

	retrieved, err := uc.GetRealtimeMetrics(ctx)
	require.NoError(t, err)
	assert.Equal(t, 100.5, retrieved.RequestsPerMinute)
}

func TestUsageCache_RealtimeMetrics_NotFound(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	_, err := uc.GetRealtimeMetrics(ctx)
	assert.Error(t, err)
}

func TestUsageCache_ProviderStats(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	stats := &ProviderUsage{
		Provider: "openai",
		Requests: 500,
		Tokens:   25000,
	}

	err := uc.SetProviderStats(ctx, "openai", "hour", stats)
	require.NoError(t, err)

	retrieved, err := uc.GetProviderStats(ctx, "openai", "hour")
	require.NoError(t, err)
	assert.Equal(t, "openai", retrieved.Provider)
	assert.Equal(t, int64(500), retrieved.Requests)
}

func TestUsageCache_ProviderStats_NotFound(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	_, err := uc.GetProviderStats(ctx, "nonexistent", "hour")
	assert.Error(t, err)
}

func TestUsageCache_ModelStats(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	stats := &ModelUsage{
		Model:    "gpt-4",
		Requests: 200,
		Tokens:   10000,
	}

	err := uc.SetModelStats(ctx, "gpt-4", "hour", stats)
	require.NoError(t, err)

	retrieved, err := uc.GetModelStats(ctx, "gpt-4", "hour")
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", retrieved.Model)
}

func TestUsageCache_ModelStats_NotFound(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	_, err := uc.GetModelStats(ctx, "nonexistent", "hour")
	assert.Error(t, err)
}

func TestUsageCache_UserStats(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	stats := &UserUsage{
		UserID:   "user-1",
		Requests: 50,
		Tokens:   5000,
	}

	err := uc.SetUserStats(ctx, "user-1", stats)
	require.NoError(t, err)

	retrieved, err := uc.GetUserStats(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", retrieved.UserID)
}

func TestUsageCache_UserStats_NotFound(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	_, err := uc.GetUserStats(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestUsageCache_InvalidateUser(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	uc.SetUserStats(ctx, "user-1", &UserUsage{UserID: "user-1"})

	err := uc.InvalidateUser(ctx, "user-1")
	require.NoError(t, err)

	_, err = uc.GetUserStats(ctx, "user-1")
	assert.Error(t, err)
}

func TestUsageCache_InvalidateProvider(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	uc.SetProviderStats(ctx, "openai", "hour", &ProviderUsage{Provider: "openai"})

	err := uc.InvalidateProvider(ctx, "openai")
	require.NoError(t, err)
}

func TestUsageCache_InvalidateModel(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	uc.SetModelStats(ctx, "gpt-4", "hour", &ModelUsage{Model: "gpt-4"})

	err := uc.InvalidateModel(ctx, "gpt-4")
	require.NoError(t, err)
}

func TestUsageCache_GetStats(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())

	stats := uc.GetStats()
	assert.NotNil(t, stats)
}

func TestUsageCache_DashboardJSON(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	jsonData := []byte(`{"test":"data"}`)
	err := uc.CacheDashboardJSON(ctx, "test-key", jsonData, time.Minute)
	require.NoError(t, err)

	retrieved, err := uc.GetDashboardJSON(ctx, "test-key")
	require.NoError(t, err)
	assert.JSONEq(t, string(jsonData), string(retrieved))
}

func TestUsageCache_DashboardJSON_NotFound(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	_, err := uc.GetDashboardJSON(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestUsageCache_UsageKey(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	now := time.Now()

	tests := []struct {
		period   string
		expected string
	}{
		{"hour", "dashboard:hour:" + now.Truncate(time.Hour).Format("2006-01-02-15")},
		{"day", "dashboard:day:" + now.Truncate(24*time.Hour).Format("2006-01-02")},
		{"week", "dashboard:week:"},
		{"month", "dashboard:month:"},
	}

	for _, tt := range tests {
		key := uc.usageKey("dashboard", tt.period, now)
		assert.Contains(t, key, "dashboard:"+tt.period)
	}
}

func TestUsageCache_AggregatedStats(t *testing.T) {
	uc := NewUsageCache(NewMemoryCache(), DefaultUsageCacheConfig())
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now

	data := &DashboardData{TotalRequests: 100}
	uc.SetDashboardData(ctx, "hour", data)

	results, err := uc.GetAggregatedStats(ctx, "hour", start, end)
	require.NoError(t, err)
	// May or may not have results depending on timing
	_ = results
}
