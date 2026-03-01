// Package cache provides embedding service for semantic caching.
// 改动点: 新增嵌入向量服务，用于语义缓存.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EmbeddingProvider defines the interface for embedding services.
type EmbeddingProvider interface {
	GetEmbedding(ctx context.Context, text string) ([]float64, error)
	GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
}

// EmbeddingService provides text embedding functionality.
type EmbeddingService struct {
	mu       sync.RWMutex
	provider string
	apiKey   string
	baseURL  string
	model    string
	cache    map[string][]float64 // simple in-memory cache for embeddings
}

// EmbeddingRequest represents an embedding API request.
type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

// EmbeddingResponse represents an embedding API response.
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

var embeddingLogger = logrus.WithField("component", "embedding_service")

// NewEmbeddingService creates a new embedding service.
func NewEmbeddingService(provider, apiKey, baseURL, model string) *EmbeddingService {
	if model == "" {
		model = "text-embedding-3-small"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &EmbeddingService{
		provider: provider,
		apiKey:   apiKey,
		baseURL:  baseURL,
		model:    model,
		cache:    make(map[string][]float64),
	}
}

// GetEmbedding returns the embedding vector for a text.
// 改动点: 获取文本的嵌入向量，带缓存.
func (s *EmbeddingService) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Check cache first.
	cacheKey := s.hashText(text)
	s.mu.RLock()
	if vec, ok := s.cache[cacheKey]; ok {
		s.mu.RUnlock()
		return vec, nil
	}
	s.mu.RUnlock()

	// Truncate long texts (OpenAI has 8191 token limit).
	if len(text) > 8000 {
		text = text[:8000]
	}

	// Call embedding API.
	req := EmbeddingRequest{
		Input: text,
		Model: s.model,
	}

	resp, err := s.callEmbeddingAPI(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding API error: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("embedding error: %s", resp.Error.Message)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	embedding := resp.Data[0].Embedding

	// Cache the result.
	s.mu.Lock()
	s.cache[cacheKey] = embedding
	s.mu.Unlock()

	embeddingLogger.WithFields(logrus.Fields{
		"text_len":      len(text),
		"embedding_len": len(embedding),
		"model":         s.model,
	}).Debug("Generated embedding")

	return embedding, nil
}

// GetEmbeddings returns embeddings for multiple texts.
func (s *EmbeddingService) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		emb, err := s.GetEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}

// callEmbeddingAPI calls the embedding API.
func (s *EmbeddingService) callEmbeddingAPI(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/embeddings", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp EmbeddingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// hashText generates a hash for caching.
func (s *EmbeddingService) hashText(text string) string {
	hash := sha256.Sum256([]byte(text + s.model))
	return hex.EncodeToString(hash[:])
}

// ClearCache clears the embedding cache.
func (s *EmbeddingService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string][]float64)
}

// SimpleEmbedding generates a simple embedding without API call.
// 改动点: 简单嵌入向量生成，用于无 API key 时的降级.
func SimpleEmbedding(text string, dimension int) []float64 {
	if dimension == 0 {
		dimension = 1536
	}

	// Use character frequency as simple embedding.
	vec := make([]float64, dimension)
	text = strings.ToLower(text)

	for i, ch := range text {
		idx := int(ch) % dimension
		vec[idx] += 1.0 / float64(len(text))
		if i < dimension {
			vec[i] += float64(ch) / 255.0 / float64(len(text))
		}
	}

	// Normalize.
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	norm := 0.0
	if sum > 0 {
		norm = 1.0 / (sum * 2)
	}
	for i := range vec {
		vec[i] *= norm
	}

	return vec
}

// MockEmbeddingService provides mock embeddings for testing.
type MockEmbeddingService struct {
	Dimension int
}

// NewMockEmbeddingService creates a mock embedding service.
func NewMockEmbeddingService(dimension int) *MockEmbeddingService {
	if dimension == 0 {
		dimension = 1536
	}
	return &MockEmbeddingService{Dimension: dimension}
}

// GetEmbedding returns a mock embedding.
func (s *MockEmbeddingService) GetEmbedding(_ context.Context, text string) ([]float64, error) {
	return SimpleEmbedding(text, s.Dimension), nil
}

// GetEmbeddings returns mock embeddings.
func (s *MockEmbeddingService) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		emb, err := s.GetEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}
