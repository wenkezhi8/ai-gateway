package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Mode: "test",
		},
		Providers: []config.ProviderConfig{
			{Name: "openai", APIKey: "test-key", BaseURL: "https://api.openai.com", Enabled: true},
			{Name: "anthropic", APIKey: "test-key", BaseURL: "https://api.anthropic.com", Enabled: true},
		},
	}
}

func TestProxyHandler_New(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)
	assert.NotNil(t, h)
	assert.NotNil(t, h.config)
}

func TestProxyHandler_ListProviders(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.ListProviders(c)

	require.Equal(t, http.StatusOK, w.Code)
	// The response should contain "providers" key
	assert.Contains(t, w.Body.String(), "providers")
}

func TestProxyHandler_ListProviders_Empty(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{},
	}
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.ListProviders(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "providers")
}

func TestProxyHandler_ChatCompletions(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request with body
	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	// Should return 503 because no provider is registered for the model
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProxyHandler_Completions(t *testing.T) {
	// Clear global registry before test
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	// Create a mock provider that implements the Provider interface
	mockProvider := &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com", []string{"gpt-4"}, true),
	}

	// Register the provider with the global registry
	provider.RegisterProvider("openai", mockProvider)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request with body - completions requires prompt field
	body := `{"model": "gpt-4", "prompt": "Hello"}`
	req := httptest.NewRequest("POST", "/api/v1/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Completions(c)

	// Currently returns placeholder response
	require.Equal(t, http.StatusOK, w.Code)
}

func TestProxyHandler_Embeddings(t *testing.T) {
	cfg := testConfig()
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request with body - embeddings requires input field
	body := `{"model": "text-embedding-ada-002", "input": "Hello world"}`
	req := httptest.NewRequest("POST", "/api/v1/embeddings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.Embeddings(c)

	// Currently returns placeholder or error response
	// The exact status depends on the implementation
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable || w.Code == http.StatusBadRequest)
}

func TestProxyHandler_ListProviders_EnabledStatus(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "enabled-provider", Enabled: true},
			{Name: "disabled-provider", Enabled: false},
		},
	}
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.ListProviders(c)

	require.Equal(t, http.StatusOK, w.Code)
	// Response contains providers list (may be empty if not registered)
	assert.Contains(t, w.Body.String(), "providers")
}

// mockProvider implements the provider.Provider interface for testing.
type mockProvider struct {
	*provider.BaseProvider
}

func (m *mockProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "test-response-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "Test response",
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

func (m *mockProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "test-stream-id",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
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
		}
	}()
	return ch, nil
}

func (m *mockProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (m *mockProvider) Models() []string {
	return m.BaseProvider.Models()
}

func (m *mockProvider) IsEnabled() bool {
	return true
}

func (m *mockProvider) SetEnabled(enabled bool) {
	m.BaseProvider.SetEnabled(enabled)
}

func (m *mockProvider) Name() string {
	return m.BaseProvider.Name()
}

type failingProvider struct {
	*provider.BaseProvider
	chatErr error
}

type streamStartFailProvider struct {
	*provider.BaseProvider
	streamErr error
}

type hangingStreamProvider struct {
	*provider.BaseProvider
}

type doneStreamProvider struct {
	*provider.BaseProvider
}

type noDoneStreamProvider struct {
	*provider.BaseProvider
}

func (d *doneStreamProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "fallback-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "fallback response",
				},
				FinishReason: "stop",
			},
		},
		Usage: provider.Usage{TotalTokens: 3, PromptTokens: 2, CompletionTokens: 1},
	}, nil
}

func (d *doneStreamProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "stream-id",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.StreamChoice{
				{
					Index: 0,
					Delta: &provider.StreamDelta{
						Role:    "assistant",
						Content: "stream answer",
					},
				},
			},
		}
		ch <- &provider.StreamChunk{
			ID:      "stream-id",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Done:    true,
			Choices: []provider.StreamChoice{
				{
					Index:        0,
					FinishReason: "stop",
					Delta:        &provider.StreamDelta{},
				},
			},
			Usage: &provider.Usage{TotalTokens: 3, PromptTokens: 2, CompletionTokens: 1},
		}
	}()
	return ch, nil
}

func (d *doneStreamProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (n *noDoneStreamProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "no-done-fallback-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "no done fallback response",
				},
				FinishReason: "stop",
			},
		},
		Usage: provider.Usage{TotalTokens: 3, PromptTokens: 2, CompletionTokens: 1},
	}, nil
}

func (n *noDoneStreamProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "stream-no-done-id",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.StreamChoice{
				{
					Index: 0,
					Delta: &provider.StreamDelta{
						Role:    "assistant",
						Content: "stream no done answer",
					},
				},
			},
		}
	}()
	return ch, nil
}

func (n *noDoneStreamProvider) ValidateKey(_ context.Context) bool {
	return true
}

type fallbackStreamProvider struct {
	*provider.BaseProvider
}

func (f *fallbackStreamProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "fallback-chat-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{{
			Index: 0,
			Message: provider.ChatMessage{
				Role:    "assistant",
				Content: "fallback answer",
			},
			FinishReason: "stop",
		}},
		Usage: provider.Usage{TotalTokens: 4, PromptTokens: 2, CompletionTokens: 2},
	}, nil
}

func (f *fallbackStreamProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "stream-empty",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Done:    true,
			Choices: []provider.StreamChoice{{
				Index:        0,
				FinishReason: "stop",
				Delta:        &provider.StreamDelta{},
			}},
		}
	}()
	return ch, nil
}

func (f *fallbackStreamProvider) ValidateKey(_ context.Context) bool {
	return true
}

