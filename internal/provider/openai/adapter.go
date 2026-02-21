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
		body, _ := io.ReadAll(resp.Body)
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
	resp, err := a.client.DoStreamRequest(ctx, "POST", "/chat/completions", openaiReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "openai",
		}
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
