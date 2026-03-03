package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		request     ChatCompletionRequest
		expectError bool
		errorField  string
	}{
		{
			name: "valid request",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			expectError: false,
		},
		{
			name: "missing model",
			request: ChatCompletionRequest{
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			expectError: true,
			errorField:  "model",
		},
		{
			name: "missing messages",
			request: ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: []ChatMessage{},
			},
			expectError: true,
			errorField:  "messages",
		},
		{
			name: "missing role",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Content: "Hello"},
				},
			},
			expectError: true,
			errorField:  "messages",
		},
		// 改动点: 验证索引格式为 messages[1].role
		{
			name: "missing role on second message",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello"},
					{Content: "Missing role"},
				},
			},
			expectError: true,
			errorField:  "messages[1].role",
		},
		{
			name: "missing content for user",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Role: "user"},
				},
			},
			expectError: true,
			errorField:  "messages",
		},
		{
			name: "system message without content is valid",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Role: "system"},
				},
			},
			expectError: false,
		},
		{
			name: "multiple messages",
			request: ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi!"},
				},
			},
			expectError: false,
		},
		{
			name: "valid reasoning effort high",
			request: ChatCompletionRequest{
				Model:           "gpt-5.3-codex",
				ReasoningEffort: "high",
				Messages:        []ChatMessage{{Role: "user", Content: "Hello"}},
			},
			expectError: false,
		},
		{
			name: "valid reasoning effort with uppercase",
			request: ChatCompletionRequest{
				Model:           "gpt-5.3-codex",
				ReasoningEffort: "XHIGH",
				Messages:        []ChatMessage{{Role: "user", Content: "Hello"}},
			},
			expectError: false,
		},
		{
			name: "invalid reasoning effort",
			request: ChatCompletionRequest{
				Model:           "gpt-5.3-codex",
				ReasoningEffort: "ultra",
				Messages:        []ChatMessage{{Role: "user", Content: "Hello"}},
			},
			expectError: true,
			errorField:  "reasoning_effort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.expectError {
				require.Error(t, err)
				validationErr, ok := err.(*ValidationError)
				require.True(t, ok)
				assert.Contains(t, validationErr.Field, tt.errorField)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNormalizeReasoningEffort(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		expected  string
		isAllowed bool
	}{
		{name: "low", raw: "low", expected: "low", isAllowed: true},
		{name: "medium", raw: "medium", expected: "medium", isAllowed: true},
		{name: "high", raw: "high", expected: "high", isAllowed: true},
		{name: "xhigh uppercase", raw: "XHIGH", expected: "xhigh", isAllowed: true},
		{name: "trim spaces", raw: " high ", expected: "high", isAllowed: true},
		{name: "invalid", raw: "ultra", expected: "", isAllowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeReasoningEffort(tt.raw)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.isAllowed, ok)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "model",
		Message: "is required",
	}

	assert.Equal(t, "model: is required", err.Error())
}

func TestChatMessage_Fields(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello, world!",
		Name:    "John",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
	assert.Equal(t, "John", msg.Name)
}

func TestChatCompletionResponse_Fields(t *testing.T) {
	temp := 0.7
	maxTokens := 100

	req := ChatCompletionRequest{
		Model:       "gpt-4",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Stream:      true,
	}

	assert.Equal(t, "gpt-4", req.Model)
	assert.Equal(t, 0.7, *req.Temperature)
	assert.Equal(t, 100, *req.MaxTokens)
	assert.True(t, req.Stream)
}

func TestChoice_Fields(t *testing.T) {
	choice := Choice{
		Index: 0,
		Message: &ChatMessage{
			Role:    "assistant",
			Content: "Response",
		},
		FinishReason: "stop",
	}

	assert.Equal(t, 0, choice.Index)
	assert.NotNil(t, choice.Message)
	assert.Equal(t, "stop", choice.FinishReason)
}

func TestUsage_Fields(t *testing.T) {
	usage := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	assert.Equal(t, 10, usage.PromptTokens)
	assert.Equal(t, 20, usage.CompletionTokens)
	assert.Equal(t, 30, usage.TotalTokens)
}

func TestCompletionRequest_Fields(t *testing.T) {
	req := CompletionRequest{
		Model:  "gpt-3.5-turbo-instruct",
		Prompt: "Complete this sentence",
	}

	assert.Equal(t, "gpt-3.5-turbo-instruct", req.Model)
	assert.NotNil(t, req.Prompt)
}

func TestEmbeddingRequest_Fields(t *testing.T) {
	req := EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: []interface{}{"Hello", "World"},
		User:  "user123",
	}

	assert.Equal(t, "text-embedding-ada-002", req.Model)
	assert.Len(t, req.Input, 2)
	assert.Equal(t, "user123", req.User)
}

func TestEmbeddingData_Fields(t *testing.T) {
	data := EmbeddingData{
		Object:    "embedding",
		Index:     0,
		Embedding: []float64{0.1, 0.2, 0.3},
	}

	assert.Equal(t, "embedding", data.Object)
	assert.Equal(t, 0, data.Index)
	assert.Len(t, data.Embedding, 3)
}

func TestModelInfo_Fields(t *testing.T) {
	info := ModelInfo{
		ID:       "gpt-4",
		Object:   "model",
		OwnedBy:  "openai",
		Provider: "openai",
		Enabled:  true,
	}

	assert.Equal(t, "gpt-4", info.ID)
	assert.Equal(t, "model", info.Object)
	assert.Equal(t, "openai", info.OwnedBy)
	assert.Equal(t, "openai", info.Provider)
	assert.True(t, info.Enabled)
}

func TestModelListResponse_Fields(t *testing.T) {
	resp := ModelListResponse{
		Object: "list",
		Data: []ModelInfo{
			{ID: "gpt-4"},
			{ID: "gpt-3.5-turbo"},
		},
	}

	assert.Equal(t, "list", resp.Object)
	assert.Len(t, resp.Data, 2)
}

func TestStreamingResponse_Fields(t *testing.T) {
	finishReason := "stop"
	resp := StreamingResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &ChatMessage{
					Role:    "assistant",
					Content: "Hello",
				},
				FinishReason: &finishReason,
			},
		},
	}

	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, "chat.completion.chunk", resp.Object)
	assert.EqualValues(t, 1234567890, resp.Created)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "stop", *resp.Choices[0].FinishReason)
}
