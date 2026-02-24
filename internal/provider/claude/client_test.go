package claude

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
		Model: "claude-3-opus-20240229",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	claudeReq := ConvertRequest(req)

	assert.Equal(t, "claude-3-opus-20240229", claudeReq.Model)
	assert.Len(t, claudeReq.Messages, 1)
	assert.Equal(t, "user", claudeReq.Messages[0].Role)
	assert.True(t, claudeReq.Stream)
	assert.Equal(t, 0.7, claudeReq.Temperature)
	assert.Equal(t, 1000, claudeReq.MaxTokens)
}

func TestConvertRequest_WithSystemPrompt(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "claude-3-opus-20240229",
		Messages: []provider.ChatMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
	}

	claudeReq := ConvertRequest(req)

	assert.Equal(t, "You are helpful", claudeReq.System)
	assert.Len(t, claudeReq.Messages, 1)
}

func TestConvertRequest_WithTools(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "claude-3-opus-20240229",
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
		ToolChoice: map[string]string{"type": "auto"},
	}

	claudeReq := ConvertRequest(req)

	assert.Len(t, claudeReq.Tools, 1)
	assert.Equal(t, "test_func", claudeReq.Tools[0].Name)
	assert.NotNil(t, claudeReq.ToolChoice)
}

func TestConvertRequest_WithExtra(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "claude-3-opus-20240229",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Extra: map[string]interface{}{
			"top_p":   0.9,
			"top_k":   50,
			"user_id": "test-user",
		},
	}

	claudeReq := ConvertRequest(req)

	assert.Equal(t, 0.9, claudeReq.TopP)
	assert.Equal(t, 50, claudeReq.TopK)
	require.NotNil(t, claudeReq.Metadata)
	assert.Equal(t, "test-user", claudeReq.Metadata.UserID)
}

