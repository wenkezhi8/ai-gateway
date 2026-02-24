package volcengine

import (
	"context"
	"encoding/json"
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

func TestConvertRequest(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "doubao-pro-4k",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1000,
		ToolChoice:  "auto",
	}

	volcReq := ConvertRequest(req)

	assert.Equal(t, "doubao-pro-4k", volcReq.Model)
	assert.Len(t, volcReq.Messages, 1)
	assert.Equal(t, "user", volcReq.Messages[0].Role)
	assert.True(t, volcReq.Stream)
	assert.Equal(t, 0.7, volcReq.Temperature)
	assert.Equal(t, 1000, volcReq.MaxTokens)
}

func TestConvertRequest_WithTools(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "doubao-pro-4k",
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

	volcReq := ConvertRequest(req)

	assert.Len(t, volcReq.Tools, 1)
	assert.Equal(t, "test_func", volcReq.Tools[0].Function.Name)
	assert.Equal(t, "auto", volcReq.ToolChoice)
}

func TestConvertRequest_WithExtra(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "doubao-pro-4k",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Extra: map[string]interface{}{
			"top_p":             0.9,
			"frequency_penalty": 0.5,
			"presence_penalty":  0.3,
			"user":              "test-user",
			"n":                 2,
			"logprobs":          true,
			"top_logprobs":      5,
		},
	}

	volcReq := ConvertRequest(req)

	assert.Equal(t, 0.9, volcReq.TopP)
	assert.Equal(t, 0.5, volcReq.FrequencyPenalty)
	assert.Equal(t, 0.3, volcReq.PresencePenalty)
	assert.Equal(t, "test-user", volcReq.User)
	assert.Equal(t, 2, volcReq.N)
	assert.True(t, volcReq.Logprobs)
	assert.Equal(t, 5, volcReq.TopLogprobs)
}

func TestConvertResponse(t *testing.T) {
	resp := &ChatResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "doubao-pro-4k",
		Choices: []ChatResponseChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Hello!",
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
	assert.Equal(t, "doubao-pro-4k", provResp.Model)
	assert.Len(t, provResp.Choices, 1)
	assert.Equal(t, 10, provResp.Usage.PromptTokens)
}

func TestConvertResponse_WithError(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "doubao-pro-4k",
		Error: &ChatResponseError{
			Code:    401,
			Message: "Invalid API key",
			Type:    "authentication_error",
		},
	}

	provResp := ConvertResponse(resp)

	require.NotNil(t, provResp.Error)
	assert.Equal(t, "Invalid API key", provResp.Error.Message)
	assert.Equal(t, 401, provResp.Error.Code)
	assert.False(t, provResp.Error.Retryable)
}

func TestConvertResponse_WithToolCalls(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "doubao-pro-4k",
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

func TestConvertStreamChunk(t *testing.T) {
	chunk := &StreamResponse{
		ID:      "test-id",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "doubao-pro-4k",
		Choices: []StreamResponseChoice{
			{
				Index: 0,
				Delta: &StreamDelta{
					Role:    "assistant",
					Content: "Hello",
				},
				FinishReason: "",
			},
		},
	}

	provChunk := ConvertStreamChunk(chunk, false)

	assert.Equal(t, "test-id", provChunk.ID)
	assert.Len(t, provChunk.Choices, 1)
	assert.Equal(t, "Hello", provChunk.Choices[0].Delta.Content)
	assert.False(t, provChunk.Done)
}

func TestConvertStreamChunk_WithDone(t *testing.T) {
	chunk := &StreamResponse{
		ID:      "test-id",
		Model:   "doubao-pro-4k",
		Choices: []StreamResponseChoice{},
	}

	provChunk := ConvertStreamChunk(chunk, true)

	assert.True(t, provChunk.Done)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
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
		result := isRetryableError(tt.code)
		assert.Equal(t, tt.expected, result, "code: %d", tt.code)
	}
}

func TestChatRequest_Marshal(t *testing.T) {
	req := &ChatRequest{
		Model: "doubao-pro-4k",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		Stream:      true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), "doubao-pro-4k")
}

func TestNewAdapter(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "volcengine",
		APIKey:  "test-key",
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		Models:  []string{"doubao-pro-32k"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "volcengine", adapter.Name())
	assert.True(t, adapter.IsEnabled())
}

func TestNewAdapter_DefaultBaseURL(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "volcengine",
		APIKey:  "test-key",
		Models:  []string{"doubao-pro-32k"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, defaultBaseURL, adapter.BaseURL())
}

func TestAdapter_Name(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "volcengine",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)
	assert.Equal(t, "volcengine", adapter.Name())
}

func TestAdapter_SetClient(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "volcengine",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)

	newClient := NewClient("new-key", "https://new.url.com")
	adapter.SetClient(newClient)

	assert.NotNil(t, adapter.client)
}

func TestDefaultModels(t *testing.T) {
	models := DefaultModels()

	assert.NotEmpty(t, models)
	assert.Contains(t, models, "doubao-pro-32k")
	assert.Contains(t, models, "doubao-lite-128k")
	assert.Contains(t, models, "skylark2-pro-4k")
}

func TestFactory(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "volcengine",
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
		Name:    "volcengine",
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
		Name:    "volcengine",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "doubao-pro-32k",
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
		Name:    "volcengine",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "doubao-pro-32k",
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
