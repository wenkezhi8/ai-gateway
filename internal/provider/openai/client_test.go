package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
			"reasoning_effort":      "xhigh",
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
	assert.Equal(t, "xhigh", openaiReq.ReasoningEffort)
	assert.Equal(t, 42, openaiReq.Seed)
	assert.True(t, openaiReq.Logprobs)
	assert.Equal(t, 5, openaiReq.TopLogprobs)
	assert.Equal(t, 2000, openaiReq.MaxCompletionTokens)
}

func TestCallResponsesAPI_ShouldCarryReasoningEffort(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","model":"gpt-5.3-codex","output_text":"ok","usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	adapter := NewAdapter(&provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "test-key",
		BaseURL: server.URL,
		Models:  []string{"gpt-5.3-codex"},
		Enabled: true,
	})

	_, err := adapter.callResponsesAPI(context.Background(), &provider.ChatRequest{
		Model:    "gpt-5.3-codex",
		Messages: []provider.ChatMessage{{Role: "user", Content: "hello"}},
		Extra: map[string]interface{}{
			"reasoning_effort": "high",
		},
	}, false)
	require.NoError(t, err)

	require.NotNil(t, capturedBody)
	assert.Equal(t, "high", capturedBody["reasoning_effort"])
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

func TestShouldUseResponsesFallback(t *testing.T) {
	assert.True(t, shouldUseResponsesFallback(400, []byte(`{"error":{"message":"Unsupported legacy protocol: /v1/chat/completions is not supported. Please use /v1/responses"}}`)))
	assert.False(t, shouldUseResponsesFallback(500, []byte(`{"error":{"message":"internal error"}}`)))
	assert.False(t, shouldUseResponsesFallback(200, []byte(`ok`)))
}

func TestExtractResponsesTextAndUsage(t *testing.T) {
	resp := map[string]interface{}{
		"id":    "resp_test",
		"model": "gpt-5",
		"output": []interface{}{
			map[string]interface{}{
				"type": "message",
				"content": []interface{}{
					map[string]interface{}{"type": "output_text", "text": "hello"},
					map[string]interface{}{"type": "output_text", "text": " world"},
				},
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  12.0,
			"output_tokens": 8.0,
			"total_tokens":  20.0,
		},
	}

	assert.Equal(t, "hello world", extractResponsesText(resp))
	usage := extractResponsesUsage(resp)
	assert.Equal(t, 12, usage.PromptTokens)
	assert.Equal(t, 8, usage.CompletionTokens)
	assert.Equal(t, 20, usage.TotalTokens)
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

func TestNewAdapter(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
		Models:  []string{"gpt-4"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "openai", adapter.Name())
	assert.True(t, adapter.IsEnabled())
}

func TestNewAdapter_DefaultBaseURL(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "test-key",
		Models:  []string{"gpt-4"},
		Enabled: true,
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, defaultBaseURL, adapter.BaseURL())
}

func TestNewAdapter_WithOrgID(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "test-key",
		Enabled: true,
		Extra: map[string]interface{}{
			"organization_id": "org-123",
		},
	}

	adapter := NewAdapter(cfg)
	require.NotNil(t, adapter)
	assert.Equal(t, "org-123", adapter.orgID)
}

func TestAdapter_Name(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)
	assert.Equal(t, "openai", adapter.Name())
}

func TestAdapter_SetClient(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "openai",
		APIKey:  "test-key",
		Enabled: true,
	}
	adapter := NewAdapter(cfg)

	newClient := NewClient("new-key", "https://new.url.com", "")
	adapter.SetClient(newClient)

	assert.NotNil(t, adapter.client)
}

func TestDefaultModels(t *testing.T) {
	models := DefaultModels()

	assert.NotEmpty(t, models)
	assert.Contains(t, models, "gpt-4o")
	assert.Contains(t, models, "gpt-4")
	assert.Contains(t, models, "gpt-3.5-turbo")
}

func TestFactory(t *testing.T) {
	cfg := &provider.ProviderConfig{
		Name:    "openai",
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
		Name:    "openai",
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
		Name:    "openai",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "gpt-4",
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
		Name:    "openai",
		APIKey:  "test-key",
		Enabled: false,
	}
	adapter := NewAdapter(cfg)

	req := &provider.ChatRequest{
		Model:    "gpt-4",
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
