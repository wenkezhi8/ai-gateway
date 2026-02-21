package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStats(t *testing.T) {
	stats := NewStats()
	assert.NotNil(t, stats)
}

func TestStats_RecordHit(t *testing.T) {
	stats := NewStats()
	latency := time.Millisecond * 10

	stats.RecordHit(latency)

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(1), snapshot.Hits)
	assert.Equal(t, int64(0), snapshot.Misses)
	assert.GreaterOrEqual(t, snapshot.AvgLatencyNs, int64(0))
}

func TestStats_RecordMiss(t *testing.T) {
	stats := NewStats()
	latency := time.Millisecond * 5

	stats.RecordMiss(latency)

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(0), snapshot.Hits)
	assert.Equal(t, int64(1), snapshot.Misses)
}

func TestStats_RecordTokensSaved(t *testing.T) {
	stats := NewStats()

	stats.RecordTokensSaved(100)
	stats.RecordTokensSaved(50)

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(150), snapshot.TokensSaved)
}

func TestStats_RecordRequestSaved(t *testing.T) {
	stats := NewStats()

	stats.RecordRequestSaved()
	stats.RecordRequestSaved()
	stats.RecordRequestSaved()

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(3), snapshot.RequestsSaved)
}

func TestStats_RecordEviction(t *testing.T) {
	stats := NewStats()

	stats.RecordEviction()

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(1), snapshot.Evictions)
}

func TestStats_RecordError(t *testing.T) {
	stats := NewStats()

	stats.RecordError()
	stats.RecordError()

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(2), snapshot.Errors)
}

func TestStats_HitRate(t *testing.T) {
	tests := []struct {
		name      string
		hits      int
		misses    int
		expectedRate float64
	}{
		{
			name:      "100% hit rate",
			hits:      10,
			misses:    0,
			expectedRate: 1.0,
		},
		{
			name:      "50% hit rate",
			hits:      5,
			misses:    5,
			expectedRate: 0.5,
		},
		{
			name:      "0% hit rate",
			hits:      0,
			misses:    10,
			expectedRate: 0.0,
		},
		{
			name:      "no operations",
			hits:      0,
			misses:    0,
			expectedRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewStats()

			for i := 0; i < tt.hits; i++ {
				stats.RecordHit(time.Millisecond)
			}
			for i := 0; i < tt.misses; i++ {
				stats.RecordMiss(time.Millisecond)
			}

			snapshot := stats.Snapshot()
			assert.Equal(t, tt.expectedRate, snapshot.HitRate)
		})
	}
}

func TestStats_Snapshot(t *testing.T) {
	stats := NewStats()

	stats.RecordHit(time.Millisecond * 10)
	stats.RecordMiss(time.Millisecond * 5)
	stats.RecordTokensSaved(100)
	stats.RecordRequestSaved()
	stats.RecordEviction()
	stats.RecordError()

	snapshot := stats.Snapshot()

	assert.Equal(t, int64(1), snapshot.Hits)
	assert.Equal(t, int64(1), snapshot.Misses)
	assert.Equal(t, 0.5, snapshot.HitRate)
	assert.Equal(t, int64(100), snapshot.TokensSaved)
	assert.Equal(t, int64(1), snapshot.RequestsSaved)
	assert.Equal(t, int64(1), snapshot.Evictions)
	assert.Equal(t, int64(1), snapshot.Errors)
	assert.Greater(t, snapshot.Timestamp, int64(0))
}

func TestStats_Reset(t *testing.T) {
	stats := NewStats()

	// Add some data
	stats.RecordHit(time.Millisecond)
	stats.RecordMiss(time.Millisecond)
	stats.RecordTokensSaved(100)
	stats.RecordRequestSaved()
	stats.RecordEviction()
	stats.RecordError()

	// Reset
	stats.Reset()

	// Verify all zeros
	snapshot := stats.Snapshot()
	assert.Equal(t, int64(0), snapshot.Hits)
	assert.Equal(t, int64(0), snapshot.Misses)
	assert.Equal(t, int64(0), snapshot.TokensSaved)
	assert.Equal(t, int64(0), snapshot.RequestsSaved)
	assert.Equal(t, int64(0), snapshot.Evictions)
	assert.Equal(t, int64(0), snapshot.Errors)
	assert.Equal(t, int64(0), snapshot.TotalOperations)
}

