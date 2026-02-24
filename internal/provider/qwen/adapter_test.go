package qwen

import (
	"context"
	"testing"

	"ai-gateway/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key", "")
	require.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, defaultBaseURL, client.baseURL)
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	client := NewClient("test-api-key", "https://custom.api.com")
	require.NotNil(t, client)
	assert.Equal(t, "https://custom.api.com", client.baseURL)
}

func TestNewAdapter(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "qwen",
		APIKey:  "test-key",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Models:  []string{"qwen-max"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "qwen", adapter.Name())
	assert.True(t, adapter.IsEnabled())
}

func TestNewAdapter_DefaultBaseURL(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "qwen",
		APIKey:  "test-key",
		Models:  []string{"qwen-max"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, defaultBaseURL, adapter.BaseURL())
}

func TestAdapter_Name(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "qwen",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)
	assert.Equal(t, "qwen", adapter.Name())
}

func TestDefaultModels(t *testing.T) {
	models := DefaultModels()

	assert.NotEmpty(t, models)
	assert.Contains(t, models, "qwen-max")
	assert.Contains(t, models, "qwen-plus")
	assert.Contains(t, models, "qwen-turbo")
}

func TestFactory(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "qwen",
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
		Name:    "qwen",
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
		Name:    "qwen",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "qwen-max",
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
		Name:    "qwen",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "qwen-max",
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
		Model: "qwen-max",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	qwenReq := ConvertRequest(req)

	assert.Equal(t, "qwen-max", qwenReq.Model)
	assert.Len(t, qwenReq.Messages, 1)
	assert.Equal(t, "user", qwenReq.Messages[0].Role)
	assert.True(t, qwenReq.Stream)
	assert.Equal(t, 0.7, qwenReq.Temperature)
	assert.Equal(t, 1000, qwenReq.MaxTokens)
}

func TestConvertRequest_WithTools(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "qwen-max",
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
		ToolChoice: "auto",
	}

	qwenReq := ConvertRequest(req)

	assert.Len(t, qwenReq.Tools, 1)
	assert.Equal(t, "test_func", qwenReq.Tools[0].Function.Name)
	assert.Equal(t, "auto", qwenReq.ToolChoice)
}

func TestConvertResponse(t *testing.T) {
	content := "Hello!"
	resp := &ChatResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "qwen-max",
		Choices: []ChatResponseChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: ChatResponseUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	provResp := ConvertResponse(resp)

	assert.Equal(t, "test-id", provResp.ID)
	assert.Equal(t, "chat.completion", provResp.Object)
	assert.Equal(t, "qwen-max", provResp.Model)
	assert.Len(t, provResp.Choices, 1)
	assert.Equal(t, 10, provResp.Usage.PromptTokens)
}

func TestConvertResponse_WithError(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "qwen-max",
		Error: &ChatResponseError{
			Code:    "invalid_api_key",
			Message: "Invalid API key",
		},
	}

	provResp := ConvertResponse(resp)

	require.NotNil(t, provResp.Error)
	assert.Equal(t, "Invalid API key", provResp.Error.Message)
}

func TestConvertResponse_WithToolCalls(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "qwen-max",
		Choices: []ChatResponseChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: FunctionCall{
								Name:      "test_func",
								Arguments: `{"arg": "value"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}

	provResp := ConvertResponse(resp)

	assert.Len(t, provResp.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "call-1", provResp.Choices[0].Message.ToolCalls[0].ID)
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
