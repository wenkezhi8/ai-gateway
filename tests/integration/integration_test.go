package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/handler"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_FullRequestFlow(t *testing.T) {
	setGinTestMode()

	// Clear global registry before test
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	// Setup configuration
	cfg := &config.Config{
		Server: config.ServerConfig{Port: "8080", Mode: "test"},
		Providers: []config.ProviderConfig{
			{Name: "mock-provider", APIKey: "test-key", BaseURL: "http://mock", Enabled: true},
		},
		Limiter: config.LimiterConfig{
			Enabled: true,
			Rate:    1000,
			Burst:   2000,
			PerUser: false,
		},
	}

	// Setup router
	ginRouter := gin.New()

	// Setup handlers
	proxyHandler := handler.NewProxyHandler(cfg, nil, nil)
	healthHandler := handler.NewHealthHandler()

	// Register a mock provider for testing
	mockProvider := &mockProviderForIntegration{
		name:    "mock-provider",
		enabled: true,
	}
	provider.RegisterProvider("mock-provider", mockProvider)

	// Register routes
	ginRouter.GET("/health", healthHandler.Check)
	ginRouter.GET("/api/v1/providers", proxyHandler.ListProviders)
	ginRouter.POST("/api/v1/chat/completions", proxyHandler.ChatCompletions)

	// Test health check
	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", http.NoBody)
		w := httptest.NewRecorder()
		ginRouter.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
	})

	// Test list providers
	t.Run("ListProviders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/providers", http.NoBody)
		w := httptest.NewRecorder()
		ginRouter.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "mock-provider")
	})

	// Test chat completions
	t.Run("ChatCompletions", func(t *testing.T) {
		body := map[string]interface{}{
			"model": "gpt-4",
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ginRouter.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIntegration_CacheWithRequests(t *testing.T) {
	setGinTestMode()

	memCache := cache.NewMemoryCache()
	responseCache := cache.NewResponseCache(memCache, time.Hour)
	ctx := context.Background()

	// Generate cache key for a request
	req := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "What is AI?"},
		},
	}

	key, err := responseCache.GenerateKey("openai", "gpt-4", req)
	require.NoError(t, err)

	// Cache a response
	cachedResp := &cache.CachedResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       json.RawMessage(`{"choices":[{"message":{"content":"AI is..."}}]}`),
		CreatedAt:  time.Now(),
		Provider:   "openai",
		Model:      "gpt-4",
	}

	err = responseCache.Set(ctx, key, cachedResp)
	require.NoError(t, err)

	// Retrieve cached response
	retrieved, err := responseCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, cachedResp.Provider, retrieved.Provider)
	assert.Equal(t, cachedResp.Model, retrieved.Model)
}

func TestIntegration_QuotaManagement(t *testing.T) {
	setGinTestMode()

	// Setup mock store
	store := newMockStore()
	tracker := limiter.NewLegacyUsageTracker(store)
	manager := limiter.NewQuotaManager(tracker)
	ctx := context.Background()

	// Set user quota
	quota := &limiter.QuotaConfig{
		UserID:     "user-integration-test",
		DailyLimit: 1000,
		Providers: map[string]int64{
			"openai": 500,
		},
	}
	manager.SetQuota(quota)

	// Simulate multiple requests
	for i := 0; i < 5; i++ {
		allowed, err := manager.CheckQuota(ctx, "user-integration-test", "openai")
		require.NoError(t, err)
		assert.True(t, allowed)

		err = manager.ConsumeQuota(ctx, "user-integration-test", "openai", 10)
		require.NoError(t, err)
	}

	// Verify quota was consumed
	_, ok := manager.GetQuota("user-integration-test")
	assert.True(t, ok)
}

func TestIntegration_RouterStrategy(t *testing.T) {
	setGinTestMode()

	// Test strategy parsing
	strategies := []struct {
		input    string
		expected router.StrategyType
	}{
		{"failover", router.StrategyFailover},
		{"roundrobin", router.StrategyRoundRobin},
		{"cost", router.StrategyCostOptimized},
		{"weighted", router.StrategyWeighted},
	}

	for _, s := range strategies {
		result := router.ParseStrategyType(s.input)
		assert.Equal(t, s.expected, result)
	}
}

type mockProviderForIntegration struct {
	name    string
	enabled bool
}

func (m *mockProviderForIntegration) Name() string                       { return m.name }
func (m *mockProviderForIntegration) Models() []string                   { return []string{"gpt-4", "gpt-3.5-turbo"} }
func (m *mockProviderForIntegration) IsEnabled() bool                    { return m.enabled }
func (m *mockProviderForIntegration) SetEnabled(enabled bool)            { m.enabled = enabled }
func (m *mockProviderForIntegration) ValidateKey(_ context.Context) bool { return true }
func (m *mockProviderForIntegration) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "test-response-id",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "Test response from mock provider",
				},
				FinishReason: "stop",
			},
		},
		Usage: provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}
func (m *mockProviderForIntegration) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "test-stream-id",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []provider.StreamChoice{
				{
					Index: 0,
					Delta: &provider.StreamDelta{
						Role:    "assistant",
						Content: "Test",
					},
				},
			},
			Done: true,
		}
	}()
	return ch, nil
}

func TestIntegration_ProviderAvailability(t *testing.T) {
	setGinTestMode()

	// This would test with actual provider instances in a real integration test
	// For now, we test the routing logic

	providers := []*router.ProviderInfo{
		{
			Provider:   &mockProviderForIntegration{name: "primary", enabled: true},
			Healthy:    true,
			QuotaLimit: 1000,
			QuotaUsed:  500,
			IsPrimary:  true,
		},
		{
			Provider:   &mockProviderForIntegration{name: "backup", enabled: true},
			Healthy:    true,
			QuotaLimit: 500,
			QuotaUsed:  0,
			IsPrimary:  false,
		},
	}

	// Check availability
	availableCount := 0
	for _, p := range providers {
		if p.Available() {
			availableCount++
		}
	}
	assert.Equal(t, 2, availableCount)

	// Simulate primary failure
	providers[0].Healthy = false
	availableCount = 0
	for _, p := range providers {
		if p.Available() {
			availableCount++
		}
	}
	assert.Equal(t, 1, availableCount)
}

type mockStore struct {
	data map[string]int64
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]int64)}
}

func (m *mockStore) Get(_ context.Context, key string) (int64, error) {
	return m.data[key], nil
}

func (m *mockStore) Incr(_ context.Context, key string) (int64, error) {
	m.data[key]++
	return m.data[key], nil
}

func (m *mockStore) Expire(_ context.Context, key string, _ time.Duration) error {
	_ = key
	return nil
}