func TestConvertRequest_DefaultMaxTokens(t *testing.T) {
	req := &provider.ChatRequest{
		Model: "claude-3-opus-20240229",
		Messages: []provider.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	claudeReq := ConvertRequest(req)

	assert.Equal(t, 4096, claudeReq.MaxTokens)
}

func TestConvertToProviderResponse(t *testing.T) {
	resp := &MessagesResponse{
		ID:   "msg-test-id",
		Type: "message",
		Role: "assistant",
		Content: []ContentBlock{
			{Type: "text", Text: "Hello!"},
		},
		Model:      "claude-3-opus-20240229",
		StopReason: "end_turn",
		Usage: ResponseUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	provResp := ConvertToProviderResponse(resp)

	assert.Equal(t, "msg-test-id", provResp.ID)
	assert.Equal(t, "chat.completion", provResp.Object)
	assert.Equal(t, "claude-3-opus-20240229", provResp.Model)
	assert.Len(t, provResp.Choices, 1)
	assert.Equal(t, "Hello!", provResp.Choices[0].Message.Content)
	assert.Equal(t, 10, provResp.Usage.PromptTokens)
	assert.Equal(t, 20, provResp.Usage.CompletionTokens)
	assert.Equal(t, 30, provResp.Usage.TotalTokens)
}

func TestConvertToProviderResponse_WithError(t *testing.T) {
	resp := &MessagesResponse{
		ID:    "msg-test-id",
		Model: "claude-3-opus-20240229",
		Error: &ResponseError{
			Type:    "authentication_error",
			Message: "Invalid API key",
		},
	}

	provResp := ConvertToProviderResponse(resp)

	require.NotNil(t, provResp.Error)
	assert.Equal(t, "Invalid API key", provResp.Error.Message)
	assert.Equal(t, 401, provResp.Error.Code)
	assert.False(t, provResp.Error.Retryable)
}

func TestConvertToProviderResponse_WithToolCalls(t *testing.T) {
	resp := &MessagesResponse{
		ID:   "msg-test-id",
		Type: "message",
		Role: "assistant",
		Content: []ContentBlock{
			{
				Type:  "tool_use",
				ID:    "toolu-123",
				Name:  "test_func",
				Input: json.RawMessage(`{"arg": "value"}`),
			},
		},
		Model:      "claude-3-opus-20240229",
		StopReason: "tool_use",
	}

	provResp := ConvertToProviderResponse(resp)

	require.Len(t, provResp.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "toolu-123", provResp.Choices[0].Message.ToolCalls[0].ID)
	assert.Equal(t, "test_func", provResp.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestConvertStreamEvent_ContentBlockDelta(t *testing.T) {
	event := &StreamEvent{
		Type:  "content_block_delta",
		Index: 0,
		Delta: &StreamDelta{
			Type: "text_delta",
			Text: "Hello",
		},
	}

	chunk := ConvertStreamEvent(event, "claude-3-opus-20240229")

	require.NotNil(t, chunk)
	assert.Equal(t, "claude-3-opus-20240229", chunk.Model)
	assert.Len(t, chunk.Choices, 1)
	assert.Equal(t, "Hello", chunk.Choices[0].Delta.Content)
}

func TestConvertStreamEvent_MessageDelta(t *testing.T) {
	event := &StreamEvent{
		Type: "message_delta",
		Delta: &StreamDelta{
			StopReason: "end_turn",
		},
	}

	chunk := ConvertStreamEvent(event, "claude-3-opus-20240229")

	require.NotNil(t, chunk)
	assert.Equal(t, "end_turn", chunk.Choices[0].FinishReason)
	assert.True(t, chunk.Done)
}

func TestConvertStreamEvent_MessageStop(t *testing.T) {
	event := &StreamEvent{
		Type: "message_stop",
	}

	chunk := ConvertStreamEvent(event, "claude-3-opus-20240229")

	require.NotNil(t, chunk)
	assert.True(t, chunk.Done)
}

func TestConvertStreamEvent_Unknown(t *testing.T) {
	event := &StreamEvent{
		Type: "unknown_event",
	}

	chunk := ConvertStreamEvent(event, "claude-3-opus-20240229")

	assert.Nil(t, chunk)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{500, true},
		{502, true},
		{503, true},
		{529, true},
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

func TestConvertRole(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"assistant", "assistant"},
		{"user", "user"},
		{"system", "user"},
		{"function", "user"},
		{"tool", "user"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := convertRole(tt.input)
		assert.Equal(t, tt.expected, result, "input: %s", tt.input)
	}
}

func TestConvertContentToBlocks_String(t *testing.T) {
	blocks := convertContentToBlocks("Hello")

	require.Len(t, blocks, 1)
	assert.Equal(t, "text", blocks[0].Type)
	assert.Equal(t, "Hello", blocks[0].Text)
}

func TestConvertContentToBlocks_EmptyString(t *testing.T) {
	blocks := convertContentToBlocks("")

	assert.Nil(t, blocks)
}

func TestConvertContentToBlocks_Array(t *testing.T) {
	content := []interface{}{
		map[string]interface{}{"type": "text", "text": "Hello"},
		map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": "https://example.com/image.png",
			},
		},
	}

	blocks := convertContentToBlocks(content)

	require.Len(t, blocks, 2)
	assert.Equal(t, "text", blocks[0].Type)
	assert.Equal(t, "Hello", blocks[0].Text)
	assert.Equal(t, "image", blocks[1].Type)
	require.NotNil(t, blocks[1].Source)
	assert.Equal(t, "url", blocks[1].Source.Type)
	assert.Equal(t, "https://example.com/image.png", blocks[1].Source.URL)
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
		{"overloaded_error", 529},
	}

	for _, tt := range tests {
		resp := &MessagesResponse{
			Error: &ResponseError{
				Type:    tt.errorType,
				Message: "Test error",
			},
		}

		provResp := ConvertToProviderResponse(resp)
		require.NotNil(t, provResp.Error, "error type: %s", tt.errorType)
		assert.Equal(t, tt.expectedCode, provResp.Error.Code, "error type: %s", tt.errorType)
	}
}

func TestMessagesRequest_Marshal(t *testing.T) {
	req := &MessagesRequest{
		Model: "claude-3-opus-20240229",
		Messages: []Message{
			{Role: "user", Content: []ContentBlock{{Type: "text", Text: "Hello"}}},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		Stream:      true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), "claude-3-opus-20240229")
}
