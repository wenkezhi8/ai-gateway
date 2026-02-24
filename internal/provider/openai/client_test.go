package openai

import (
	"encoding/json"
	"testing"

	"ai-gateway/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key", "", "")
	require.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, defaultBaseURL, client.baseURL)
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	client := NewClient("test-api-key", "https://custom.api.com", "org-123")
	require.NotNil(t, client)
	assert.Equal(t, "https://custom.api.com", client.baseURL)
	assert.Equal(t, "org-123", client.orgID)
}

func TestConvertRequest(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "gpt-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1000,
		ToolChoice:  "auto",
	}

	openaiReq := ConvertRequest(req)

	assert.Equal(t, "gpt-4", openaiReq.Model)
	assert.Len(t, openaiReq.Messages, 1)
	assert.Equal(t, "user", openaiReq.Messages[0].Role)
	assert.True(t, openaiReq.Stream)
	assert.Equal(t, 0.7, openaiReq.Temperature)
	assert.Equal(t, 1000, openaiReq.MaxTokens)
	assert.NotNil(t, openaiReq.StreamOptions)
	assert.True(t, openaiReq.StreamOptions.IncludeUsage)
}

func TestConvertRequest_WithTools(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "gpt-4",
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

	openaiReq := ConvertRequest(req)

	assert.Len(t, openaiReq.Tools, 1)
	assert.Equal(t, "test_func", openaiReq.Tools[0].Function.Name)
	assert.Equal(t, "auto", openaiReq.ToolChoice)
}

func TestConvertRequest_WithExtra(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "gpt-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Extra: map[string]interface{}{
			"top_p":                 0.9,
			"frequency_penalty":     0.5,
			"presence_penalty":      0.3,
			"user":                  "test-user",
			"seed":                  42,
			"logprobs":              true,
			"top_logprobs":          5,
			"max_completion_tokens": 2000,
		},
	}

	openaiReq := ConvertRequest(req)

	assert.Equal(t, 0.9, openaiReq.TopP)
	assert.Equal(t, 0.5, openaiReq.FrequencyPenalty)
	assert.Equal(t, 0.3, openaiReq.PresencePenalty)
	assert.Equal(t, "test-user", openaiReq.User)
	assert.Equal(t, 42, openaiReq.Seed)
	assert.True(t, openaiReq.Logprobs)
	assert.Equal(t, 5, openaiReq.TopLogprobs)
	assert.Equal(t, 2000, openaiReq.MaxCompletionTokens)
}

func TestConvertRequest_WithLogitBias(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "gpt-4",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Extra: map[string]interface{}{
			"logit_bias": map[string]interface{}{
				"1234": 5.0,
				"5678": -5.0,
			},
		},
	}

	openaiReq := ConvertRequest(req)

	require.NotNil(t, openaiReq.LogitBias)
	assert.Equal(t, 5.0, openaiReq.LogitBias["1234"])
	assert.Equal(t, -5.0, openaiReq.LogitBias["5678"])
}

func TestConvertResponse(t *testing.T) {
	content := "Hello!"
	resp := &ChatResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
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
	assert.Equal(t, "gpt-4", provResp.Model)
	assert.Len(t, provResp.Choices, 1)
	assert.Equal(t, 10, provResp.Usage.PromptTokens)
}

func TestConvertResponse_WithError(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-4",
		Error: &ChatResponseError{
			Code:    "invalid_api_key",
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
		Model: "gpt-4",
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
		Model:   "gpt-4",
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

func TestConvertStreamChunk_WithUsage(t *testing.T) {
	usage := &ChatResponseUsage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	chunk := &StreamResponse{
		ID:      "test-id",
		Model:   "gpt-4",
		Choices: []StreamResponseChoice{},
		Usage:   usage,
	}

	provChunk := ConvertStreamChunk(chunk, true)

	require.NotNil(t, provChunk.Usage)
	assert.Equal(t, 30, provChunk.Usage.TotalTokens)
	assert.True(t, provChunk.Done)
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

func TestChatRequest_Marshal(t *testing.T) {
	req := &ChatRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		Stream:      true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), "gpt-4")
}

func TestErrorTypeToStatusCode(t *testing.T) {
	tests := []struct {
		errorType    string
		expectedCode int
	}{
		{"invalid_request_error", 400},
		{"authentication_error", 401},
		{"permission_error", 403},
		{"not_found_error", 404},
		{"rate_limit_error", 429},
		{"server_error", 500},
	}

	for _, tt := range tests {
		resp := &ChatResponse{
			Error: &ChatResponseError{
				Type:    tt.errorType,
				Message: "Test error",
			},
		}

		provResp := ConvertResponse(resp)
		require.NotNil(t, provResp.Error, "error type: %s", tt.errorType)
		assert.Equal(t, tt.expectedCode, provResp.Error.Code, "error type: %s", tt.errorType)
	}
}

func TestConvertResponse_WithArrayContent(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-4",
		Choices: []ChatResponseChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role: "assistant",
					Content: []interface{}{
						map[string]interface{}{"type": "text", "text": "Hello"},
						map[string]interface{}{"type": "text", "text": " World"},
					},
				},
				FinishReason: "stop",
			},
		},
	}

	provResp := ConvertResponse(resp)

	assert.Equal(t, "Hello World", provResp.Choices[0].Message.Content)
}
