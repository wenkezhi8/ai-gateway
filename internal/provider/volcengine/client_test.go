package volcengine

import (
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
