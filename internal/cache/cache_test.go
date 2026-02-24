package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseCache_New(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)
	assert.NotNil(t, rc)
	assert.Equal(t, "ai-response:", rc.prefix)
}

func TestResponseCache_GenerateKey(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)

	req := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}

	key, err := rc.GenerateKey("openai", "gpt-4", req)
	require.NoError(t, err)
	assert.Contains(t, key, "ai-response:")
	assert.Contains(t, key, "openai")
	assert.Contains(t, key, "gpt-4")
}

func TestResponseCache_GenerateKey_Consistency(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)

	req := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}

	key1, err := rc.GenerateKey("openai", "gpt-4", req)
	require.NoError(t, err)

	key2, err := rc.GenerateKey("openai", "gpt-4", req)
	require.NoError(t, err)

	// Same request should generate same key
	assert.Equal(t, key1, key2)
}

func TestResponseCache_GenerateKey_DifferentRequests(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)

	req1 := map[string]interface{}{
		"model":    "gpt-4",
		"messages": []map[string]string{{"role": "user", "content": "Hello"}},
	}

	req2 := map[string]interface{}{
		"model":    "gpt-4",
		"messages": []map[string]string{{"role": "user", "content": "World"}},
	}

	key1, err := rc.GenerateKey("openai", "gpt-4", req1)
	require.NoError(t, err)

	key2, err := rc.GenerateKey("openai", "gpt-4", req2)
	require.NoError(t, err)

	// Different requests should generate different keys
	assert.NotEqual(t, key1, key2)
}

func TestResponseCache_SetAndGet(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)
	ctx := context.Background()

	key := "test-cache-key"
	response := &CachedResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"result": "success"}`),
		CreatedAt:  time.Now(),
		Provider:   "openai",
		Model:      "gpt-4",
	}

	err := rc.Set(ctx, key, response)
	require.NoError(t, err)

	result, err := rc.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, response.StatusCode, result.StatusCode)
	assert.Equal(t, response.Provider, result.Provider)
	assert.Equal(t, response.Model, result.Model)
}

func TestResponseCache_SetWithTTL(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)
	ctx := context.Background()

	key := "custom-ttl-key"
	response := &CachedResponse{
		StatusCode: 200,
		Body:       []byte(`{"ok": true}`),
		CreatedAt:  time.Now(),
		Provider:   "openai",
		Model:      "gpt-4",
	}

	err := rc.SetWithTTL(ctx, key, response, 5*time.Second)
	require.NoError(t, err)

	meta := memCache.GetMeta(key)
	require.NotNil(t, meta)
	assert.Equal(t, 5, meta.TTL)
}

func TestResponseCache_Get_NotFound(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)
	ctx := context.Background()

	_, err := rc.Get(ctx, "non-existent-key")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestResponseCache_Delete(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)
	ctx := context.Background()

	key := "delete-test-key"
	response := &CachedResponse{
		StatusCode: 200,
		Body:       []byte(`{"test": "data"}`),
		CreatedAt:  time.Now(),
	}

	err := rc.Set(ctx, key, response)
	require.NoError(t, err)

	err = rc.Delete(ctx, key)
	require.NoError(t, err)

	_, err = rc.Get(ctx, key)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestResponseCache_IsCacheable(t *testing.T) {
	memCache := NewMemoryCache()
	rc := NewResponseCache(memCache, time.Hour)

	// Test streamable request
	streamReq := &mockStreamableRequest{stream: true}
	assert.False(t, rc.IsCacheable(streamReq))

	// Test non-streamable request
	nonStreamReq := &mockStreamableRequest{stream: false}
	assert.True(t, rc.IsCacheable(nonStreamReq))

	// Test regular request (no IsStream method)
	regularReq := map[string]string{"test": "data"}
	assert.True(t, rc.IsCacheable(regularReq))
}

// mockStreamableRequest for testing IsCacheable
type mockStreamableRequest struct {
	stream bool
}

func (r *mockStreamableRequest) IsStream() bool {
	return r.stream
}

func TestCachedResponse_JSON(t *testing.T) {
	response := CachedResponse{
		StatusCode: 200,
		Headers:    map[string]string{"X-Custom": "value"},
		Body:       []byte(`{"message": "Hello"}`),
		CreatedAt:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Provider:   "anthropic",
		Model:      "claude-3",
	}

	// Verify JSON marshaling works
	data, err := response.Body.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), "Hello")
}
