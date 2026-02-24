package zhipu

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
		Name:    "zhipu",
		APIKey:  "test-key",
		BaseURL: "https://open.bigmodel.cn/api/paas/v4",
		Models:  []string{"glm-4"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "zhipu", adapter.Name())
	assert.True(t, adapter.IsEnabled())
}

func TestNewAdapter_DefaultBaseURL(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "zhipu",
		APIKey:  "test-key",
		Models:  []string{"glm-4"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, defaultBaseURL, adapter.BaseURL())
}

func TestAdapter_Name(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "zhipu",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)
	assert.Equal(t, "zhipu", adapter.Name())
}

func TestDefaultModels(t *testing.T) {
	models := DefaultModels()

	assert.NotEmpty(t, models)
	assert.Contains(t, models, "glm-4")
	assert.Contains(t, models, "glm-4-plus")
	assert.Contains(t, models, "glm-3-turbo")
}

func TestFactory(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "zhipu",
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
		Name:    "zhipu",
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
		Name:    "zhipu",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "glm-4",
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
		Name:    "zhipu",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "glm-4",
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
		Model: "glm-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	zhipuReq := ConvertRequest(req)

	assert.Equal(t, "glm-4", zhipuReq.Model)
	assert.Len(t, zhipuReq.Messages, 1)
	assert.Equal(t, "user", zhipuReq.Messages[0].Role)
	assert.True(t, zhipuReq.Stream)
	assert.Equal(t, 0.7, zhipuReq.Temperature)
	assert.Equal(t, 1000, zhipuReq.MaxTokens)
}

func TestConvertRequest_WithTools(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "glm-4",
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

	zhipuReq := ConvertRequest(req)

	assert.Len(t, zhipuReq.Tools, 1)
	assert.Equal(t, "test_func", zhipuReq.Tools[0].Function.Name)
	assert.Equal(t, "auto", zhipuReq.ToolChoice)
}

func TestConvertResponse(t *testing.T) {
	content := "Hello!"
	resp := &ChatResponse{
		ID:      "test-id",
		Created: 1234567890,
		Model:   "glm-4",
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
	assert.Equal(t, "glm-4", provResp.Model)
	assert.Len(t, provResp.Choices, 1)
	assert.Equal(t, 10, provResp.Usage.PromptTokens)
}

func TestConvertResponse_WithError(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "glm-4",
		Error: &ChatResponseError{
			Code:    401,
			Message: "Invalid API key",
		},
	}

	provResp := ConvertResponse(resp)

	require.NotNil(t, provResp.Error)
	assert.Equal(t, "Invalid API key", provResp.Error.Message)
	assert.Equal(t, 401, provResp.Error.Code)
}

func TestConvertResponse_WithToolCalls(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "glm-4",
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
