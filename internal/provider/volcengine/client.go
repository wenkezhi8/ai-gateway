//nolint:godot,gocritic,dupl
package volcengine

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-gateway/internal/provider"
)

const (
	defaultBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
	defaultTimeout = 60 * time.Second
)

// Client implements the HTTP client for Volcengine Ark API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Volcengine client
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

// DoRequest makes an HTTP request to the Volcengine API
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	return c.httpClient.Do(req)
}

// Volcengine API request/response types

// ChatRequest represents a Volcengine chat request
type ChatRequest struct {
	Model            string        `json:"model"`
	Messages         []ChatMessage `json:"messages"`
	Temperature      float64       `json:"temperature,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	Stream           bool          `json:"stream,omitempty"`
	Stop             []string      `json:"stop,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
	Logprobs         bool          `json:"logprobs,omitempty"`
	TopLogprobs      int           `json:"top_logprobs,omitempty"`
	N                int           `json:"n,omitempty"`
	User             string        `json:"user,omitempty"`
	Tools            []Tool        `json:"tools,omitempty"`
	ToolChoice       interface{}   `json:"tool_choice,omitempty"`
}

// ChatMessage represents a message in Volcengine chat
type ChatMessage struct {
	Role      string      `json:"role"`
	Content   interface{} `json:"content,omitempty"` // 支持 string 或 []interface{} (多模态)
	Name      string      `json:"name,omitempty"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
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
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse represents a Volcengine chat response
type ChatResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []ChatResponseChoice `json:"choices"`
	Usage   ChatResponseUsage    `json:"usage"`
	Error   *ChatResponseError   `json:"error,omitempty"`
}

// ChatResponseChoice represents a choice in response
type ChatResponseChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
}

// ChatResponseUsage represents token usage
type ChatResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponseError represents an error response
type ChatResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []StreamResponseChoice `json:"choices"`
	Usage   *ChatResponseUsage     `json:"usage,omitempty"`
}

// StreamResponseChoice represents a choice in streaming response
type StreamResponseChoice struct {
	Index        int          `json:"index"`
	Delta        *StreamDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}

// StreamDelta represents delta content in streaming
type StreamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ModelsResponse represents the models list response
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// ModelInfo represents model information
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ValidateKey validates the API key by making a test request
func (c *Client) ValidateKey(ctx context.Context) bool {
	// Volcengine uses Bearer token authentication
	// We validate by making a simple models list request
	resp, err := c.DoRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// SignRequest signs a request for Volcengine API (for non-Bearer auth scenarios)
func (c *Client) SignRequest(method, path string, body []byte, timestamp time.Time) string {
	// Parse the base URL to extract host
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return ""
	}
	host := parsedURL.Host

	// Build canonical request
	canonicalURI := path
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\n", host)
	signedHeaders := "content-type;host"

	// Hash payload
	payloadHash := sha256.Sum256(body)
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		strings.ToUpper(method),
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHashHex,
	)

	// Build string to sign
	algorithm := "HMAC-SHA256"
	date := timestamp.UTC().Format("20060102T150405Z")
	credentialScope := timestamp.UTC().Format("20060102") + "/cn-beijing/ark/request"

	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		date,
		credentialScope,
		hex.EncodeToString(canonicalRequestHash[:]),
	)

	// Calculate signature
	mac := hmac.New(sha256.New, []byte(c.apiKey))
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	return signature
}

// ParseStreamResponse parses a Server-Sent Events stream
func ParseStreamResponse(reader io.Reader) (<-chan *StreamResponse, <-chan error) {
	eventChan := make(chan *StreamResponse, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		var gzipReader io.Reader

		// Try to detect and handle gzip compression
		buf := make([]byte, 2)
		n, err := reader.Read(buf)
		if err != nil {
			errChan <- err
			return
		}

		// Check for gzip magic number
		if n >= 2 && buf[0] == 0x1f && buf[1] == 0x8b {
			var gzipReadCloser io.ReadCloser
			gzipReadCloser, err = gzip.NewReader(io.MultiReader(bytes.NewReader(buf), reader))
			if err != nil {
				errChan <- err
				return
			}
			defer func() {
				_ = gzipReadCloser.Close()
			}()
			gzipReader = gzipReadCloser
		} else {
			// Not gzipped, create a multi-reader with the already-read bytes
			gzipReader = io.MultiReader(bytes.NewReader(buf), reader)
		}

		decoder := json.NewDecoder(gzipReader)

		for {
			var event StreamResponse
			if err := decoder.Decode(&event); err != nil {
				if err == io.EOF {
					return
				}
				// Try to read SSE format
				continue
			}
			eventChan <- &event
		}
	}()

	return eventChan, errChan
}

// ConvertToProviderResponse converts Volcengine response to provider response
func ConvertToProviderResponse(resp *ChatResponse) *provider.ChatResponse {
	choices := make([]provider.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		var toolCalls []provider.ToolCall
		if len(c.Message.ToolCalls) > 0 {
			toolCalls = make([]provider.ToolCall, len(c.Message.ToolCalls))
			for j, tc := range c.Message.ToolCalls {
				toolCalls[j] = provider.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: provider.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		choices[i] = provider.Choice{
			Index: c.Index,
			Message: provider.ChatMessage{
				Role:      c.Message.Role,
				Content:   c.Message.Content,
				Name:      c.Message.Name,
				ToolCalls: toolCalls,
			},
			FinishReason: c.FinishReason,
		}
	}

	var providerErr *provider.ProviderError
	if resp.Error != nil {
		providerErr = &provider.ProviderError{
			Code:      resp.Error.Code,
			Message:   resp.Error.Message,
			Type:      resp.Error.Type,
			Provider:  "volcengine",
			Retryable: resp.Error.Code >= 500 || resp.Error.Code == 429,
		}
	}

	return &provider.ChatResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Error: providerErr,
	}
}

// ConvertToProviderChunk converts stream response to provider chunk
func ConvertToProviderChunk(resp *StreamResponse, done bool) *provider.StreamChunk {
	choices := make([]provider.StreamChoice, len(resp.Choices))
	for i, c := range resp.Choices {
		var delta *provider.StreamDelta
		if c.Delta != nil {
			delta = &provider.StreamDelta{
				Role:    c.Delta.Role,
				Content: c.Delta.Content,
			}
			if len(c.Delta.ToolCalls) > 0 {
				delta.ToolCalls = make([]provider.ToolCall, len(c.Delta.ToolCalls))
				for j, tc := range c.Delta.ToolCalls {
					delta.ToolCalls[j] = provider.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: provider.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
		}
		choices[i] = provider.StreamChoice{
			Index:        c.Index,
			Delta:        delta,
			FinishReason: c.FinishReason,
		}
	}

	return &provider.StreamChunk{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Done:    done,
	}
}
