//nolint:godot
package claude

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

// Adapter implements the Provider interface for Anthropic Claude
type Adapter struct {
	*provider.BaseProvider
	client *Client
	mu     sync.RWMutex
}

// NewAdapter creates a new Claude adapter
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

// Chat sends a chat completion request to Claude
func (a *Adapter) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	if !a.IsEnabled() {
		return nil, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "claude provider is disabled",
			Provider: "claude",
		}
	}

	// Convert to Claude request format
	claudeReq := ConvertRequest(req)
	claudeReq.Stream = false

	// Make the API call
	resp, err := a.client.DoRequest(ctx, "POST", "/messages", claudeReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make request: %v", err),
			Provider: "claude",
		}
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(readErr.Error())
		}
		var errResp MessagesResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			return nil, ConvertToProviderResponse(&errResp).Error
		}
		return nil, &provider.ProviderError{
			Code:      resp.StatusCode,
			Message:   string(body),
			Provider:  "claude",
			Retryable: isRetryableError(resp.StatusCode),
		}
	}

	// Parse response
	var claudeResp MessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode response: %v", err),
			Provider: "claude",
		}
	}

	return ConvertResponse(&claudeResp), nil
}

// StreamChat sends a streaming chat completion request to Claude
func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "claude provider is disabled",
			Provider: "claude",
		}
	}

	// Convert to Claude request format
	claudeReq := ConvertRequest(req)
	claudeReq.Stream = true

	// Make the streaming API call
	resp, err := a.client.DoStreamRequest(ctx, "POST", "/messages", claudeReq) //nolint:bodyclose
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "claude",
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
			Provider:  "claude",
			Retryable: isRetryableError(resp.StatusCode),
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

			// Parse the JSON event
			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			// Convert event to chunk
			chunk := ConvertStreamEvent(&event, req.Model)
			if chunk != nil {
				chunkChan <- chunk
				if chunk.Done {
					return
				}
			}
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
	return "claude"
}

// SetClient allows setting a custom HTTP client (for testing)
func (a *Adapter) SetClient(client *Client) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.client = client
}

// DefaultModels returns the default models for Claude
func DefaultModels() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",
	}
}

// Factory creates a new Claude adapter from config
func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}
