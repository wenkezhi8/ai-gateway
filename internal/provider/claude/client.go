package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-gateway/internal/provider"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	defaultTimeout = 120 * time.Second // Claude may take longer for complex requests
)

// Client implements the HTTP client for Anthropic Claude API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Claude client
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// DoRequest makes an HTTP request to the Claude API
func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	return c.httpClient.Do(req)
}

// DoStreamRequest makes a streaming HTTP request
func (c *Client) DoStreamRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	return c.httpClient.Do(req)
}

// Claude API request/response types

// MessagesRequest represents a Claude messages request
type MessagesRequest struct {
	Model         string      `json:"model"`
	Messages      []Message   `json:"messages"`
	System        interface{} `json:"system,omitempty"`
	MaxTokens     int         `json:"max_tokens"`
	Temperature   float64     `json:"temperature,omitempty"`
	TopP          float64     `json:"top_p,omitempty"`
	TopK          int         `json:"top_k,omitempty"`
	Stream        bool        `json:"stream,omitempty"`
	StopSequences []string    `json:"stop_sequences,omitempty"`
	Tools         []Tool      `json:"tools,omitempty"`
	ToolChoice    interface{} `json:"tool_choice,omitempty"`
	Metadata      *Metadata   `json:"metadata,omitempty"`
}

// Message represents a message in Claude
type Message struct {
	Role    string         `json:"role"`
	Content MessageContent `json:"content"`
}

// MessageContent can be a string or array of content blocks
type MessageContent []ContentBlock

// ContentBlock represents a content block
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// For images
	Source *ImageSource `json:"source,omitempty"`
	// For tool use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// For tool result
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

// ImageSource represents an image source
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// Tool represents a tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Metadata represents request metadata
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// MessagesResponse represents a Claude messages response
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason,omitempty"`
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        ResponseUsage  `json:"usage"`
	Error        *ResponseError `json:"error,omitempty"`
}

// ResponseUsage represents token usage
type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ResponseError represents an error response
type ResponseError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type         string            `json:"type"`
	Index        int               `json:"index,omitempty"`
	Delta        *StreamDelta      `json:"delta,omitempty"`
	ContentBlock *ContentBlock     `json:"content_block,omitempty"`
	Message      *MessagesResponse `json:"message,omitempty"`
	Usage        *ResponseUsage    `json:"usage,omitempty"`
}

// StreamDelta represents delta content in streaming
type StreamDelta struct {
	Type       string          `json:"type,omitempty"`
	Text       string          `json:"text,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
}

// ValidateKey validates the API key
func (c *Client) ValidateKey(ctx context.Context) bool {
	// Make a minimal request to validate the key
	req := &MessagesRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 1,
		Messages: []Message{
			{Role: "user", Content: []ContentBlock{{Type: "text", Text: "hi"}}},
		},
	}

	resp, err := c.DoRequest(ctx, "POST", "/messages", req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 400 might mean invalid request but valid auth
	// 401 means invalid auth
	return resp.StatusCode != http.StatusUnauthorized
}

// isRetryableError determines if an error code is retryable
func isRetryableError(statusCode int) bool {
	return statusCode >= 500 || statusCode == 429 || statusCode == 408
}

// ConvertToProviderResponse converts Claude response to provider response
func ConvertToProviderResponse(resp *MessagesResponse) *provider.ChatResponse {
	choices := make([]provider.Choice, 1)
	content := ""
	var toolCalls []provider.ToolCall

	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		} else if block.Type == "tool_use" {
			if toolCalls == nil {
				toolCalls = make([]provider.ToolCall, 0)
			}
			args := "{}"
			if len(block.Input) > 0 {
				args = string(block.Input)
			}
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: provider.FunctionCall{
					Name:      block.Name,
					Arguments: args,
				},
			})
		}
	}

	choices[0] = provider.Choice{
		Index: 0,
		Message: provider.ChatMessage{
			Role:      resp.Role,
			Content:   content,
			ToolCalls: toolCalls,
		},
		FinishReason: resp.StopReason,
	}

	var providerErr *provider.ProviderError
	if resp.Error != nil {
		statusCode := 500
		switch resp.Error.Type {
		case "invalid_request_error":
			statusCode = 400
		case "authentication_error":
			statusCode = 401
		case "permission_error":
			statusCode = 403
		case "not_found_error":
			statusCode = 404
		case "rate_limit_error":
			statusCode = 429
		case "overloaded_error":
			statusCode = 529
		}

		providerErr = &provider.ProviderError{
			Code:      statusCode,
			Message:   resp.Error.Message,
			Type:      resp.Error.Type,
			Provider:  "claude",
			Retryable: isRetryableError(statusCode),
		}
	}

	return &provider.ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: choices,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Error: providerErr,
	}
}
