package ernie

import (
	"context"
	"testing"

	"ai-gateway/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key", "test-secret-key", "")
	require.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "test-secret-key", client.secretKey)
	assert.Equal(t, defaultBaseURL, client.baseURL)
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	client := NewClient("test-api-key", "test-secret-key", "https://custom.api.com")
	require.NotNil(t, client)
	assert.Equal(t, "https://custom.api.com", client.baseURL)
}

func TestNewAdapter(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		BaseURL: "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop",
		Models:  []string{"ernie-4.0"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "ernie", adapter.Name())
	assert.True(t, adapter.IsEnabled())
}

func TestNewAdapter_DefaultBaseURL(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Models:  []string{"ernie-4.0"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, defaultBaseURL, adapter.BaseURL())
}

func TestNewAdapter_WithSecretKey(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Enabled: true,
		Extra: map[string]interface{}{
			"secret_key": "my-secret-key",
		},
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "my-secret-key", adapter.secretKey)
}

func TestAdapter_Name(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)
	assert.Equal(t, "ernie", adapter.Name())
}

func TestDefaultModels(t *testing.T) {
	models := DefaultModels()

	assert.NotEmpty(t, models)
	assert.Contains(t, models, "ernie-4.0")
	assert.Contains(t, models, "ernie-3.5")
	assert.Contains(t, models, "ernie-speed")
}

func TestFactory(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Enabled: true,
	}

	prov := Factory(cfg)
	require.NotNil(t, prov)

	_, ok := prov.(*Adapter)
	assert.True(t, ok)
}

func TestFactory_DefaultModels(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Models:  []string{},
		Enabled: true,
	}

	prov := Factory(cfg)
	require.NotNil(t, prov)

	assert.NotEmpty(t, prov.Models())
}

func TestAdapter_Chat_Disabled(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "ernie-4.0",
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	}

	resp, err := adapter.Chat(context.Background(), req)
	assert.Nil(t, resp)
	require.Error(t, err)

	provErr, ok := err.(*provider.ProviderError)
	require.True(t, ok)
	assert.Equal(t, 503, provErr.Code)
	assert.Contains(t, provErr.Message, "disabled")
}

func TestAdapter_StreamChat_Disabled(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "ernie",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "ernie-4.0",
		Messages: []provider.ChatMessage{{Role: "user", Content: "Hello"}},
	}

	ch, err := adapter.StreamChat(context.Background(), req)
	require.Error(t, err)
	assert.NotNil(t, ch)

	provErr, ok := err.(*provider.ProviderError)
	require.True(t, ok)
	assert.Equal(t, 503, provErr.Code)
	assert.Contains(t, provErr.Message, "disabled")
}

func TestConvertRequest(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "ernie-4.0",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	ernieReq := ConvertRequest(req)

	assert.Len(t, ernieReq.Messages, 1)
	assert.Equal(t, "user", ernieReq.Messages[0].Role)
	assert.True(t, ernieReq.Stream)
	assert.Equal(t, 0.7, ernieReq.Temperature)
}

func TestConvertRequest_WithFunctions(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "ernie-4.0",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Tools: []provider.Tool{
			{
				Type: "function",
				Function: provider.Function{
					Name:        "test_func",
					Description: "Test function",
					Parameters:  map[string]interface{}{"type": "object"},
				},
			},
		},
	}

	ernieReq := ConvertRequest(req)

	assert.Len(t, ernieReq.Functions, 1)
	assert.Equal(t, "test_func", ernieReq.Functions[0].Name)
}

func TestConvertResponse(t *testing.T) {
	content := "Hello!"
	resp := &ChatResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Result:  content,
		Usage: ChatResponseUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	provResp := ConvertResponse(resp, "ernie-4.0")

	assert.Equal(t, "ernie-4.0", provResp.Model)
	assert.Len(t, provResp.Choices, 1)
	assert.Equal(t, content, provResp.Choices[0].Message.Content)
	assert.Equal(t, 10, provResp.Usage.PromptTokens)
}

func TestConvertResponse_WithError(t *testing.T) {
	resp := &ChatResponse{
		ID:        "test-id",
		ErrorMsg:  "Invalid API key",
		ErrorCode: 401,
	}

	provResp := ConvertResponse(resp, "ernie-4.0")

	require.NotNil(t, provResp.Error)
	assert.Contains(t, provResp.Error.Message, "Invalid API key")
}

func TestConvertResponse_WithFunctionCall(t *testing.T) {
	resp := &ChatResponse{
		Result: "",
		FunctionCall: &FunctionCall{
			Name:      "test_func",
			Arguments: `{"arg": "value"}`,
		},
	}

	provResp := ConvertResponse(resp, "ernie-4.0")

	assert.Len(t, provResp.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "test_func", provResp.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestGetModelPath(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"ernie-4.0-8k", "/completions_pro"},
		{"ernie-4.0", "/completions_pro"},
		{"ernie-3.5-8k", "/completions"},
		{"ernie-3.5", "/completions"},
		{"ernie-speed-8k", "/ernie_speed"},
		{"ernie-speed", "/ernie_speed"},
		{"ernie-lite-8k", "/ernie_lite"},
		{"ernie-lite", "/ernie_lite"},
		{"ernie-tiny-8k", "/ernie_tiny"},
		{"unknown-model", "/completions"},
	}

	for _, tt := range tests {
		result := getModelPath(tt.model)
		assert.Equal(t, tt.expected, result, "model: %s", tt.model)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{500, true},
		{502, true},
		{503, true},
		{429, true},
		{408, true},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{200, false},
	}

	for _, tt := range tests {
		result := isRetryableError(tt.statusCode)
		assert.Equal(t, tt.expected, result, "statusCode: %d", tt.statusCode)
	}
}
