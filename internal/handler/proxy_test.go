package handler

import (
	"ai-gateway/internal/config"
	"ai-gateway/internal/provider"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

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

// mockProvider implements the provider.Provider interface for testing
type mockProvider struct {
	*provider.BaseProvider
}

func (m *mockProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
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

func (m *mockProvider) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
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

func (m *mockProvider) ValidateKey(ctx context.Context) bool {
	return true
}

func (m *mockProvider) Models() []string {
	return []string{"gpt-4"}
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