func TestStats_Concurrent(t *testing.T) {
	stats := NewStats()
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent hits
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			stats.RecordHit(time.Millisecond)
		}
	}()

	// Concurrent misses
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			stats.RecordMiss(time.Millisecond)
		}
	}()

	// Concurrent token saves
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			stats.RecordTokensSaved(10)
		}
	}()

	wg.Wait()

	snapshot := stats.Snapshot()
	assert.Equal(t, int64(iterations), snapshot.Hits)
	assert.Equal(t, int64(iterations), snapshot.Misses)
	assert.Equal(t, int64(iterations*10), snapshot.TokensSaved)
}

// StatsCollector Tests

func TestNewStatsCollector(t *testing.T) {
	collector := NewStatsCollector()
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.stats)
}

func TestStatsCollector_GetStats(t *testing.T) {
	collector := NewStatsCollector()

	// Get stats for new cache
	stats1 := collector.GetStats("cache1")
	assert.NotNil(t, stats1)

	// Record some data
	stats1.RecordHit(time.Millisecond)

	// Get same stats again
	stats1Again := collector.GetStats("cache1")
	snapshot := stats1Again.Snapshot()
	assert.Equal(t, int64(1), snapshot.Hits)
}

func TestStatsCollector_AllStats(t *testing.T) {
	collector := NewStatsCollector()

	// Add multiple caches
	stats1 := collector.GetStats("cache1")
	stats1.RecordHit(time.Millisecond)

	stats2 := collector.GetStats("cache2")
	stats2.RecordMiss(time.Millisecond)
	stats2.RecordMiss(time.Millisecond)

	allStats := collector.AllStats()
	assert.Len(t, allStats, 2)
	assert.Equal(t, int64(1), allStats["cache1"].Hits)
	assert.Equal(t, int64(2), allStats["cache2"].Misses)
}

func TestGlobalStatsCollector(t *testing.T) {
	// Test global functions
	stats := GetCacheStats("global-test")
	assert.NotNil(t, stats)

	stats.RecordHit(time.Millisecond)

	allStats := GetAllCacheStats()
	assert.Contains(t, allStats, "global-test")
}

// TrackedCache Tests

type mockCache struct {
	data map[string]interface{}
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]interface{}),
	}
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, ok := m.data[key]
	if !ok {
		return ErrNotFound
	}
	// Simple assignment for testing
	switch v := dest.(type) {
	case *string:
		*v = val.(string)
	case *int:
		*v = val.(int)
	}
	return nil
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) DeleteByPattern(ctx context.Context, pattern string) error {
	// Simple implementation for testing
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func TestNewTrackedCache(t *testing.T) {
	mock := newMockCache()
	tc := NewTrackedCache(mock, "test-cache")

	assert.NotNil(t, tc)
	assert.Equal(t, mock, tc.Cache)
	assert.NotNil(t, tc.stats)
	assert.Equal(t, "test-cache", tc.name)
}

func TestTrackedCache_Get_Hit(t *testing.T) {
	mock := newMockCache()
	mock.data["key1"] = "value1"

	tc := NewTrackedCache(mock, "test-cache")
	var result string
	err := tc.Get(context.Background(), "key1", &result)

	require.NoError(t, err)
	assert.Equal(t, "value1", result)

	snapshot := tc.GetStats().Snapshot()
	assert.Equal(t, int64(1), snapshot.Hits)
	assert.Equal(t, int64(0), snapshot.Misses)
}

func TestTrackedCache_Get_Miss(t *testing.T) {
	mock := newMockCache()
	tc := NewTrackedCache(mock, "test-cache")

	var result string
	err := tc.Get(context.Background(), "nonexistent", &result)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)

	snapshot := tc.GetStats().Snapshot()
	assert.Equal(t, int64(1), snapshot.Misses) // Should record a miss
}

func TestTrackedCache_Set(t *testing.T) {
	mock := newMockCache()
	tc := NewTrackedCache(mock, "test-cache")

	err := tc.Set(context.Background(), "key1", "value1", time.Minute)
	require.NoError(t, err)

	assert.Equal(t, "value1", mock.data["key1"])
}

func TestTrackedCache_Delete(t *testing.T) {
	mock := newMockCache()
	mock.data["key1"] = "value1"

	tc := NewTrackedCache(mock, "test-cache")
	err := tc.Delete(context.Background(), "key1")

	require.NoError(t, err)
	_, exists := mock.data["key1"]
	assert.False(t, exists)
}
