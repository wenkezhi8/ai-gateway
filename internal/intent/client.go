package intent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrClientDisabled = errors.New("intent engine client disabled")
)

// Config defines local intent-engine client configuration.
type Config struct {
	Enabled           bool
	BaseURL           string
	Timeout           time.Duration
	Language          string
	ExpectedDimension int
}

// Client is a minimal HTTP client for local intent+embedding inference service.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates an intent-engine client with safe defaults.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://127.0.0.1:18566"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	if cfg.Language == "" {
		cfg.Language = "zh-CN"
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Enabled reports whether intent-engine integration is enabled.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled
}

// Infer sends one-shot query to local intent engine and validates output.
func (c *Client) Infer(ctx context.Context, query, contextText string) (*EmbeddingResult, error) {
	if c == nil || !c.cfg.Enabled {
		return nil, ErrClientDisabled
	}

	payload := EmbeddingRequest{
		Query:   strings.TrimSpace(query),
		Context: strings.TrimSpace(contextText),
		Lang:    c.cfg.Language,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal intent request: %w", err)
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1/intent-embed"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create intent request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call intent engine: %w", err)
	}
	defer resp.Body.Close()

	result, err := decodeInferResponse(resp)
	if err != nil {
		return nil, err
	}

	if err := c.validateEmbeddingResult(result); err != nil {
		return nil, err
	}

	return result, nil
}

// Health checks intent-engine health endpoint.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	if c == nil || !c.cfg.Enabled {
		return map[string]any{"enabled": false, "healthy": false, "message": "intent engine disabled"}, nil
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call health endpoint: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read health response: %w", err)
	}
	result := map[string]any{
		"enabled": true,
		"healthy": resp.StatusCode >= 200 && resp.StatusCode < 300,
		"status":  resp.StatusCode,
	}

	if len(payload) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(payload, &parsed); err == nil {
			for k, v := range parsed {
				result[k] = v
			}
		} else {
			result["raw"] = strings.TrimSpace(string(payload))
		}
	}
	return result, nil
}

func decodeInferResponse(resp *http.Response) (*EmbeddingResult, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read intent response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("intent engine status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result EmbeddingResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode intent response: %w", err)
	}

	if result.EmbeddingDim <= 0 {
		result.EmbeddingDim = len(result.Embedding)
	}
	if result.Slots == nil {
		result.Slots = map[string]string{}
	}

	return &result, nil
}

func (c *Client) validateEmbeddingResult(result *EmbeddingResult) error {
	if c.cfg.ExpectedDimension > 0 && result.EmbeddingDim != c.cfg.ExpectedDimension {
		return fmt.Errorf("intent embedding dimension mismatch: expected %d got %d", c.cfg.ExpectedDimension, result.EmbeddingDim)
	}
	if result.EmbeddingDim > 0 && len(result.Embedding) != result.EmbeddingDim {
		return fmt.Errorf("intent embedding payload invalid: dim=%d len=%d", result.EmbeddingDim, len(result.Embedding))
	}

	return nil
}