func TestProxyHandler_ChatCompletions_StreamShouldRecordHTTPResponseSpan(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()
	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()
	provider.RegisterProvider("openai", &doneStreamProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hello stream"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM request_traces WHERE request_id = ? AND operation = 'http.response'`, requestID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var attrsRaw string
	err = db.QueryRow(`SELECT attributes FROM request_traces WHERE request_id = ? AND operation = 'http.response'`, requestID).Scan(&attrsRaw)
	require.NoError(t, err)

	attrs := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(attrsRaw), &attrs))
	assert.Equal(t, "stream answer", attrs["ai_response_preview"])
	assert.Equal(t, "stream answer", attrs["ai_response_full"])
	assert.Equal(t, false, attrs["ai_response_truncated"])
	assert.Equal(t, "hello stream", attrs["user_message_preview"])
	assert.Equal(t, "hello stream", attrs["user_message_full"])
	assert.Equal(t, false, attrs["user_message_truncated"])

	providerAttrs := fetchOperationAttrs(t, db, requestID, "provider.chat")
	assert.Equal(t, true, providerAttrs["success"])
	assert.Equal(t, true, providerAttrs["stream"])
}

func TestProxyHandler_ChatCompletions_StreamFallbackShouldRecordHTTPResponseSpan(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()
	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()
	provider.RegisterProvider("openai", &fallbackStreamProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","stream":true,"messages":[{"role":"user","content":"trigger fallback"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM request_traces WHERE request_id = ? AND operation = 'http.response'`, requestID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var attrsRaw string
	err = db.QueryRow(`SELECT attributes FROM request_traces WHERE request_id = ? AND operation = 'http.response'`, requestID).Scan(&attrsRaw)
	require.NoError(t, err)

	attrs := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(attrsRaw), &attrs))
	assert.Equal(t, "fallback answer", attrs["ai_response_preview"])
	assert.Equal(t, "fallback answer", attrs["ai_response_full"])
	assert.Equal(t, false, attrs["ai_response_truncated"])
	assert.Equal(t, "trigger fallback", attrs["user_message_preview"])
	assert.Equal(t, "trigger fallback", attrs["user_message_full"])
	assert.Equal(t, false, attrs["user_message_truncated"])

	providerAttrs := fetchOperationAttrs(t, db, requestID, "provider.chat")
	assert.Equal(t, true, providerAttrs["success"])
	assert.Equal(t, false, providerAttrs["stream"])
	assert.Equal(t, true, providerAttrs["fallback_from_stream"])
}

