package ernie

import (
	"bufio"
	"bytes"
	"context"
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
	defaultBaseURL = "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop"
	defaultTimeout = 60 * time.Second
)

type Client struct {
	baseURL    string
	apiKey     string
	secretKey  string
	httpClient *http.Client
}

func NewClient(apiKey, secretKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:   baseURL,
		apiKey:    apiKey,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	reqURL := fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s",
		url.QueryEscape(c.apiKey), url.QueryEscape(c.secretKey))

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("failed to get access token: %s", tokenResp.Error)
	}
	return tokenResp.AccessToken, nil
}

func (c *Client) DoRequest(ctx context.Context, accessToken, modelPath string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	reqURL := fmt.Sprintf("%s%s?access_token=%s", c.baseURL, modelPath, accessToken)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) DoStreamRequest(ctx context.Context, accessToken, modelPath string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	reqURL := fmt.Sprintf("%s%s?access_token=%s", c.baseURL, modelPath, accessToken)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	return c.httpClient.Do(req)
}

type ChatRequest struct {
	Messages        []ChatMessage          `json:"messages"`
	Temperature     float64                `json:"temperature,omitempty"`
	TopP            float64                `json:"top_p,omitempty"`
	MaxOutputTokens int                    `json:"max_output_tokens,omitempty"`
	Stream          bool                   `json:"stream,omitempty"`
	Stop            []string               `json:"stop,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Functions       []Function             `json:"functions,omitempty"`
	Extra           map[string]interface{} `json:"-"`
}

type ChatMessage struct {
	Role         string        `json:"role"`
	Content      interface{}   `json:"content,omitempty"` // 支持 string 或 []interface{} (多模态)
	Name         string        `json:"name,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	ID               string             `json:"id"`
	Object           string             `json:"object"`
	Created          int64              `json:"created"`
	Result           string             `json:"result"`
	IsTruncated      bool               `json:"is_truncated"`
	NeedClearHistory bool               `json:"need_clear_history"`
	Usage            ChatResponseUsage  `json:"usage"`
	FunctionCall     *FunctionCall      `json:"function_call,omitempty"`
	Error            *ChatResponseError `json:"error,omitempty"`
	ErrorMsg         string             `json:"error_msg,omitempty"`
	ErrorCode        int                `json:"error_code,omitempty"`
}

type ChatResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Msg     string `json:"msg"`
}

type StreamResponse struct {
	ID               string             `json:"id"`
	Object           string             `json:"object"`
	Created          int64              `json:"created"`
	Result           string             `json:"result"`
	IsEnd            bool               `json:"is_end"`
	IsTruncated      bool               `json:"is_truncated"`
	NeedClearHistory bool               `json:"need_clear_history"`
	Usage            *ChatResponseUsage `json:"usage,omitempty"`
	FunctionCall     *FunctionCall      `json:"function_call,omitempty"`
	ErrorMsg         string             `json:"error_msg,omitempty"`
	ErrorCode        int                `json:"error_code,omitempty"`
}

func (c *Client) ValidateKey(ctx context.Context) bool {
	_, err := c.getAccessToken(ctx)
	return err == nil
}

func isRetryableError(code int) bool {
	return code >= 500 || code == 429 || code == 408
}

func getModelPath(model string) string {
	modelPaths := map[string]string{
		"ernie-4.0-8k":   "/completions_pro",
		"ernie-4.0":      "/completions_pro",
		"ernie-3.5-8k":   "/completions",
		"ernie-3.5":      "/completions",
		"ernie-speed-8k": "/ernie_speed",
		"ernie-speed":    "/ernie_speed",
		"ernie-lite-8k":  "/ernie_lite",
		"ernie-lite":     "/ernie_lite",
		"ernie-tiny-8k":  "/ernie_tiny",
	}
	if path, ok := modelPaths[model]; ok {
		return path
	}
	return "/completions"
}

func ConvertRequest(req *provider.ChatRequest) *ChatRequest {
	messages := make([]ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msg := ChatMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		}
		if len(m.ToolCalls) > 0 {
			tc := m.ToolCalls[0]
			msg.FunctionCall = &FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
		messages = append(messages, msg)
	}

	ernieReq := &ChatRequest{
		Messages: messages,
		Stream:   req.Stream,
	}

	if len(req.Tools) > 0 {
		ernieReq.Functions = make([]Function, len(req.Tools))
		for i, t := range req.Tools {
			ernieReq.Functions[i] = Function{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			}
		}
	}

	if req.Temperature > 0 {
		ernieReq.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		ernieReq.MaxOutputTokens = req.MaxTokens
	}

	if req.Extra != nil {
		if topP, ok := req.Extra["top_p"].(float64); ok {
			ernieReq.TopP = topP
		}
		if stop, ok := req.Extra["stop"].([]string); ok {
			ernieReq.Stop = stop
		}
		if userID, ok := req.Extra["user_id"].(string); ok {
			ernieReq.UserID = userID
		}
	}

	return ernieReq
}

