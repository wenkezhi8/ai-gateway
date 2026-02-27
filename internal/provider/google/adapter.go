package google

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
		return nil, &provider.ProviderError{Code: http.StatusServiceUnavailable, Message: "google provider is disabled", Provider: "google"}
	}

	gReq := ConvertRequest(req)
	resp, err := a.client.DoGenerate(ctx, req.Model, gReq)
	if err != nil {
		return nil, &provider.ProviderError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to make request: %v", err), Provider: "google"}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, parseProviderError(resp.StatusCode, body)
	}

	var gResp GenerateContentResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		return nil, &provider.ProviderError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to decode response: %v", err), Provider: "google"}
	}

	if gResp.Error != nil {
		converted := ConvertResponse(&gResp, req.Model)
		if converted.Error != nil {
			return nil, converted.Error
		}
	}

	return ConvertResponse(&gResp, req.Model), nil
}

func (a *Adapter) StreamChat(ctx context.Context, req *provider.ChatRequest) (<-chan *provider.StreamChunk, error) {
	chunkChan := make(chan *provider.StreamChunk, 100)

	if !a.IsEnabled() {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{Code: http.StatusServiceUnavailable, Message: "google provider is disabled", Provider: "google"}
	}

	gReq := ConvertRequest(req)
	resp, err := a.client.DoStreamGenerate(ctx, req.Model, gReq)
	if err != nil {
		close(chunkChan)
		return chunkChan, &provider.ProviderError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to make stream request: %v", err), Provider: "google"}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		close(chunkChan)
		return chunkChan, parseProviderError(resp.StatusCode, body)
	}

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		currentEvent := "message"
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "event:") {
				currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				if currentEvent == "" {
					currentEvent = "message"
				}
				continue
			}

			if !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				chunkChan <- &provider.StreamChunk{Done: true}
				return
			}

			var gResp GenerateContentResponse
			if err := json.Unmarshal([]byte(data), &gResp); err != nil {
				continue
			}

			if currentEvent == "error" || gResp.Error != nil {
				code := http.StatusBadGateway
				msg := "provider stream error"
				if gResp.Error != nil {
					if gResp.Error.Code > 0 {
						code = gResp.Error.Code
					}
					if strings.TrimSpace(gResp.Error.Message) != "" {
						msg = gResp.Error.Message
					}
				}
				chunkChan <- &provider.StreamChunk{
					Error: &provider.ProviderError{
						Code:      code,
						Message:   msg,
						Provider:  "google",
						Retryable: isRetryableError(code),
					},
					Done: true,
				}
				return
			}

			done := false
			if len(gResp.Candidates) > 0 && gResp.Candidates[0].FinishReason != "" {
				done = true
			}

			chunk := ConvertStreamChunk(&gResp, req.Model, done)
			if chunk != nil {
				chunkChan <- chunk
			}
			if done {
				return
			}
			currentEvent = "message"
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
	return "google"
}

func DefaultModels() []string {
	return []string{
		"gemini-3.1-pro-preview",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}
}

func Factory(cfg *provider.ProviderConfig) provider.Provider {
	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels()
	}
	return NewAdapter(cfg)
}

func parseProviderError(statusCode int, body []byte) *provider.ProviderError {
	var wrapped struct {
		Error *APIError `json:"error"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Error != nil {
		code := wrapped.Error.Code
		if code == 0 {
			code = statusCode
		}
		return &provider.ProviderError{
			Code:      code,
			Message:   wrapped.Error.Message,
			Provider:  "google",
			Retryable: isRetryableError(code),
		}
	}

	return &provider.ProviderError{
		Code:      statusCode,
		Message:   string(body),
		Provider:  "google",
		Retryable: isRetryableError(statusCode),
	}
}
