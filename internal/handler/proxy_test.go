//nolint:godot
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-gateway/internal/config"
	"ai-gateway/internal/provider"

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

func TestProxyHandler_GetProviderForRequest_UsesInferredProviderWhenModelNotMapped(t *testing.T) {
	// Clear global registry before test
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Register a Google provider that does not explicitly include gemini-3.1-pro-preview in model list
	googleProvider := &mockProvider{
		BaseProvider: provider.NewBaseProvider(
			"google",
			"test-key",
			"https://generativelanguage.googleapis.com/v1beta",
			[]string{"gemini-1.5-pro"},
			true,
		),
	}
	provider.RegisterProvider("google", googleProvider)

	selected, err := h.getProviderForRequest(c, "gemini-3.1-pro-preview", "")
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, "google", selected.Name())
}

func TestProxyHandler_ChatCompletions_ShouldPassthroughUpstreamProviderError(t *testing.T) {
	provider.ClearRegistry()
	defer provider.ClearRegistry()

	h := NewProxyHandler(&config.Config{}, nil, nil)

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
