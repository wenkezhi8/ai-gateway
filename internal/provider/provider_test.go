package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockProviderForTest implements Provider interface for testing
type mockProviderForTest struct {
	*BaseProvider
}

func (m *mockProviderForTest) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		ID:    "test-response",
		Model: req.Model,
	}, nil
}

func (m *mockProviderForTest) StreamChat(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	ch := make(chan *StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockProviderForTest) ValidateKey(ctx context.Context) bool {
	return true
}

func TestBaseProvider_New(t *testing.T) {
	bp := NewBaseProvider("openai", "test-api-key", "https://api.openai.com", []string{"gpt-4", "gpt-3.5-turbo"}, true)

	assert.Equal(t, "openai", bp.Name())
	assert.Equal(t, "https://api.openai.com", bp.BaseURL())
	assert.Equal(t, "test-api-key", bp.APIKey())
	assert.True(t, bp.IsEnabled())
	assert.Len(t, bp.Models(), 2)
}

func TestBaseProvider_Disabled(t *testing.T) {
	bp := NewBaseProvider("openai", "test-api-key", "https://api.openai.com", nil, false)
	assert.False(t, bp.IsEnabled())
}

func TestChatRequest_Fields(t *testing.T) {
	req := ChatRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
		Stream:      false,
		Extra: map[string]interface{}{
			"top_p": 0.9,
		},
	}

	assert.Equal(t, "gpt-4", req.Model)
	assert.Len(t, req.Messages, 2)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Equal(t, 1000, req.MaxTokens)
	assert.False(t, req.Stream)
	assert.NotNil(t, req.Extra)
}

func TestChatMessage_Fields(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "What is the weather?",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "What is the weather?", msg.Content)
}

func TestChatResponse_Fields(t *testing.T) {
	resp := ChatResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []Choice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Hello! How can I help?",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, 30, resp.Usage.TotalTokens)
}

func TestUsage_Fields(t *testing.T) {
	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	assert.Equal(t, 100, usage.PromptTokens)
	assert.Equal(t, 50, usage.CompletionTokens)
	assert.Equal(t, 150, usage.TotalTokens)
}

func TestChoice_Fields(t *testing.T) {
	choice := Choice{
		Index: 0,
		Message: ChatMessage{
			Role:    "assistant",
			Content: "Response content",
		},
		FinishReason: "stop",
	}

	assert.Equal(t, 0, choice.Index)
	assert.Equal(t, "stop", choice.FinishReason)
	assert.Equal(t, "assistant", choice.Message.Role)
}

func TestBaseProvider_SetEnabled(t *testing.T) {
	bp := NewBaseProvider("test", "key", "url", nil, true)
	assert.True(t, bp.IsEnabled())

	bp.SetEnabled(false)
	assert.False(t, bp.IsEnabled())

	bp.SetEnabled(true)
	assert.True(t, bp.IsEnabled())
}