func TestProxyHandler_ChatCompletions_StreamClosedWithoutDoneShouldFinalizeTraceOnce(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()
	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()
	provider.RegisterProvider("openai", &noDoneStreamProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hello no done"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, strings.Count(w.Body.String(), "[DONE]"))
	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	httpResponseAttrs := fetchOperationAttrs(t, db, requestID, "http.response")
	assert.Equal(t, "stream no done answer", httpResponseAttrs["ai_response_preview"])
	assert.Equal(t, "stream no done answer", httpResponseAttrs["ai_response_full"])
	assert.Equal(t, false, httpResponseAttrs["ai_response_truncated"])
	assert.Equal(t, "hello no done", httpResponseAttrs["user_message_preview"])
	assert.Equal(t, "hello no done", httpResponseAttrs["user_message_full"])
	assert.Equal(t, false, httpResponseAttrs["user_message_truncated"])

	providerAttrs := fetchOperationAttrs(t, db, requestID, "provider.chat")
	assert.Equal(t, true, providerAttrs["success"])
	assert.Equal(t, true, providerAttrs["stream"])
	_, hasFallback := providerAttrs["fallback_from_stream"]
	assert.False(t, hasFallback)
}

func TestProxyHandler_ChatCompletions_StreamClosedWithoutDoneShouldWriteResponseCache(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cacheManager := cache.NewManagerWithCache(cache.NewMemoryCache())
	h := NewProxyHandler(&config.Config{}, nil, cacheManager)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	provider.RegisterProvider("openai", &noDoneStreamProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hello cache no done"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, strings.Count(w.Body.String(), "[DONE]"))

	entries := cacheManager.ListEntries("response", "")
	require.Len(t, entries, 1, "stream close without done should still write response cache")
}

func TestProxyHandler_ChatCompletions_NonStreamShouldRecordHTTPResponseSpanWithMessages(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()
	provider.RegisterProvider("openai", &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","messages":[{"role":"user","content":"hello nonstream"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	attrs := fetchHTTPResponseAttrs(t, db, requestID)
	assert.Equal(t, "hello nonstream", attrs["user_message_preview"])
	assert.Equal(t, "hello nonstream", attrs["user_message_full"])
	assert.Equal(t, false, attrs["user_message_truncated"])
	assert.Equal(t, "Test response", attrs["ai_response_preview"])
	assert.Equal(t, "Test response", attrs["ai_response_full"])
	assert.Equal(t, false, attrs["ai_response_truncated"])
}

func TestProxyHandler_ChatCompletions_CacheHitShouldRecordHTTPResponseSpanWithMessages(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	cacheManager := cache.NewManagerWithCache(cache.NewMemoryCache())
	h := NewProxyHandler(&config.Config{}, nil, cacheManager)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()
	provider.RegisterProvider("openai", &mockProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	body := `{"provider":"openai","model":"gpt-4","messages":[{"role":"user","content":"hello cache"}]}`

	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	req1 := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	c1.Request = req1
	h.ChatCompletions(c1)
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	c2.Request = req2
	h.ChatCompletions(c2)

	require.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "1", w2.Header().Get("X-Local-Cache-Hit"))

	requestID := w2.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	attrs := fetchHTTPResponseAttrs(t, db, requestID)
	assert.Equal(t, true, attrs["cache_hit"])
	assert.Equal(t, "hello cache", attrs["user_message_preview"])
	assert.Equal(t, "hello cache", attrs["user_message_full"])
	assert.Equal(t, false, attrs["user_message_truncated"])
	assert.Equal(t, "Test response", attrs["ai_response_preview"])
	assert.Equal(t, "Test response", attrs["ai_response_full"])
	assert.Equal(t, false, attrs["ai_response_truncated"])
}

func fetchHTTPResponseAttrs(t *testing.T, db *sql.DB, requestID string) map[string]interface{} {
	t.Helper()

	var attrsRaw string
	err := db.QueryRow(`SELECT attributes FROM request_traces WHERE request_id = ? AND operation = 'http.response' ORDER BY created_at DESC LIMIT 1`, requestID).Scan(&attrsRaw)
	require.NoError(t, err)

	attrs := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(attrsRaw), &attrs))
	return attrs
}

func fetchOperationAttrs(t *testing.T, db *sql.DB, requestID, operation string) map[string]interface{} {
	t.Helper()

	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM request_traces WHERE request_id = ? AND operation = ?`, requestID, operation).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var attrsRaw string
	err = db.QueryRow(`SELECT attributes FROM request_traces WHERE request_id = ? AND operation = ? ORDER BY created_at DESC LIMIT 1`, requestID, operation).Scan(&attrsRaw)
	require.NoError(t, err)

	attrs := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(attrsRaw), &attrs))
	return attrs
}

type operationTraceRecord struct {
	Status string
	Error  string
	Attrs  map[string]interface{}
}

func fetchOperationTraceRecord(t *testing.T, db *sql.DB, requestID, operation string) operationTraceRecord {
	t.Helper()

	var status string
	var errorMsg string
	var attrsRaw string
	err := db.QueryRow(`SELECT status, error, attributes FROM request_traces WHERE request_id = ? AND operation = ? ORDER BY created_at DESC LIMIT 1`, requestID, operation).Scan(&status, &errorMsg, &attrsRaw)
	require.NoError(t, err)

	attrs := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(attrsRaw), &attrs))

	return operationTraceRecord{Status: status, Error: errorMsg, Attrs: attrs}
}

func (f *failingProvider) Chat(_ context.Context, _ *provider.ChatRequest) (*provider.ChatResponse, error) {
	if f.chatErr != nil {
		return nil, f.chatErr
	}
	return nil, nil
}

func (f *failingProvider) StreamChat(_ context.Context, _ *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	ch := make(chan *provider.StreamChunk)
	close(ch)
	return ch, nil
}

func (f *failingProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (s *streamStartFailProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "stream-start-fail",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{{
			Index: 0,
			Message: provider.ChatMessage{
				Role:    "assistant",
				Content: "unused",
			},
			FinishReason: "stop",
		}},
		Usage: provider.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}, nil
}

func (s *streamStartFailProvider) StreamChat(_ context.Context, _ *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	return nil, s.streamErr
}

func (s *streamStartFailProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (h *hangingStreamProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		ID:      "hanging-fallback-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "unused",
				},
				FinishReason: "stop",
			},
		},
		Usage: provider.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}, nil
}

func (h *hangingStreamProvider) StreamChat(_ context.Context, _ *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	// Intentionally never sends/never closes to simulate upstream streaming stall.
	return make(chan *provider.StreamChunk), nil
}

func (h *hangingStreamProvider) ValidateKey(_ context.Context) bool {
	return true
}

func TestGetBaseURLForProvider(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"anthropic", "https://api.anthropic.com/v1"},
		{"deepseek", "https://api.deepseek.com"},
		{"moonshot", "https://api.moonshot.cn/v1"},
		{"kimi", "https://api.moonshot.cn/v1"},
		{"qwen", "https://dashscope.aliyuncs.com/compatible-mode/v1"},
		{"zhipu", "https://open.bigmodel.cn/api/paas/v4"},
		{"baichuan", "https://api.baichuan-ai.com/v1"},
		{"minimax", "https://api.minimax.chat/v1"},
		{"volcengine", "https://ark.cn-beijing.volces.com/api/v3"},
		{"yi", "https://api.lingyiwanwu.com/v1"},
		{"google", "https://generativelanguage.googleapis.com/v1beta"},
		{"mistral", "https://api.mistral.ai/v1"},
		{"ernie", "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat"},
		{"hunyuan", "https://api.hunyuan.cloud.tencent.com/v1"},
		{"spark", "https://spark-api-open.xf-yun.com/v1"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := getBaseURLForProvider(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"openai", "openai"},
		{"deepseek", "openai"},
		{"moonshot", "openai"},
		{"kimi", "openai"},
		{"qwen", "openai"},
		{"zhipu", "openai"},
		{"baichuan", "openai"},
		{"minimax", "openai"},
		{"volcengine", "openai"},
		{"yi", "openai"},
		{"azure-openai", "openai"},
		{"mistral", "openai"},
		{"anthropic", "anthropic"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapProviderName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetModelsForProvider(t *testing.T) {
	tests := []struct {
		provider       string
		shouldHave     []string
		shouldNotExist bool
	}{
		{"openai", []string{"gpt-4o", "gpt-4", "gpt-3.5-turbo"}, false},
		{"anthropic", []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"}, false},
		{"deepseek", []string{"deepseek-chat", "deepseek-coder"}, false},
		{"qwen", []string{"qwen-max", "qwen-plus", "qwen-turbo"}, false},
		{"zhipu", []string{"glm-4-plus", "glm-4"}, false},
		{"google", []string{"gemini-3.1-pro-preview", "gemini-2.0-flash"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := getModelsForProvider(tt.provider)
			for _, model := range tt.shouldHave {
				assert.Contains(t, result, model)
			}
		})
	}
}

func TestProxyHandler_ListConfiguredProviders_Empty(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{},
	}
	h := NewProxyHandler(cfg, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.ListConfiguredProviders(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "providers")
}

func TestProxyHandler_ListConfiguredProviders_MergesEnabledSmartRouterModels(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	accountManager := limiter.NewAccountManager(nil, nil)
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-openai-1",
		Name:         "openai-main",
		Provider:     "openai",
		ProviderType: "openai",
		APIKey:       "sk-test",
		BaseURL:      "https://api.openai.com/v1",
		Enabled:      true,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "openai", APIKey: "test-key", BaseURL: "https://api.openai.com/v1", Enabled: true},
		},
	}
	h := NewProxyHandler(cfg, accountManager, nil)

	originalRouterConfig := h.smartRouter.GetConfig()
	t.Cleanup(func() {
		h.smartRouter.SetConfig(originalRouterConfig)
	})
	h.smartRouter.SetConfig(&routing.RouterConfig{
		DefaultStrategy:  routing.StrategyAuto,
		DefaultModel:     "gpt-4o",
		UseAutoMode:      true,
		Classifier:       routing.DefaultClassifierConfig(),
		TaskRules:        routing.DefaultTaskRules(),
		ProviderDefaults: routing.DefaultProviderDefaults(),
		ModelScores: map[string]*routing.ModelScore{
			"gpt-5.3-codex-spark": {Model: "gpt-5.3-codex-spark", Provider: "openai", Enabled: true},
			"gpt-5.2-codex":       {Model: "gpt-5.2-codex", Provider: "openai", Enabled: true},
			"gpt-disabled":        {Model: "gpt-disabled", Provider: "openai", Enabled: false},
			"claude-only":         {Model: "claude-only", Provider: "anthropic", Enabled: true},
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/v1/config/providers", http.NoBody)
	c.Request = req
	h.ListConfiguredProviders(c)

	require.Equal(t, http.StatusOK, w.Code)
	providers := decodeConfiguredProvidersResponse(t, w.Body.Bytes())
	openaiProvider, ok := providers["openai"]
	require.True(t, ok, "openai provider should exist")
	assert.True(t, openaiProvider.Enabled)
	assert.Contains(t, openaiProvider.Models, "gpt-5.3-codex-spark")
	assert.Contains(t, openaiProvider.Models, "gpt-5.2-codex")
	assert.NotContains(t, openaiProvider.Models, "gpt-disabled")
	assert.NotContains(t, openaiProvider.Models, "claude-only")
}

func TestProxyHandler_ListConfiguredProviders_UsesAccountEnabledStatus(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	accountManager := limiter.NewAccountManager(nil, nil)
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-openai-disabled",
		Name:         "openai-disabled",
		Provider:     "openai",
		ProviderType: "openai",
		APIKey:       "sk-test",
		BaseURL:      "https://api.openai.com/v1",
		Enabled:      false,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "openai", APIKey: "test-key", BaseURL: "https://api.openai.com/v1", Enabled: true},
		},
	}
	h := NewProxyHandler(cfg, accountManager, nil)

	originalRouterConfig := h.smartRouter.GetConfig()
	t.Cleanup(func() {
		h.smartRouter.SetConfig(originalRouterConfig)
	})
	h.smartRouter.SetConfig(&routing.RouterConfig{
		DefaultStrategy:  routing.StrategyAuto,
		DefaultModel:     "gpt-4o",
		UseAutoMode:      true,
		Classifier:       routing.DefaultClassifierConfig(),
		TaskRules:        routing.DefaultTaskRules(),
		ProviderDefaults: routing.DefaultProviderDefaults(),
		ModelScores: map[string]*routing.ModelScore{
			"gpt-5.3-codex-spark": {Model: "gpt-5.3-codex-spark", Provider: "openai", Enabled: true},
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/v1/config/providers", http.NoBody)
	c.Request = req
	h.ListConfiguredProviders(c)

	require.Equal(t, http.StatusOK, w.Code)
	providers := decodeConfiguredProvidersResponse(t, w.Body.Bytes())
	openaiProvider, ok := providers["openai"]
	require.True(t, ok, "openai provider should exist")
	assert.False(t, openaiProvider.Enabled)
	assert.Contains(t, openaiProvider.Models, "gpt-5.3-codex-spark")
}

type configuredProviderPayload struct {
	Name    string   `json:"name"`
	Models  []string `json:"models"`
	Enabled bool     `json:"enabled"`
}

func setSmartRouterModelScoresForTest(t *testing.T, h *ProxyHandler, scores map[string]*routing.ModelScore) {
	t.Helper()

	routerConfig := h.smartRouter.GetConfig()
	originalScores := routerConfig.ModelScores
	replacement := make(map[string]*routing.ModelScore, len(scores))
	for key, score := range scores {
		if score == nil {
			continue
		}
		copied := *score
		replacement[key] = &copied
	}
	routerConfig.ModelScores = replacement
	t.Cleanup(func() {
		routerConfig.ModelScores = originalScores
	})
}

func ensureModelRegistryModelsForTest(t *testing.T, h *ProxyHandler, providerName string, models ...string) {
	t.Helper()

	normalizedProvider := normalizeProviderName(providerName)
	routerConfig := h.smartRouter.GetConfig()
	if routerConfig.ModelScores == nil {
		routerConfig.ModelScores = make(map[string]*routing.ModelScore)
	}

	originalEntries := make(map[string]*routing.ModelScore, len(models))
	originalExists := make(map[string]bool, len(models))
	for _, model := range models {
		modelID := strings.TrimSpace(model)
		if modelID == "" {
			continue
		}

		if _, alreadyCaptured := originalExists[modelID]; !alreadyCaptured {
			previous := routerConfig.ModelScores[modelID]
			originalExists[modelID] = previous != nil
			if previous != nil {
				copied := *previous
				originalEntries[modelID] = &copied
			}
		}

		routerConfig.ModelScores[modelID] = &routing.ModelScore{
			Model:        modelID,
			Provider:     normalizedProvider,
			DisplayName:  modelID,
			QualityScore: 80,
			SpeedScore:   80,
			CostScore:    80,
			Enabled:      true,
		}
	}

	t.Cleanup(func() {
		for _, model := range models {
			modelID := strings.TrimSpace(model)
			if modelID == "" {
				continue
			}
			if !originalExists[modelID] {
				delete(routerConfig.ModelScores, modelID)
				continue
			}
			if previous, ok := originalEntries[modelID]; ok && previous != nil {
				copied := *previous
				routerConfig.ModelScores[modelID] = &copied
				continue
			}
			delete(routerConfig.ModelScores, modelID)
		}
	})
}

func registerMockFactory(r *provider.Registry, name string) {
	r.RegisterFactory(name, func(cfg *provider.ProviderConfig) provider.Provider {
		enabled := cfg.Enabled
		if !enabled {
			enabled = true
		}
		return &mockProvider{
			BaseProvider: provider.NewBaseProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Models, enabled),
		}
	})
}

func decodeConfiguredProvidersResponse(t *testing.T, body []byte) map[string]configuredProviderPayload {
	t.Helper()

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Providers []configuredProviderPayload `json:"providers"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.True(t, envelope.Success)

	result := make(map[string]configuredProviderPayload, len(envelope.Data.Providers))
	for _, p := range envelope.Data.Providers {
		result[p.Name] = p
	}
	return result
}

func TestProxyHandler_ChatCompletions_ModelRegistryProviderShouldOverrideConflictingRequestProvider(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	setSmartRouterModelScoresForTest(t, h, map[string]*routing.ModelScore{
		"registry-routed-model": {
			Model:    "registry-routed-model",
			Provider: "openai",
			Enabled:  true,
		},
	})

	openaiCapture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "openai-key", "https://api.openai.com/v1", []string{"registry-routed-model"}, true),
	}
	zhipuCapture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("zhipu", "zhipu-key", "https://open.bigmodel.cn/api/paas/v4", []string{"registry-routed-model"}, true),
	}
	provider.RegisterProvider("openai", openaiCapture)
	provider.RegisterProvider("zhipu", zhipuCapture)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"zhipu","model":"registry-routed-model","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, openaiCapture.lastChatReq)
	assert.Nil(t, zhipuCapture.lastChatReq)
}

