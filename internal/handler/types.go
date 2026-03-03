//nolint:godot
package handler

import (
	"strconv"
	"strings"
	"time"
)

// ChatCompletionRequest represents an OpenAI-compatible chat completion request
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Provider         string                 `json:"provider,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	N                *int                   `json:"n,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float64     `json:"logit_bias,omitempty"`
	User             string                 `json:"user,omitempty"`
	DeepThink        bool                   `json:"deepThink,omitempty"` // 改动点: 前端深度思考开关
	ReasoningEffort  string                 `json:"reasoning_effort,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	Extra            map[string]interface{} `json:"-"`
}

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role             string      `json:"role"`
	Content          interface{} `json:"content"`                     // string 或 []ContentPart (多模态)
	ReasoningContent string      `json:"reasoning_content,omitempty"` // DeepSeek R1 深度思考内容
	Name             string      `json:"name,omitempty"`
	ToolCalls        []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID       string      `json:"tool_call_id,omitempty"`
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

// ChatCompletionResponse represents an OpenAI-compatible chat completion response
type ChatCompletionResponse struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"`
	Created           int64        `json:"created"`
	Model             string       `json:"model"`
	SystemFingerprint string       `json:"system_fingerprint,omitempty"`
	Choices           []Choice     `json:"choices"`
	Usage             Usage        `json:"usage"`
	GatewayMeta       *GatewayMeta `json:"gateway_meta,omitempty"`
}

type GatewayMeta struct {
	ReasoningEffortDowngraded bool `json:"reasoning_effort_downgraded,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionRequest represents an OpenAI-compatible completion request
type CompletionRequest struct {
	Model       string                 `json:"model"`
	Prompt      interface{}            `json:"prompt"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	N           *int                   `json:"n,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Stop        interface{}            `json:"stop,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// CompletionResponse represents an OpenAI-compatible completion response
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

// CompletionChoice represents a completion choice
type CompletionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
}

// EmbeddingRequest represents an OpenAI-compatible embedding request
type EmbeddingRequest struct {
	Model string        `json:"model"`
	Input []interface{} `json:"input"`
	User  string        `json:"user,omitempty"`
}

// EmbeddingResponse represents an OpenAI-compatible embedding response
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData represents a single embedding
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// ModelInfo represents model information
type ModelInfo struct {
	ID       string    `json:"id"`
	Object   string    `json:"object"`
	Created  time.Time `json:"created"`
	OwnedBy  string    `json:"owned_by"`
	Provider string    `json:"provider,omitempty"`
	Enabled  bool      `json:"enabled,omitempty"`
}

// ModelListResponse represents the response for listing models
type ModelListResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// StreamingResponse represents a streaming chunk
type StreamingResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
	GatewayMeta       *GatewayMeta   `json:"gateway_meta,omitempty"`
}

// StreamChoice represents a choice in streaming response
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        *ChatMessage `json:"delta"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// Validate validates the chat completion request
func (r *ChatCompletionRequest) Validate() error {
	if r.Model == "" {
		return &ValidationError{Field: "model", Message: "model is required"}
	}
	if len(r.Messages) == 0 {
		return &ValidationError{Field: "messages", Message: "messages is required and cannot be empty"}
	}
	for i, msg := range r.Messages {
		if msg.Role == "" {
			return &ValidationError{Field: "messages[" + strconv.Itoa(i) + "].role", Message: "role is required"} // 改动点: 修复索引格式
		}
		if msg.Role != "system" {
			if !hasContent(msg.Content) {
				return &ValidationError{Field: "messages[" + strconv.Itoa(i) + "].content", Message: "content is required"} // 改动点: 修复索引格式
			}
		}
	}

	if strings.TrimSpace(r.ReasoningEffort) != "" {
		normalized, ok := normalizeReasoningEffort(r.ReasoningEffort)
		if !ok {
			return &ValidationError{Field: "reasoning_effort", Message: "must be one of: low, medium, high, xhigh"}
		}
		r.ReasoningEffort = normalized
	} else {
		r.ReasoningEffort = ""
	}

	return nil
}

func normalizeReasoningEffort(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", false
	}

	switch normalized {
	case "low", "medium", "high", "xhigh":
		return normalized, true
	default:
		return "", false
	}
}

// hasContent checks if content is non-empty (supports string and []ContentPart)
func hasContent(content interface{}) bool {
	switch v := content.(type) {
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	default:
		return content != nil
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
