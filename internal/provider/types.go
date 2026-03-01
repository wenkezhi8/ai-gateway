//nolint:godot,gocritic,revive
package provider

import (
	"context"
)

// ChatRequest represents a unified chat completion request
type ChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Tools       []Tool                 `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	Extra       map[string]interface{} `json:"-"`
	RawBody     map[string]interface{} `json:"-"`
}

// ChatMessage represents a message in a chat
type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content,omitempty"` // string 或 []ContentPart (多模态)
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ContentPart represents a part of multimodal content
type ContentPart struct {
	Type     string    `json:"type"` // "text" 或 "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL in multimodal content
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// GetTextContent extracts text content from ChatMessage.Content
func (m *ChatMessage) GetTextContent() string {
	switch v := m.Content.(type) {
	case string:
		return v
	case []ContentPart:
		for _, part := range v {
			if part.Type == "text" {
				return part.Text
			}
		}
	case []interface{}:
		for _, item := range v {
			if pm, ok := item.(map[string]interface{}); ok {
				if pm["type"] == "text" {
					if text, ok := pm["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

// IsMultimodal checks if the message contains multimodal content
func (m *ChatMessage) IsMultimodal() bool {
	switch m.Content.(type) {
	case []ContentPart, []interface{}:
		return true
	}
	return false
}

// GetContentParts returns content as []ContentPart if multimodal, nil otherwise
func (m *ChatMessage) GetContentParts() []ContentPart {
	switch v := m.Content.(type) {
	case []ContentPart:
		return v
	case []interface{}:
		parts := make([]ContentPart, 0, len(v))
		for _, item := range v {
			if pm, ok := item.(map[string]interface{}); ok {
				part := ContentPart{}
				if t, ok := pm["type"].(string); ok {
					part.Type = t
				}
				if text, ok := pm["text"].(string); ok {
					part.Text = text
				}
				if imgURL, ok := pm["image_url"].(map[string]interface{}); ok {
					part.ImageURL = &ImageURL{}
					if url, ok := imgURL["url"].(string); ok {
						part.ImageURL.URL = url
					}
					if detail, ok := imgURL["detail"].(string); ok {
						part.ImageURL.Detail = detail
					}
				}
				parts = append(parts, part)
			}
		}
		return parts
	}
	return nil
}

// Tool represents a tool definition
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	Index    int          `json:"index,omitempty"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse represents a unified chat completion response
type ChatResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []Choice       `json:"choices"`
	Usage   Usage          `json:"usage"`
	Extra   map[string]any `json:"-"`
	RawBody map[string]any `json:"-"`
	Error   *ProviderError `json:"error,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderError represents an error from a provider
type ProviderError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Type      string `json:"type,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return e.Message
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
	Error   *ProviderError `json:"error,omitempty"`
	Done    bool           `json:"-"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        *StreamDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}

// StreamDelta represents the delta content in a streaming response
type StreamDelta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // DeepSeek R1 深度思考内容
	Reasoning        string     `json:"reasoning,omitempty"`         // 兼容字段
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ProviderConfig holds provider configuration
type ProviderConfig struct {
	Name    string         `json:"name"`
	APIKey  string         `json:"api_key"`
	BaseURL string         `json:"base_url"`
	Models  []string       `json:"models"`
	Enabled bool           `json:"enabled"`
	Extra   map[string]any `json:"extra"`
}

// Provider is the interface that all AI providers must implement
type Provider interface {
	// Name returns the provider name
	Name() string

	// Chat sends a chat completion request
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a streaming chat completion request
	StreamChat(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error)

	// Models returns the list of supported models
	Models() []string

	// ValidateKey validates the API key
	ValidateKey(ctx context.Context) bool

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool

	// SetEnabled enables or disables the provider
	SetEnabled(enabled bool)
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name    string
	apiKey  string
	baseURL string
	models  []string
	enabled bool
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name, apiKey, baseURL string, models []string, enabled bool) *BaseProvider {
	return &BaseProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		enabled: enabled,
	}
}

// Name returns the provider name
func (p *BaseProvider) Name() string {
	return p.name
}

// Models returns supported models
func (p *BaseProvider) Models() []string {
	return p.models
}

// APIKey returns the API key
func (p *BaseProvider) APIKey() string {
	return p.apiKey
}

// BaseURL returns the base URL
func (p *BaseProvider) BaseURL() string {
	return p.baseURL
}

// IsEnabled returns whether the provider is enabled
func (p *BaseProvider) IsEnabled() bool {
	return p.enabled
}

// SetEnabled enables or disables the provider
func (p *BaseProvider) SetEnabled(enabled bool) {
	p.enabled = enabled
}
