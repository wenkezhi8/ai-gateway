package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderError_Error(t *testing.T) {
	err := &ProviderError{
		Code:      429,
		Message:   "Rate limit exceeded",
		Type:      "rate_limit",
		Provider:  "openai",
		Retryable: true,
	}

	assert.Equal(t, "Rate limit exceeded", err.Error())
}

func TestProviderError_Fields(t *testing.T) {
	err := &ProviderError{
		Code:      500,
		Message:   "Internal server error",
		Type:      "server_error",
		Provider:  "anthropic",
		Retryable: false,
	}

	assert.Equal(t, 500, err.Code)
	assert.Equal(t, "Internal server error", err.Message)
	assert.Equal(t, "server_error", err.Type)
	assert.Equal(t, "anthropic", err.Provider)
	assert.False(t, err.Retryable)
}

func TestChatRequest_FieldsWithExtra(t *testing.T) {
	req := ChatRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.8,
		MaxTokens:   2000,
		Stream:      false,
		Extra: map[string]interface{}{
			"presence_penalty": 0.5,
		},
		RawBody: map[string]interface{}{
			"custom_field": "value",
		},
	}

	assert.Equal(t, "gpt-4", req.Model)
	assert.Len(t, req.Messages, 2)
	assert.Equal(t, 0.8, req.Temperature)
	assert.Equal(t, 2000, req.MaxTokens)
	assert.False(t, req.Stream)
	assert.NotNil(t, req.Extra)
	assert.NotNil(t, req.RawBody)
}

func TestChatResponse_FieldsWithExtra(t *testing.T) {
	resp := ChatResponse{
		ID:      "resp-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "claude-3",
		Choices: []Choice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "end_turn",
			},
		},
		Usage: Usage{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      15,
		},
		Extra: map[string]any{
			"custom": "field",
		},
	}

	assert.Equal(t, "resp-123", resp.ID)
	assert.Equal(t, "claude-3", resp.Model)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestStreamChunk_Fields(t *testing.T) {
	chunk := StreamChunk{
		ID:      "chunk-123",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &StreamDelta{
					Role:    "assistant",
					Content: "Hello",
				},
				FinishReason: "",
			},
		},
		Done: false,
	}

	assert.Equal(t, "chunk-123", chunk.ID)
	assert.False(t, chunk.Done)
	assert.Len(t, chunk.Choices, 1)
	assert.Equal(t, "Hello", chunk.Choices[0].Delta.Content)
}

func TestProviderConfig_Fields(t *testing.T) {
	cfg := ProviderConfig{
		Name:    "openai",
		APIKey:  "sk-test",
		BaseURL: "https://api.openai.com",
		Models:  []string{"gpt-4", "gpt-3.5-turbo"},
		Enabled: true,
		Extra: map[string]any{
			"organization": "org-123",
		},
	}

	assert.Equal(t, "openai", cfg.Name)
	assert.Equal(t, "sk-test", cfg.APIKey)
	assert.Len(t, cfg.Models, 2)
	assert.True(t, cfg.Enabled)
}

func TestBaseProvider_Methods(t *testing.T) {
	bp := NewBaseProvider(
		"anthropic",
		"sk-ant-test",
		"https://api.anthropic.com",
		[]string{"claude-3", "claude-2"},
		true,
	)

	assert.Equal(t, "anthropic", bp.Name())
	assert.Equal(t, "sk-ant-test", bp.APIKey())
	assert.Equal(t, "https://api.anthropic.com", bp.BaseURL())
	assert.Len(t, bp.Models(), 2)
	assert.True(t, bp.IsEnabled())

	bp.SetEnabled(false)
	assert.False(t, bp.IsEnabled())

	bp.SetEnabled(true)
	assert.True(t, bp.IsEnabled())
}

func TestChatMessage_WithName(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello",
		Name:    "John",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello", msg.Content)
	assert.Equal(t, "John", msg.Name)
}

func TestChatResponse_WithError(t *testing.T) {
	resp := ChatResponse{
		ID:    "resp-error",
		Model: "gpt-4",
		Error: &ProviderError{
			Code:    429,
			Message: "Rate limit exceeded",
			Type:    "rate_limit",
		},
	}

	assert.NotNil(t, resp.Error)
	assert.Equal(t, 429, resp.Error.Code)
}

func TestStreamDelta_Fields(t *testing.T) {
	delta := StreamDelta{
		Role:    "assistant",
		Content: "World",
	}

	assert.Equal(t, "assistant", delta.Role)
	assert.Equal(t, "World", delta.Content)
}
