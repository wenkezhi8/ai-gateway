package deepseek

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
	defaultBaseURL = "https://api.deepseek.com"
	defaultTimeout = 60 * time.Second
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

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

type ChatRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	N                int                    `json:"n,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	StreamOptions    *StreamOptions         `json:"stream_options,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	Logprobs         bool                   `json:"logprobs,omitempty"`
	TopLogprobs      int                    `json:"top_logprobs,omitempty"`
	User             string                 `json:"user,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	ResponseFormat   interface{}            `json:"response_format,omitempty"`
	Extra            map[string]interface{} `json:"-"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string 或 []ContentPart (多模态)
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolCall struct {
	Index    int          `json:"index,omitempty"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []ChatResponseChoice `json:"choices"`
	Usage   ChatResponseUsage    `json:"usage"`
	Error   *ChatResponseError   `json:"error,omitempty"`
}

type ChatResponseChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
}

type ChatResponseUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`
}

type ChatResponseError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
	Param      string `json:"param,omitempty"`
	StatusCode int    `json:"-"`
}

type StreamResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []StreamResponseChoice `json:"choices"`
	Usage   *ChatResponseUsage     `json:"usage,omitempty"`
}

type StreamResponseChoice struct {
	Index        int          `json:"index"`
	Delta        *StreamDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
	Logprobs     interface{}  `json:"logprobs,omitempty"`
}

type StreamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func (c *Client) ValidateKey(ctx context.Context) bool {
	resp, err := c.DoRequest(ctx, "GET", "/v1/models", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func isRetryableError(statusCode int) bool {
	return statusCode >= 500 || statusCode == 429 || statusCode == 408
}

func ConvertRequest(req *provider.ChatRequest) *ChatRequest {
	messages := make([]ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ChatMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				messages[i].ToolCalls[j] = ToolCall{
					Index: tc.Index,
					ID:    tc.ID,
					Type:  tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	dsReq := &ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	// Enable usage statistics for streaming requests
	if req.Stream {
		dsReq.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	if len(req.Tools) > 0 {
		dsReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			dsReq.Tools[i] = Tool{
				Type: t.Type,
				Function: Function{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}
	if req.ToolChoice != nil {
		dsReq.ToolChoice = req.ToolChoice
	}

	if req.Temperature > 0 {
		dsReq.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		dsReq.MaxTokens = req.MaxTokens
	}

	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"].(float64); ok {
			dsReq.TopP = topP
		}
		if topK, ok := req.Extra["top_k"].(int); ok {
			dsReq.TopK = topK
		}
		if freqPenalty, ok := req.Extra["frequency_penalty"].(float64); ok {
			dsReq.FrequencyPenalty = freqPenalty
		}
		if presPenalty, ok := req.Extra["presence_penalty"].(float64); ok {
			dsReq.PresencePenalty = presPenalty
		}
		if stop, ok := req.Extra["stop"].([]string); ok {
			dsReq.Stop = stop
		}
		if user, ok := req.Extra["user"].(string); ok {
			dsReq.User = user
		}
		if responseFormat, ok := req.Extra["response_format"].(interface{}); ok {
			dsReq.ResponseFormat = responseFormat
		}
	}

	return dsReq
}

func ConvertResponse(resp *ChatResponse) *provider.ChatResponse {
	choices := make([]provider.Choice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		var toolCalls []provider.ToolCall
		if len(c.Message.ToolCalls) > 0 {
			toolCalls = make([]provider.ToolCall, len(c.Message.ToolCalls))
			for i, tc := range c.Message.ToolCalls {
				toolCalls[i] = provider.ToolCall{
					Index: tc.Index,
					ID:    tc.ID,
					Type:  tc.Type,
					Function: provider.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choice := provider.Choice{
			Index: c.Index,
			Message: provider.ChatMessage{
				Role:      c.Message.Role,
				Content:   c.Message.Content,
				Name:      c.Message.Name,
				ToolCalls: toolCalls,
			},
			FinishReason: c.FinishReason,
		}
		choices = append(choices, choice)
	}

	var providerErr *provider.ProviderError
	if resp.Error != nil {
		statusCode := resp.Error.StatusCode
		if statusCode == 0 {
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
			case "insufficient_quota":
				statusCode = 429
			case "server_error":
				statusCode = 500
			}
		}

		providerErr = &provider.ProviderError{
			Code:      statusCode,
			Message:   resp.Error.Message,
			Type:      resp.Error.Type,
			Provider:  "deepseek",
			Retryable: isRetryableError(statusCode),
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

func ConvertStreamChunk(resp *StreamResponse, done bool) *provider.StreamChunk {
	choices := make([]provider.StreamChoice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		var delta *provider.StreamDelta
		if c.Delta != nil {
			delta = &provider.StreamDelta{
				Role:    c.Delta.Role,
				Content: c.Delta.Content,
			}
			if len(c.Delta.ToolCalls) > 0 {
				delta.ToolCalls = make([]provider.ToolCall, len(c.Delta.ToolCalls))
				for i, tc := range c.Delta.ToolCalls {
					delta.ToolCalls[i] = provider.ToolCall{
						Index: tc.Index,
						ID:    tc.ID,
						Type:  tc.Type,
						Function: provider.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
		}
		choices = append(choices, provider.StreamChoice{
			Index:        c.Index,
			Delta:        delta,
			FinishReason: c.FinishReason,
		})
	}

	var usage *provider.Usage
	if resp.Usage != nil {
		usage = &provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return &provider.StreamChunk{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
		Done:    done,
	}
}
