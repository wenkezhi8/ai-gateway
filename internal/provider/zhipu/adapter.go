package zhipu

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/provider"
)

const (
	defaultBaseURL = "https://open.bigmodel.cn/api/paas/v4"
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
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	Tools       []Tool                 `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
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
}

type ChatResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

type StreamResponse struct {
	ID      string                 `json:"id"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []StreamResponseChoice `json:"choices"`
	Usage   *ChatResponseUsage     `json:"usage,omitempty"`
}

type StreamResponseChoice struct {
	Index        int          `json:"index"`
	Delta        *StreamDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}

type StreamDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func (c *Client) ValidateKey(ctx context.Context) bool {
	req := &ChatRequest{
		Model:     "glm-4-flash",
		MaxTokens: 1,
		Messages:  []ChatMessage{{Role: "user", Content: "hi"}},
	}
	resp, err := c.DoRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode != http.StatusUnauthorized
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
					ID:   tc.ID,
					Type: tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	zhipuReq := &ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if len(req.Tools) > 0 {
		zhipuReq.Tools = make([]Tool, len(req.Tools))
		for i, t := range req.Tools {
			zhipuReq.Tools[i] = Tool{
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
		zhipuReq.ToolChoice = req.ToolChoice
	}

	if req.Temperature > 0 {
		zhipuReq.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		zhipuReq.MaxTokens = req.MaxTokens
	}

	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"].(float64); ok {
			zhipuReq.TopP = topP
		}
		if stop, ok := req.Extra["stop"].([]string); ok {
			zhipuReq.Stop = stop
		}
		if userID, ok := req.Extra["user_id"].(string); ok {
			zhipuReq.UserID = userID
		}
	}

	return zhipuReq
}

func ConvertResponse(resp *ChatResponse) *provider.ChatResponse {
	choices := make([]provider.Choice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		var toolCalls []provider.ToolCall
		if len(c.Message.ToolCalls) > 0 {
			toolCalls = make([]provider.ToolCall, len(c.Message.ToolCalls))
			for i, tc := range c.Message.ToolCalls {
				toolCalls[i] = provider.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
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
		providerErr = &provider.ProviderError{
			Code:      resp.Error.Code,
			Message:   resp.Error.Message,
			Type:      resp.Error.Type,
			Provider:  "zhipu",
			Retryable: isRetryableError(resp.Error.Code),
		}
	}

	return &provider.ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
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
		choices = append(choices, provider.StreamChoice{
			Index:        c.Index,
			Delta:        delta,
			FinishReason: c.FinishReason,
		})
	}

	return &provider.StreamChunk{
		ID:      resp.ID,
		Object:  "chat.completion.chunk",
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Done:    done,
	}
}

type Adapter struct {
	*provider.BaseProvider
	client *Client
	mu     sync.RWMutex
}

func NewAdapter(cfg *provider.ProviderConfig) *Adapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Adapter{
		BaseProvider: provider.NewBaseProvider(
			cfg.Name,
			cfg.APIKey,
			baseURL,
			cfg.Models,
			cfg.Enabled,
		),
		client: NewClient(cfg.APIKey, baseURL),
	}
}

func (a *Adapter) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	if !a.IsEnabled() {
		return nil, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "zhipu provider is disabled",
			Provider: "zhipu",
		}
	}

	zhipuReq := ConvertRequest(req)
	zhipuReq.Stream = false

	resp, err := a.client.DoRequest(ctx, "POST", "/chat/completions", zhipuReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make request: %v", err),
			Provider: "zhipu",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp ChatResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			return nil, ConvertResponse(&errResp).Error
		}
		return nil, &provider.ProviderError{
			Code:      resp.StatusCode,
			Message:   string(body),
			Provider:  "zhipu",
			Retryable: isRetryableError(resp.StatusCode),
		}
	}

	var zhipuResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&zhipuResp); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode response: %v", err),
			Provider: "zhipu",
		}
	}

	return ConvertResponse(&zhipuResp), nil
}

func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "zhipu provider is disabled",
			Provider: "zhipu",
		}
	}

	zhipuReq := ConvertRequest(req)
	zhipuReq.Stream = true

	resp, err := a.client.DoStreamRequest(ctx, "POST", "/chat/completions", zhipuReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "zhipu",
		}
	}

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				chunkChan <- &provider.StreamChunk{Done: true}
				return
			}

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			done := false
			for _, c := range streamResp.Choices {
				if c.FinishReason != "" {
					done = true
					break
				}
			}

			chunkChan <- ConvertStreamChunk(&streamResp, done)
		}
	}()

	return chunkChan, nil
}

func (a *Adapter) ValidateKey(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return a.client.ValidateKey(ctx)
}

func (a *Adapter) Name() string {
	return "zhipu"
}

func DefaultModels() []string {
	return []string{
		"glm-4-plus",
		"glm-4-0520",
		"glm-4-air",
		"glm-4-airx",
		"glm-4-long",
		"glm-4-flash",
		"glm-4",
		"glm-4v",
		"glm-3-turbo",
	}
}

func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}
