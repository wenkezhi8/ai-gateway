package volcengine

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

// Adapter implements the Provider interface for Volcengine (火山方舟)
type Adapter struct {
	*provider.BaseProvider
	client *Client
	mu     sync.RWMutex
}

// NewAdapter creates a new Volcengine adapter
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

// Chat sends a chat completion request to Volcengine
func (a *Adapter) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	if !a.IsEnabled() {
		return nil, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "volcengine provider is disabled",
			Provider: "volcengine",
		}
	}

	// Convert to Volcengine request format
	volcReq := ConvertRequest(req)
	volcReq.Stream = false

	// Make the API call
	resp, err := a.client.DoRequest(ctx, "POST", "/chat/completions", volcReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make request: %v", err),
			Provider: "volcengine",
		}
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp ChatResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			return nil, &provider.ProviderError{
				Code:      errResp.Error.Code,
				Message:   errResp.Error.Message,
				Type:      errResp.Error.Type,
				Provider:  "volcengine",
				Retryable: isRetryableError(errResp.Error.Code),
			}
		}
		return nil, &provider.ProviderError{
			Code:      resp.StatusCode,
			Message:   string(body),
			Provider:  "volcengine",
			Retryable: isRetryableError(resp.StatusCode),
		}
	}

	// Parse response
	var volcResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&volcResp); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode response: %v", err),
			Provider: "volcengine",
		}
	}

	return ConvertResponse(&volcResp), nil
}

// StreamChat sends a streaming chat completion request to Volcengine
func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "volcengine provider is disabled",
			Provider: "volcengine",
		}
	}

	// Convert to Volcengine request format
	volcReq := ConvertRequest(req)
	volcReq.Stream = true

	// Make the streaming API call
	resp, err := a.client.DoStreamRequest(ctx, "POST", "/chat/completions", volcReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "volcengine",
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

			// Handle SSE format
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

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
	return "volcengine"
}

// SetClient allows setting a custom HTTP client (for testing)
func (a *Adapter) SetClient(client *Client) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.client = client
}

// DefaultModels returns the default models for Volcengine
func DefaultModels() []string {
	return []string{
		"doubao-pro-32k",
		"doubao-pro-128k",
		"doubao-lite-32k",
		"doubao-lite-128k",
		"doubao-1.5-pro-32k",
		"doubao-1.5-pro-256k",
		"doubao-1.5-lite-32k",
		"skylark2-pro-4k",
		"skylark2-pro-32k",
		"skylark2-lite-4k",
	}
}

// Factory creates a new Volcengine adapter from config
func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}