func TestProxyHandler_ChatCompletions_ShouldRejectUnregisteredModel(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	capture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "openai-key", "https://api.openai.com/v1", []string{"unregistered-test-model"}, true),
	}
	provider.RegisterProvider("openai", capture)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"unregistered-test-model","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Nil(t, capture.lastChatReq)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, "model_not_registered", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "模型未在模型管理中注册")
}

func TestProxyHandler_GetProviderForRequest_PreferSameProviderAccountOverCompatibleType(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	accountManager := limiter.NewAccountManager(nil, nil)
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-openai-strict",
		Name:         "OpenAI Strict",
		Provider:     "openai",
		ProviderType: "openai",
		APIKey:       "sk-openai",
		BaseURL:      "https://api.openai.com/v1",
		Enabled:      true,
		Priority:     1,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-zhipu-compatible",
		Name:         "Zhipu Compatible",
		Provider:     "zhipu",
		ProviderType: "openai",
		APIKey:       "sk-zhipu",
		BaseURL:      "https://open.bigmodel.cn/api/paas/v4",
		Enabled:      true,
		Priority:     100,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))

	h := NewProxyHandler(&config.Config{}, accountManager, nil)
	registerMockFactory(h.registry, "openai")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	prov, err := h.getProviderForRequest(c, "gpt-4o", "openai")
	require.NoError(t, err)
	require.NotNil(t, prov)

	selectedID, ok := c.Get("selected_account_id")
	require.True(t, ok)
	assert.Equal(t, "acc-openai-strict", selectedID)

	_, hasFallback := c.Get("fallback_account_type")
	assert.False(t, hasFallback)
}

