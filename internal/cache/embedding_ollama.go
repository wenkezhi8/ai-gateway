//nolint:dupl,gocritic // Endpoint compatibility requires mirrored request paths and signatures.
package cache

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
	OllamaEndpointModeAuto       = "auto"
	OllamaEndpointModeEmbed      = "embed"
	OllamaEndpointModeEmbeddings = "embeddings"
)

type OllamaEmbeddingConfig struct {
	BaseURL      string
	Model        string
	Timeout      time.Duration
	EndpointMode string
}

type OllamaEmbeddingService struct {
	cfg        OllamaEmbeddingConfig
	httpClient *http.Client
}

func NewOllamaEmbeddingService(cfg OllamaEmbeddingConfig) *OllamaEmbeddingService {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "nomic-embed-text"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	mode := normalizeOllamaEndpointMode(cfg.EndpointMode)

	return &OllamaEmbeddingService{
		cfg: OllamaEmbeddingConfig{
			BaseURL:      baseURL,
			Model:        model,
			Timeout:      timeout,
			EndpointMode: mode,
		},
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *OllamaEmbeddingService) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("embedding text is empty")
	}

	mode := normalizeOllamaEndpointMode(s.cfg.EndpointMode)
	switch mode {
	case OllamaEndpointModeEmbed:
		return s.callEmbedEndpoint(ctx, text)
	case OllamaEndpointModeEmbeddings:
		return s.callEmbeddingsEndpoint(ctx, text)
	default:
		vec, err := s.callEmbedEndpoint(ctx, text)
		if err == nil {
			return vec, nil
		}
		if isOllamaEmbeddingTimeout(err) {
			return nil, err
		}
		return s.callEmbeddingsEndpoint(ctx, text)
	}
}

func (s *OllamaEmbeddingService) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vec, err := s.GetEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		results = append(results, vec)
	}
	return results, nil
}

func (s *OllamaEmbeddingService) callEmbedEndpoint(ctx context.Context, text string) ([]float64, error) {
	payload := map[string]any{
		"model": s.cfg.Model,
		"input": text,
	}
	respBody, statusCode, err := s.post(ctx, "/api/embed", payload)
	if err != nil {
		return nil, err
	}
	if statusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("ollama /api/embed status %d: %s", statusCode, strings.TrimSpace(string(respBody)))
	}
	vec, err := parseOllamaEmbeddingPayload(respBody)
	if err != nil {
		return nil, fmt.Errorf("parse /api/embed response: %w", err)
	}
	return vec, nil
}

func (s *OllamaEmbeddingService) callEmbeddingsEndpoint(ctx context.Context, text string) ([]float64, error) {
	payload := map[string]any{
		"model":  s.cfg.Model,
		"prompt": text,
	}
	respBody, statusCode, err := s.post(ctx, "/api/embeddings", payload)
	if err != nil {
		return nil, err
	}
	if statusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("ollama /api/embeddings status %d: %s", statusCode, strings.TrimSpace(string(respBody)))
	}
	vec, err := parseOllamaEmbeddingPayload(respBody)
	if err != nil {
		return nil, fmt.Errorf("parse /api/embeddings response: %w", err)
	}
	return vec, nil
}

func (s *OllamaEmbeddingService) post(ctx context.Context, path string, payload map[string]any) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal ollama embedding payload: %w", err)
	}
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call ollama embedding: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read ollama embedding response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

func parseOllamaEmbeddingPayload(respBody []byte) ([]float64, error) {
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(respBody, &generic); err != nil {
		return nil, err
	}

	if raw, ok := generic["embedding"]; ok {
		var vec []float64
		if err := json.Unmarshal(raw, &vec); err == nil && len(vec) > 0 {
			return vec, nil
		}
	}

	if raw, ok := generic["embeddings"]; ok {
		var vectors [][]float64
		if err := json.Unmarshal(raw, &vectors); err == nil && len(vectors) > 0 && len(vectors[0]) > 0 {
			return vectors[0], nil
		}
		var single []float64
		if err := json.Unmarshal(raw, &single); err == nil && len(single) > 0 {
			return single, nil
		}
	}

	if raw, ok := generic["data"]; ok {
		var rows []struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.Unmarshal(raw, &rows); err == nil && len(rows) > 0 && len(rows[0].Embedding) > 0 {
			return rows[0].Embedding, nil
		}
	}

	return nil, fmt.Errorf("embedding vector not found")
}

func normalizeOllamaEndpointMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case OllamaEndpointModeEmbed, OllamaEndpointModeEmbeddings:
		return normalized
	default:
		return OllamaEndpointModeAuto
	}
}

func isOllamaEmbeddingTimeout(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "client.timeout")
}