func ConvertResponse(resp *ChatResponse, model string) *provider.ChatResponse {
	var toolCalls []provider.ToolCall
	if resp.FunctionCall != nil {
		toolCalls = []provider.ToolCall{{
			ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
			Type: "function",
			Function: provider.FunctionCall{
				Name:      resp.FunctionCall.Name,
				Arguments: resp.FunctionCall.Arguments,
			},
		}}
	}

	choices := []provider.Choice{{
		Index: 0,
		Message: provider.ChatMessage{
			Role:      "assistant",
			Content:   resp.Result,
			ToolCalls: toolCalls,
		},
		FinishReason: "stop",
	}}

	var providerErr *provider.ProviderError
	if resp.Error != nil || resp.ErrorMsg != "" {
		code := resp.ErrorCode
		if code == 0 && resp.Error != nil {
			code = resp.Error.Code
		}
		msg := resp.ErrorMsg
		if msg == "" && resp.Error != nil {
			msg = resp.Error.Message
		}
		providerErr = &provider.ProviderError{
			Code:      code,
			Message:   msg,
			Provider:  "ernie",
			Retryable: isRetryableError(code),
		}
	}

	return &provider.ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created,
		Model:   model,
		Choices: choices,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Error: providerErr,
	}
}

func ConvertStreamChunk(resp *StreamResponse, model string, done bool) *provider.StreamChunk {
	var toolCalls []provider.ToolCall
	if resp.FunctionCall != nil {
		toolCalls = []provider.ToolCall{{
			ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
			Type: "function",
			Function: provider.FunctionCall{
				Name:      resp.FunctionCall.Name,
				Arguments: resp.FunctionCall.Arguments,
			},
		}}
	}

	finishReason := ""
	if resp.IsEnd {
		finishReason = "stop"
	}

	var delta *provider.StreamDelta
	if resp.Result != "" || len(toolCalls) > 0 {
		delta = &provider.StreamDelta{
			Role:      "assistant",
			Content:   resp.Result,
			ToolCalls: toolCalls,
		}
	}

	choices := []provider.StreamChoice{{
		Index:        0,
		Delta:        delta,
		FinishReason: finishReason,
	}}

	return &provider.StreamChunk{
		ID:      resp.ID,
		Object:  "chat.completion.chunk",
		Created: resp.Created,
		Model:   model,
		Choices: choices,
		Done:    done || resp.IsEnd,
	}
}

type Adapter struct {
	*provider.BaseProvider
	client    *Client
	secretKey string
}

func NewAdapter(cfg *provider.ProviderConfig) *Adapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	secretKey := ""
	if cfg.Extra != nil {
		if sk, ok := cfg.Extra["secret_key"].(string); ok {
			secretKey = sk
		}
	}

	return &Adapter{
		BaseProvider: provider.NewBaseProvider(
			cfg.Name,
			cfg.APIKey,
			baseURL,
			cfg.Models,
			cfg.Enabled,
		),
		client:    NewClient(cfg.APIKey, secretKey, baseURL),
		secretKey: secretKey,
	}
}

func (a *Adapter) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	if !a.IsEnabled() {
		return nil, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "ernie provider is disabled",
			Provider: "ernie",
		}
	}

	accessToken, err := a.client.getAccessToken(ctx)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusUnauthorized,
			Message:  fmt.Sprintf("failed to get access token: %v", err),
			Provider: "ernie",
		}
	}

	ernieReq := ConvertRequest(req)
	ernieReq.Stream = false
	modelPath := getModelPath(req.Model)

	resp, err := a.client.DoRequest(ctx, accessToken, modelPath, ernieReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make request: %v", err),
			Provider: "ernie",
		}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to read response: %v", readErr),
			Provider: "ernie",
		}
	}

	var ernieResp ChatResponse
	if err := json.Unmarshal(body, &ernieResp); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode response: %v", err),
			Provider: "ernie",
		}
	}

	if ernieResp.ErrorMsg != "" || ernieResp.Error != nil {
		return nil, ConvertResponse(&ernieResp, req.Model).Error
	}

	return ConvertResponse(&ernieResp, req.Model), nil
}

func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "ernie provider is disabled",
			Provider: "ernie",
		}
	}

	accessToken, err := a.client.getAccessToken(ctx)
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusUnauthorized,
			Message:  fmt.Sprintf("failed to get access token: %v", err),
			Provider: "ernie",
		}
	}

	ernieReq := ConvertRequest(req)
	ernieReq.Stream = true
	modelPath := getModelPath(req.Model)

	resp, err := a.client.DoStreamRequest(ctx, accessToken, modelPath, ernieReq) //nolint:bodyclose
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "ernie",
		}
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(readErr.Error())
		}
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:      resp.StatusCode,
			Message:   string(body),
			Provider:  "ernie",
			Retryable: isRetryableError(resp.StatusCode),
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

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if streamResp.ErrorMsg != "" {
				chunkChan <- &provider.StreamChunk{
					Done: true,
				}
				return
			}

			chunkChan <- ConvertStreamChunk(&streamResp, req.Model, streamResp.IsEnd)

			if streamResp.IsEnd {
				return
			}
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
	return "ernie"
}

func DefaultModels() []string {
	return []string{
		"ernie-4.0-8k",
		"ernie-4.0",
		"ernie-3.5-8k",
		"ernie-3.5",
		"ernie-speed-8k",
		"ernie-speed",
		"ernie-lite-8k",
		"ernie-lite",
	}
}

func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}