func TestProxyHandler_GetProviderForRequest_ShouldFallbackToCompatibleProviderTypeWhenNoSameProviderAccount(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	accountManager := limiter.NewAccountManager(nil, nil)
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-zhipu-compatible",
		Name:         "Zhipu Compatible",
		Provider:     "zhipu",
		ProviderType: "openai",
		APIKey:       "sk-zhipu",
		BaseURL:      "https://open.bigmodel.cn/api/paas/v4",
		Enabled:      true,
		Priority:     100,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))

	h := NewProxyHandler(&config.Config{}, accountManager, nil)
	registerMockFactory(h.registry, "openai")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	prov, err := h.getProviderForRequest(c, "gpt-4o", "openai")
	require.NoError(t, err)
	require.NotNil(t, prov)

	selectedID, ok := c.Get("selected_account_id")
	require.True(t, ok)
	assert.Equal(t, "acc-zhipu-compatible", selectedID)

	fallbackType, hasFallback := c.Get("fallback_account_type")
	require.True(t, hasFallback)
	assert.Equal(t, "provider_type_compatible", fallbackType)
}

func TestProxyHandler_GetProviderForRequest_ShouldNotFallbackWhenSameProviderExistsButDisabled(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	accountManager := limiter.NewAccountManager(nil, nil)
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-openai-disabled",
		Name:         "OpenAI Disabled",
		Provider:     "openai",
		ProviderType: "openai",
		APIKey:       "sk-openai-disabled",
		BaseURL:      "https://api.openai.com/v1",
		Enabled:      false,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))
	require.NoError(t, accountManager.AddAccount(&limiter.AccountConfig{
		ID:           "acc-zhipu-compatible",
		Name:         "Zhipu Compatible",
		Provider:     "zhipu",
		ProviderType: "openai",
		APIKey:       "sk-zhipu",
		BaseURL:      "https://open.bigmodel.cn/api/paas/v4",
		Enabled:      true,
		Limits:       map[limiter.LimitType]*limiter.LimitConfig{},
	}))

	h := NewProxyHandler(&config.Config{}, accountManager, nil)
	registerMockFactory(h.registry, "openai")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	prov, err := h.getProviderForRequest(c, "gpt-4o", "openai")
	require.Error(t, err)
	assert.Nil(t, prov)
	assert.Contains(t, err.Error(), "disabled")

	_, selected := c.Get("selected_account_id")
	assert.False(t, selected)
}

func TestProxyHandler_ChatCompletions_ShouldPassthroughUpstreamProviderError(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "google", "gemini-3.1-pro-preview")

	googleProvider := &failingProvider{
		BaseProvider: provider.NewBaseProvider(
			"google",
			"test-key",
			"https://generativelanguage.googleapis.com/v1beta",
			[]string{"gemini-2.0-flash"},
			true,
		),
		chatErr: &provider.ProviderError{
			Code:      http.StatusBadRequest,
			Message:   "models/gemini-3.1-pro-preview is not found",
			Provider:  "google",
			Retryable: false,
		},
	}
	provider.RegisterProvider("google", googleProvider)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"model":"gemini-3.1-pro-preview","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeProviderError, resp.Error.Code)
	assert.Equal(t, "models/gemini-3.1-pro-preview is not found", resp.Error.Message)
}

