package deepseek

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/provider"
)

type Adapter struct {
	*provider.BaseProvider
	client *Client
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
			Message:  "deepseek provider is disabled",
			Provider: "deepseek",
		}
	}

	dsReq := ConvertRequest(req)
	dsReq.Stream = false

	resp, err := a.client.DoRequest(ctx, "POST", "/v1/chat/completions", dsReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make request: %v", err),
			Provider: "deepseek",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(readErr.Error())
		}
		var errResp ChatResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			errResp.Error.StatusCode = resp.StatusCode
			return nil, ConvertResponse(&errResp).Error
		}
		return nil, &provider.ProviderError{
			Code:      resp.StatusCode,
			Message:   string(body),
			Provider:  "deepseek",
			Retryable: isRetryableError(resp.StatusCode),
		}
	}

	var dsResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&dsResp); err != nil {
		return nil, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to decode response: %v", err),
			Provider: "deepseek",
		}
	}

	return ConvertResponse(&dsResp), nil
}

func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusServiceUnavailable,
			Message:  "deepseek provider is disabled",
			Provider: "deepseek",
		}
	}

	dsReq := ConvertRequest(req)
	dsReq.Stream = true

	resp, err := a.client.DoStreamRequest(ctx, "POST", "/v1/chat/completions", dsReq) //nolint:bodyclose
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{
			Code:     http.StatusInternalServerError,
			Message:  fmt.Sprintf("failed to make stream request: %v", err),
			Provider: "deepseek",
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
			Provider:  "deepseek",
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

			// Handle both "data: " and "data:" formats
			var data string
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			} else if strings.HasPrefix(line, "data:") {
				data = strings.TrimPrefix(line, "data:")
			} else {
				continue
			}

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
	return "deepseek"
}

func DefaultModels() []string {
	return []string{
		"deepseek-chat",
		"deepseek-coder",
		"deepseek-reasoner",
	}
}

func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}
