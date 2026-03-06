//nolint:godot
package openai

import (
	"bufio"
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

// Adapter implements the Provider interface for OpenAI
type Adapter struct {
	*provider.BaseProvider
	client *Client
	orgID  string
	mu     sync.RWMutex
}

// NewAdapter creates a new OpenAI adapter
func NewAdapter(cfg *provider.ProviderConfig) *Adapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	orgID := ""
	if cfg.Extra != nil {
		if id, ok := cfg.Extra["organization_id"].(string); ok {
			orgID = id
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
		client: NewClient(cfg.APIKey, baseURL, orgID),
		orgID:  orgID,
	}
}

// Chat sends a chat completion request to OpenAI
func (a *Adapter) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	if !a.IsEnabled() {
		return nil, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "openai provider is disabled",
			Provider: "openai",
		}
	}

	// Convert to OpenAI request format
	openaiReq := ConvertRequest(req)
	openaiReq.Stream = false

	// Make the API call
	resp, err := a.client.DoRequest(ctx, "POST", "/chat/completions", openaiReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make request: %v", err),
			Provider: "openai",
		}
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(readErr.Error())
		}
		if shouldUseResponsesFallback(resp.StatusCode, body) {
			return a.chatWithResponses(ctx, req)
		}
		var errResp ChatResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			errResp.Error.StatusCode = resp.StatusCode
			return nil, ConvertResponse(&errResp).Error
		}
		return nil, &provider.ProviderError{
			Code:      resp.StatusCode,
			Message:   string(body),
			Provider:  "openai",
			Retryable: isRetryableError(resp.StatusCode),
		}
	}

	// Parse response
	var openaiResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode response: %v", err),
			Provider: "openai",
		}
	}

	return ConvertResponse(&openaiResp), nil
}

// StreamChat sends a streaming chat completion request to OpenAI
func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "openai provider is disabled",
			Provider: "openai",
		}
	}

	// Convert to OpenAI request format
	openaiReq := ConvertRequest(req)
	openaiReq.Stream = true

	// Make the streaming API call
	resp, err := a.client.DoStreamRequest(ctx, "POST", "/chat/completions", openaiReq) //nolint:bodyclose
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "openai",
		}
	}

	// Some OpenAI-compatible backends only support /v1/responses.
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(readErr.Error())
		}
		_ = resp.Body.Close()
		if shouldUseResponsesFallback(resp.StatusCode, body) {
			return a.streamViaResponses(ctx, req)
		}
		close(chunkChan)
		return chunkChan, providerErrorFromBody(resp.StatusCode, body)
	}

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// Handle SSE format (both "data: " and "data:")
			var data string
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			} else if strings.HasPrefix(line, "data:") {
				data = strings.TrimPrefix(line, "data:")
			} else {
				continue
			}

			// Check for stream end
			if data == "[DONE]" {
				chunkChan <- &provider.StreamChunk{Done: true}
				return
			}

			// Parse the JSON data
			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			// Check if this is the last chunk
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

func (a *Adapter) chatWithResponses(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	respBody, err := a.callResponsesAPI(ctx, req, false)
	if err != nil {
		return nil, err
	}

	id := getStringFromMap(respBody, "id")
	model := getStringFromMap(respBody, "model")
	if model == "" {
		model = req.Model
	}
	if id == "" {
		id = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}

	content := extractResponsesText(respBody)
	usage := extractResponsesUsage(respBody)

	return &provider.ChatResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []provider.Choice{
			{
				Index: 0,
				Message: provider.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: usage,
	}, nil
}

func (a *Adapter) streamViaResponses(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 8)

	resp, err := a.chatWithResponses(ctx, req)
	if err != nil {
		close(chunkChan)
		return chunkChan, err
	}

	go func() {
		defer close(chunkChan)

		content := ""
		if len(resp.Choices) > 0 {
			if text, ok := resp.Choices[0].Message.Content.(string); ok {
				content = text
			}
		}

		if content != "" {
			chunkChan <- &provider.StreamChunk{
				ID:      resp.ID,
				Object:  "chat.completion.chunk",
				Created: resp.Created,
				Model:   resp.Model,
				Choices: []provider.StreamChoice{
					{
						Index: 0,
						Delta: &provider.StreamDelta{
							Role:    "assistant",
							Content: content,
						},
					},
				},
			}
		}

		chunkChan <- &provider.StreamChunk{
			ID:      resp.ID,
			Object:  "chat.completion.chunk",
			Created: resp.Created,
			Model:   resp.Model,
			Choices: []provider.StreamChoice{
				{
					Index:        0,
					Delta:        &provider.StreamDelta{},
					FinishReason: "stop",
				},
			},
			Usage: &provider.Usage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
				CachedReadTokens: resp.Usage.CachedReadTokens,
			},
			Done: true,
		}
	}()

	return chunkChan, nil
}