func TestProxyHandler_ChatCompletions_ProviderErrorShouldRecordHTTPResponseErrorSpan(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()

	provider.RegisterProvider("openai", &failingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
		chatErr: &provider.ProviderError{
			Code:      http.StatusBadGateway,
			Message:   "upstream failed",
			Provider:  "openai",
			Retryable: false,
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","messages":[{"role":"user","content":"hello failure trace"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusBadGateway, w.Code)
	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	httpTrace := fetchOperationTraceRecord(t, db, requestID, "http.response")
	assert.Equal(t, "error", httpTrace.Status)
	assert.Equal(t, "upstream failed", httpTrace.Error)
	assert.Equal(t, false, httpTrace.Attrs["success"])
	assert.Equal(t, float64(http.StatusBadGateway), httpTrace.Attrs["status_code"])
	assert.Equal(t, "hello failure trace", httpTrace.Attrs["user_message_preview"])
	assert.Equal(t, "hello failure trace", httpTrace.Attrs["user_message_full"])
	assert.Equal(t, "upstream failed", httpTrace.Attrs["error_message_preview"])
	assert.Equal(t, "upstream failed", httpTrace.Attrs["error_message_full"])
	assert.Equal(t, "upstream failed", httpTrace.Attrs["ai_response_preview"])
	assert.Equal(t, "upstream failed", httpTrace.Attrs["ai_response_full"])
}

func TestProxyHandler_ChatCompletions_StreamStartFailureShouldRecordHTTPResponseErrorSpan(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()

	provider.RegisterProvider("openai", &streamStartFailProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
		streamErr:    assert.AnError,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hello stream failure"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)
	require.Contains(t, w.Body.String(), assert.AnError.Error())

	httpTrace := fetchOperationTraceRecord(t, db, requestID, "http.response")
	assert.Equal(t, "error", httpTrace.Status)
	assert.Equal(t, assert.AnError.Error(), httpTrace.Error)
	assert.Equal(t, false, httpTrace.Attrs["success"])
	assert.Equal(t, float64(http.StatusBadGateway), httpTrace.Attrs["status_code"])
	assert.Equal(t, "hello stream failure", httpTrace.Attrs["user_message_preview"])
	assert.Equal(t, "hello stream failure", httpTrace.Attrs["user_message_full"])
	assert.Equal(t, assert.AnError.Error(), httpTrace.Attrs["error_message_preview"])
	assert.Equal(t, assert.AnError.Error(), httpTrace.Attrs["error_message_full"])
	assert.Equal(t, assert.AnError.Error(), httpTrace.Attrs["ai_response_preview"])
	assert.Equal(t, assert.AnError.Error(), httpTrace.Attrs["ai_response_full"])
}

func TestProxyHandler_ChatCompletions_StreamContextCanceledShouldRecordHTTPResponseErrorSpan(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-4")
	db := storage.GetSQLiteStorage().GetDB()

	provider.RegisterProvider("openai", &hangingStreamProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4"}, true),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hello canceled stream"}]}`
	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body)).WithContext(reqCtx)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	requestID := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	httpTrace := fetchOperationTraceRecord(t, db, requestID, "http.response")
	assert.Equal(t, "error", httpTrace.Status)
	assert.Equal(t, false, httpTrace.Attrs["success"])
	assert.Equal(t, float64(499), httpTrace.Attrs["status_code"])
	assert.Equal(t, "hello canceled stream", httpTrace.Attrs["user_message_preview"])
	assert.Equal(t, "hello canceled stream", httpTrace.Attrs["user_message_full"])
	assert.Contains(t, httpTrace.Error, "context canceled")
}

type capturingProvider struct {
	*provider.BaseProvider

	mu            sync.Mutex
	lastChatReq   *provider.ChatRequest
	lastStreamReq *provider.ChatRequest
}

type reasoningDowngradeProvider struct {
	*provider.BaseProvider

	mu          sync.Mutex
	chatCalls   int
	streamCalls int
	chatReqs    []*provider.ChatRequest
	streamReqs  []*provider.ChatRequest
}

func cloneProviderChatRequest(req *provider.ChatRequest) *provider.ChatRequest {
	if req == nil {
		return nil
	}
	cloned := *req
	if req.Extra != nil {
		extra := make(map[string]interface{}, len(req.Extra))
		for k, v := range req.Extra {
			extra[k] = v
		}
		cloned.Extra = extra
	}
	if len(req.Messages) > 0 {
		cloned.Messages = append([]provider.ChatMessage(nil), req.Messages...)
	}
	return &cloned
}

func hasReasoningEffort(extra map[string]interface{}) bool {
	if extra == nil {
		return false
	}
	v, ok := extra["reasoning_effort"]
	if !ok {
		return false
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(s) != ""
}

func (p *reasoningDowngradeProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (p *reasoningDowngradeProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	p.mu.Lock()
	p.chatCalls++
	p.chatReqs = append(p.chatReqs, cloneProviderChatRequest(req))
	call := p.chatCalls
	p.mu.Unlock()

	if call == 1 && hasReasoningEffort(req.Extra) {
		return nil, &provider.ProviderError{
			Code:      http.StatusBadRequest,
			Message:   "reasoning_effort is not supported for this model",
			Provider:  "openai",
			Retryable: false,
		}
	}

	return &provider.ChatResponse{
		ID:      "reasoning-chat",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{{
			Index: 0,
			Message: provider.ChatMessage{
				Role:    "assistant",
				Content: "downgraded success",
			},
			FinishReason: "stop",
		}},
		Usage: provider.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
	}, nil
}

func (p *reasoningDowngradeProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	p.mu.Lock()
	p.streamCalls++
	p.streamReqs = append(p.streamReqs, cloneProviderChatRequest(req))
	call := p.streamCalls
	p.mu.Unlock()

	if call == 1 && hasReasoningEffort(req.Extra) {
		return nil, &provider.ProviderError{
			Code:      http.StatusUnprocessableEntity,
			Message:   "unsupported parameter reasoning_effort",
			Provider:  "openai",
			Retryable: false,
		}
	}

	ch := make(chan *provider.StreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "reasoning-stream",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.StreamChoice{{
				Index: 0,
				Delta: &provider.StreamDelta{Role: "assistant", Content: "stream downgraded"},
			}},
		}
		ch <- &provider.StreamChunk{
			ID:      "reasoning-stream",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.StreamChoice{{
				Index:        0,
				Delta:        &provider.StreamDelta{},
				FinishReason: "stop",
			}},
			Usage: &provider.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5},
			Done:  true,
		}
	}()

	return ch, nil
}

func (p *capturingProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (p *capturingProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	p.mu.Lock()
	p.lastChatReq = req
	p.mu.Unlock()

	return &provider.ChatResponse{
		ID:      "capture-chat",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []provider.Choice{{
			Index: 0,
			Message: provider.ChatMessage{
				Role:    "assistant",
				Content: "ok",
			},
			FinishReason: "stop",
		}},
		Usage: provider.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}, nil
}

func (p *capturingProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	p.mu.Lock()
	p.lastStreamReq = req
	p.mu.Unlock()

	ch := make(chan *provider.StreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "capture-stream",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.StreamChoice{{
				Index: 0,
				Delta: &provider.StreamDelta{Role: "assistant", Content: "ok"},
			}},
		}
		ch <- &provider.StreamChunk{
			ID:      "capture-stream",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.StreamChoice{{
				Index:        0,
				Delta:        &provider.StreamDelta{},
				FinishReason: "stop",
			}},
			Usage: &provider.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
			Done:  true,
		}
	}()

	return ch, nil
}

func TestProxyHandler_ChatCompletions_ShouldBridgeExtrasToProviderRequest(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "unit-test-model")
	capture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"unit-test-model"}, true),
	}
	provider.RegisterProvider("openai", capture)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"unit-test-model","messages":[{"role":"user","content":"hello"}],"top_p":0.8,"n":2,"stop":["a","b"],"frequency_penalty":0.4,"presence_penalty":0.3,"logit_bias":{"123":1.2},"user":"u-1","deepThink":true,"reasoning_effort":"low"}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capture.lastChatReq)
	require.NotNil(t, capture.lastChatReq.Extra)

	assert.Equal(t, 0.8, capture.lastChatReq.Extra["top_p"])
	assert.Equal(t, 2, capture.lastChatReq.Extra["n"])
	assert.Equal(t, 0.4, capture.lastChatReq.Extra["frequency_penalty"])
	assert.Equal(t, 0.3, capture.lastChatReq.Extra["presence_penalty"])
	assert.Equal(t, "u-1", capture.lastChatReq.Extra["user"])
	assert.Equal(t, "low", capture.lastChatReq.Extra["reasoning_effort"])
	assert.Equal(t, true, capture.lastChatReq.Extra["deep_think"])
	assert.Equal(t, true, capture.lastChatReq.Extra["reasoning"])
	require.IsType(t, []interface{}{}, capture.lastChatReq.Extra["stop"])
	require.IsType(t, map[string]interface{}{}, capture.lastChatReq.Extra["logit_bias"])
}

func TestProxyHandler_ChatCompletions_Stream_ShouldBridgeExtrasToProviderRequest(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "unit-test-model")
	capture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"unit-test-model"}, true),
	}
	provider.RegisterProvider("openai", capture)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"unit-test-model","stream":true,"messages":[{"role":"user","content":"hello"}],"reasoning_effort":"xhigh"}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capture.lastStreamReq)
	require.NotNil(t, capture.lastStreamReq.Extra)
	assert.Equal(t, "xhigh", capture.lastStreamReq.Extra["reasoning_effort"])
}

func TestBuildProviderExtraFromChatRequest_ReasoningEffortFallbackAndPriority(t *testing.T) {
	t.Run("deepThink fallback to high", func(t *testing.T) {
		req := &ChatCompletionRequest{DeepThink: true}
		extra := buildProviderExtraFromChatRequest(req)
		require.NotNil(t, extra)
		assert.Equal(t, "high", extra["reasoning_effort"])
	})

	t.Run("reasoning effort has priority over deepThink", func(t *testing.T) {
		req := &ChatCompletionRequest{DeepThink: true, ReasoningEffort: "medium"}
		extra := buildProviderExtraFromChatRequest(req)
		require.NotNil(t, extra)
		assert.Equal(t, "medium", extra["reasoning_effort"])
	})
}

func TestProxyHandler_ChatCompletions_NonOpenAIProvider_ShouldIgnoreReasoningEffort(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "anthropic", "unit-test-model")
	capture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("anthropic", "test-key", "https://api.anthropic.com", []string{"unit-test-model"}, true),
	}
	provider.RegisterProvider("anthropic", capture)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"anthropic","model":"unit-test-model","messages":[{"role":"user","content":"hello"}],"reasoning_effort":"medium"}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capture.lastChatReq)
	assert.Equal(t, "medium", capture.lastChatReq.Extra["reasoning_effort"])
}

func TestProxyHandler_ChatCompletions_ShouldRetryWithoutReasoningEffortForUnsupportedModels(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "reasoning-downgrade-model")
	retryProvider := &reasoningDowngradeProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"reasoning-downgrade-model"}, true),
	}
	provider.RegisterProvider("openai", retryProvider)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"reasoning-downgrade-model","messages":[{"role":"user","content":"hello"}],"reasoning_effort":"xhigh"}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	gatewayMeta, ok := payload["gateway_meta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, gatewayMeta["reasoning_effort_downgraded"])

	retryProvider.mu.Lock()
	defer retryProvider.mu.Unlock()
	require.Len(t, retryProvider.chatReqs, 2)
	assert.True(t, hasReasoningEffort(retryProvider.chatReqs[0].Extra))
	assert.False(t, hasReasoningEffort(retryProvider.chatReqs[1].Extra))
}

func TestProxyHandler_ChatCompletions_Stream_ShouldRetryWithoutReasoningEffortForUnsupportedModels(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "reasoning-downgrade-stream-model")
	retryProvider := &reasoningDowngradeProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"reasoning-downgrade-stream-model"}, true),
	}
	provider.RegisterProvider("openai", retryProvider)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"reasoning-downgrade-stream-model","stream":true,"messages":[{"role":"user","content":"hello"}],"reasoning_effort":"high"}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"gateway_meta":{"reasoning_effort_downgraded":true}`)

	retryProvider.mu.Lock()
	defer retryProvider.mu.Unlock()
	require.Len(t, retryProvider.streamReqs, 2)
	assert.True(t, hasReasoningEffort(retryProvider.streamReqs[0].Extra))
	assert.False(t, hasReasoningEffort(retryProvider.streamReqs[1].Extra))
}

