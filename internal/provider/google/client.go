package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
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

func (c *Client) buildGeneratePath(model string) string {
	return fmt.Sprintf("/models/%s:generateContent?key=%s", model, c.apiKey)
}

func (c *Client) buildStreamPath(model string) string {
	return fmt.Sprintf("/models/%s:streamGenerateContent?alt=sse&key=%s", model, c.apiKey)
}

func (c *Client) buildModelsPath() string {
	return fmt.Sprintf("/models?key=%s", c.apiKey)
}

func (c *Client) DoGenerate(ctx context.Context, model string, body interface{}) (*http.Response, error) {
	return c.doJSONRequest(ctx, http.MethodPost, c.buildGeneratePath(model), body, false)
}

func (c *Client) DoStreamGenerate(ctx context.Context, model string, body interface{}) (*http.Response, error) {
	return c.doJSONRequest(ctx, http.MethodPost, c.buildStreamPath(model), body, true)
}

func (c *Client) ValidateKey(ctx context.Context) bool {
	resp, err := c.doJSONRequest(ctx, http.MethodGet, c.buildModelsPath(), nil, false)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) doJSONRequest(ctx context.Context, method, path string, body interface{}, stream bool) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(b)
	}

	url := strings.TrimRight(c.baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
	}

	return c.httpClient.Do(req)
}