func (a *Adapter) callResponsesAPI(ctx context.Context, req *provider.ChatRequest, stream bool) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"model":  req.Model,
		"input":  buildResponsesInput(req.Messages),
		"stream": stream,
	}

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_output_tokens"] = req.MaxTokens
	}
	if req.Extra != nil {
		if reasoningEffort, ok := req.Extra["reasoning_effort"].(string); ok && strings.TrimSpace(reasoningEffort) != "" {
			body["reasoning_effort"] = reasoningEffort
		}
	}

	resp, err := a.client.DoRequest(ctx, "POST", "/responses", body)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make responses request: %v", err),
			Provider: "openai",
		}
	}
	defer resp.Body.Close()

	respBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to read responses body: %v", readErr),
			Provider: "openai",
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, providerErrorFromBody(resp.StatusCode, respBytes)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode responses body: %v", err),
			Provider: "openai",
		}
	}

	return parsed, nil
}

func shouldUseResponsesFallback(statusCode int, body []byte) bool {
	if statusCode < 400 {
		return false
	}
	text := strings.ToLower(string(body))
	return strings.Contains(text, "unsupported legacy protocol") && strings.Contains(text, "/v1/responses")
}

func providerErrorFromBody(statusCode int, body []byte) *provider.ProviderError {
	message := strings.TrimSpace(string(body))
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if errObj, ok := parsed["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				message = msg
			}
		}
	}
	return &provider.ProviderError{
		Code:      statusCode,
		Message:   message,
		Provider:  "openai",
		Retryable: isRetryableError(statusCode),
	}
}

func buildResponsesInput(messages []provider.ChatMessage) []map[string]interface{} {
	input := make([]map[string]interface{}, 0, len(messages))
	for i := range messages {
		msg := messages[i]
		text := ""
		switch v := msg.Content.(type) {
		case string:
			text = v
		default:
			text = (&msg).GetTextContent()
		}
		input = append(input, map[string]interface{}{
			"role":    msg.Role,
			"content": text,
		})
	}
	return input
}

func extractResponsesText(resp map[string]interface{}) string {
	if outputText, ok := resp["output_text"].(string); ok && outputText != "" {
		return outputText
	}
	output, ok := resp["output"].([]interface{})
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, item := range output {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		contentArr, ok := itemMap["content"].([]interface{})
		if !ok {
			continue
		}
		for _, c := range contentArr {
			contentMap, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := contentMap["text"].(string); ok {
				b.WriteString(text)
			} else if text, ok := contentMap["output_text"].(string); ok {
				b.WriteString(text)
			}
		}
	}
	return b.String()
}

func extractResponsesUsage(resp map[string]interface{}) provider.Usage {
	usage := provider.Usage{}
	usageMap, ok := resp["usage"].(map[string]interface{})
	if !ok {
		return usage
	}
	usage.PromptTokens = int(getFloatFromMap(usageMap, "input_tokens"))
	usage.CompletionTokens = int(getFloatFromMap(usageMap, "output_tokens"))
	usage.TotalTokens = int(getFloatFromMap(usageMap, "total_tokens"))
	usage.CachedReadTokens = extractCachedTokensFromUsageMap(usageMap)
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return usage
}

func extractCachedTokensFromUsageMap(usageMap map[string]interface{}) int {
	if direct := int(getFloatFromMap(usageMap, "cached_read_tokens")); direct > 0 {
		return direct
	}

	for _, key := range []string{"prompt_tokens_details", "input_tokens_details"} {
		detailMap, ok := usageMap[key].(map[string]interface{})
		if !ok {
			continue
		}
		if cached := int(getFloatFromMap(detailMap, "cached_tokens")); cached > 0 {
			return cached
		}
	}
	return 0
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
		return 0
	}
	return 0
}

// ValidateKey validates the API key
func (a *Adapter) ValidateKey(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return a.client.ValidateKey(ctx)
}

// Name returns the provider name
func (a *Adapter) Name() string {
	return "openai"
}

// SetClient allows setting a custom HTTP client (for testing)
func (a *Adapter) SetClient(client *Client) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.client = client
}

// DefaultModels returns the default models for OpenAI
func DefaultModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4-turbo-preview",
		"gpt-4",
		"gpt-4-32k",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
		"o1",
		"o1-mini",
		"o1-preview",
	}
}

// Factory creates a new OpenAI adapter from config
func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}