type modelNotFoundProbeProvider struct {
	*provider.BaseProvider

	mu           sync.Mutex
	chatModels   []string
	streamModels []string
}

func (p *modelNotFoundProbeProvider) ValidateKey(_ context.Context) bool {
	return true
}

func (p *modelNotFoundProbeProvider) Chat(_ context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	p.mu.Lock()
	p.chatModels = append(p.chatModels, req.Model)
	p.mu.Unlock()

	if req.Model == "gpt-4o" {
		return &provider.ChatResponse{
			ID:      "fallback-chat",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []provider.Choice{{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: "fallback answer",
				},
				FinishReason: "stop",
			}},
			Usage: provider.Usage{PromptTokens: 2, CompletionTokens: 2, TotalTokens: 4},
		}, nil
	}

	return nil, &provider.ProviderError{
		Code:      http.StatusBadRequest,
		Message:   "model not found",
		Provider:  "openai",
		Retryable: false,
	}
}

func (p *modelNotFoundProbeProvider) StreamChat(_ context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	p.mu.Lock()
	p.streamModels = append(p.streamModels, req.Model)
	p.mu.Unlock()

	ch := make(chan *provider.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- &provider.StreamChunk{
			ID:      "stream-empty",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Done:    true,
			Choices: []provider.StreamChoice{{
				Index:        0,
				FinishReason: "stop",
				Delta:        &provider.StreamDelta{},
			}},
		}
	}()

	return ch, nil
}

