package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_New(t *testing.T) {
	cache := NewMemoryCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.items)
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Test set and get
	original := &testData{Name: "test", Value: 42}
	err := cache.Set(ctx, "test-key", original, time.Hour)
	require.NoError(t, err)

	var result testData
	err = cache.Get(ctx, "test-key", &result)
	require.NoError(t, err)
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Value, result.Value)
}

func TestMemoryCache_Get_NotFound(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	var result string
	err := cache.Get(ctx, "non-existent-key", &result)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "delete-key", "test-value", time.Hour)
	require.NoError(t, err)

	// Verify it exists
	exists, err := cache.Exists(ctx, "delete-key")
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete it
	err = cache.Delete(ctx, "delete-key")
	require.NoError(t, err)

	// Verify it's gone
	exists, err = cache.Exists(ctx, "delete-key")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Test non-existent key
	exists, err := cache.Exists(ctx, "non-existent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Set a value
	err = cache.Set(ctx, "exists-key", "value", time.Hour)
	require.NoError(t, err)

	// Test existing key
	exists, err = cache.Exists(ctx, "exists-key")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set with short TTL
	err := cache.Set(ctx, "expiring-key", "value", 50*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	exists, err := cache.Exists(ctx, "expiring-key")
	require.NoError(t, err)
	assert.True(t, exists)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	var result string
	err = cache.Get(ctx, "expiring-key", &result)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryCache_Concurrent(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			key := string(rune('a' + id))
			for j := 0; j < 100; j++ {
				_ = cache.Set(ctx, key, j, time.Hour)
				var val int
				_ = cache.Get(ctx, key, &val)
				_, _ = cache.Exists(ctx, key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMemoryCache_Set_Overwrite(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Set initial value
	err := cache.Set(ctx, "overwrite-key", "initial", time.Hour)
	require.NoError(t, err)

	// Overwrite with new value
	err = cache.Set(ctx, "overwrite-key", "updated", time.Hour)
	require.NoError(t, err)

	// Verify updated value
	var result string
	err = cache.Get(ctx, "overwrite-key", &result)
	require.NoError(t, err)
	assert.Equal(t, "updated", result)
}

func TestMemoryCache_DifferentTypes(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "str-key", "hello world"},
		{"int", "int-key", 42},
		{"float", "float-key", 3.14159},
		{"bool", "bool-key", true},
		{"slice", "slice-key", []int{1, 2, 3}},
		{"map", "map-key", map[string]int{"a": 1, "b": 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.Set(ctx, tt.key, tt.value, time.Hour)
			require.NoError(t, err)

			exists, err := cache.Exists(ctx, tt.key)
			require.NoError(t, err)
			assert.True(t, exists)
		})
	}
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	cache := NewMemoryCacheWithMaxEntries(2)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "k1", "v1", time.Hour))
	require.NoError(t, cache.Set(ctx, "k2", "v2", time.Hour))
	require.NoError(t, cache.Set(ctx, "k3", "v3", time.Hour))

	var v string
	err := cache.Get(ctx, "k1", &v)
	assert.ErrorIs(t, err, ErrNotFound)

	err = cache.Get(ctx, "k2", &v)
	require.NoError(t, err)
	assert.Equal(t, "v2", v)

	err = cache.Get(ctx, "k3", &v)
	require.NoError(t, err)
	assert.Equal(t, "v3", v)
}

func TestMemoryCache_MaxEntries(t *testing.T) {
	cache := NewMemoryCacheWithMaxEntries(3)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		require.NoError(t, cache.Set(ctx, string(rune('a'+i)), i, time.Hour))
	}

	assert.Len(t, cache.items, 3)
}

func TestMemoryCache_AccessUpdatesLRU(t *testing.T) {
	cache := NewMemoryCacheWithMaxEntries(2)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "k1", "v1", time.Hour))
	require.NoError(t, cache.Set(ctx, "k2", "v2", time.Hour))

	var v string
	require.NoError(t, cache.Get(ctx, "k1", &v))

	require.NoError(t, cache.Set(ctx, "k3", "v3", time.Hour))

	err := cache.Get(ctx, "k2", &v)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, cache.Get(ctx, "k1", &v))
	assert.Equal(t, "v1", v)
	require.NoError(t, cache.Get(ctx, "k3", &v))
	assert.Equal(t, "v3", v)
}

func TestMemoryCache_EvictionOrder(t *testing.T) {
	cache := NewMemoryCacheWithMaxEntries(3)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "k1", "v1", time.Hour))
	require.NoError(t, cache.Set(ctx, "k2", "v2", time.Hour))
	require.NoError(t, cache.Set(ctx, "k3", "v3", time.Hour))

	var v string
	require.NoError(t, cache.Get(ctx, "k1", &v))
	require.NoError(t, cache.Set(ctx, "k4", "v4", time.Hour))

	err := cache.Get(ctx, "k2", &v)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, cache.Get(ctx, "k3", &v))
	require.NoError(t, cache.Set(ctx, "k5", "v5", time.Hour))

	err = cache.Get(ctx, "k1", &v)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, cache.Get(ctx, "k3", &v))
	require.NoError(t, cache.Get(ctx, "k4", &v))
	require.NoError(t, cache.Get(ctx, "k5", &v))
}