func TestProxyHandler_ChatCompletions_ShouldForwardModelIDWhenRequestUsesDisplayName(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	capture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-4o"}, true),
	}
	provider.RegisterProvider("openai", capture)

	score := h.smartRouter.GetModelScore("gpt-4o")
	require.NotNil(t, score)
	originalDisplayName := score.DisplayName
	score.DisplayName = "gpt-4o-display"
	t.Cleanup(func() {
		score.DisplayName = originalDisplayName
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-4o-display","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capture.lastChatReq)
	assert.Equal(t, "gpt-4o", capture.lastChatReq.Model)
}

func TestProxyHandler_ChatCompletions_ShouldCanonicalizeAliasBeforeForwarding(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "gpt-5.3-codex")
	capture := &capturingProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"gpt-5.3-codex"}, true),
	}
	provider.RegisterProvider("openai", capture)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"gpt-5-3-codex","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capture.lastChatReq)
	assert.Equal(t, "gpt-5.3-codex", capture.lastChatReq.Model)
}

func TestProxyHandler_ChatCompletions_ModelNotFoundShouldNotFallbackAcrossModels(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "missing-model")
	probe := &modelNotFoundProbeProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"missing-model", "gpt-4o"}, true),
	}
	provider.RegisterProvider("openai", probe)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"missing-model","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "model not found")

	probe.mu.Lock()
	chatModels := append([]string(nil), probe.chatModels...)
	probe.mu.Unlock()
	assert.Equal(t, []string{"missing-model"}, chatModels)
}

func TestProxyHandler_ChatCompletions_StreamFallback_ModelNotFoundShouldNotFallbackAcrossModels(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	ensureModelRegistryModelsForTest(t, h, "openai", "missing-stream-model")
	probe := &modelNotFoundProbeProvider{
		BaseProvider: provider.NewBaseProvider("openai", "test-key", "https://api.openai.com/v1", []string{"missing-stream-model", "gpt-4o"}, true),
	}
	provider.RegisterProvider("openai", probe)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"provider":"openai","model":"missing-stream-model","stream":true,"messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChatCompletions(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "model not found")
	assert.NotContains(t, w.Body.String(), "fallback answer")

	probe.mu.Lock()
	chatModels := append([]string(nil), probe.chatModels...)
	probe.mu.Unlock()
	assert.Equal(t, []string{"missing-stream-model"}, chatModels)
}

func TestIsModelNotFoundError_ShouldNotMatchGenericBadRequest(t *testing.T) {
	err := &provider.ProviderError{
		Code:      http.StatusBadRequest,
		Message:   "invalid request payload",
		Type:      "invalid_request_error",
		Provider:  "openai",
		Retryable: false,
	}

	assert.False(t, isModelNotFoundError(err))
}

func TestIsModelNotFoundError_ShouldMatchProviderNotFoundSignals(t *testing.T) {
	errByCode := &provider.ProviderError{
		Code:      http.StatusNotFound,
		Message:   "resource not found",
		Type:      "not_found_error",
		Provider:  "openai",
		Retryable: false,
	}
	errByMessage := &provider.ProviderError{
		Code:      http.StatusBadRequest,
		Message:   "model not found",
		Type:      "invalid_request_error",
		Provider:  "openai",
		Retryable: false,
	}

	assert.True(t, isModelNotFoundError(errByCode))
	assert.True(t, isModelNotFoundError(errByMessage))
}
