package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaEmbeddingService_GetEmbedding_AutoEndpointFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embed":
			// Simulate older/unsupported endpoint on some runtimes.
			http.Error(w, "not found", http.StatusNotFound)
		case "/api/embeddings":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"embedding": []float64{0.11, 0.22, 0.33},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewOllamaEmbeddingService(OllamaEmbeddingConfig{
		BaseURL:      server.URL,
		Model:        "nomic-embed-text",
		Timeout:      500 * time.Millisecond,
		EndpointMode: OllamaEndpointModeAuto,
	})

	vec, err := svc.GetEmbedding(context.Background(), "向量检索测试")
	if err != nil {
		t.Fatalf("expected embedding success with fallback endpoint, got err=%v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected embedding size 3, got %d", len(vec))
	}
}

func TestOllamaEmbeddingService_GetEmbedding_ParseEmbedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"embeddings": [][]float64{
				{0.7, 0.8, 0.9, 1.0},
			},
		})
	}))
	defer server.Close()

	svc := NewOllamaEmbeddingService(OllamaEmbeddingConfig{
		BaseURL:      server.URL,
		Model:        "nomic-embed-text",
		Timeout:      500 * time.Millisecond,
		EndpointMode: OllamaEndpointModeEmbed,
	})

	vec, err := svc.GetEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected embedding success, got err=%v", err)
	}
	if len(vec) != 4 {
		t.Fatalf("expected embedding size 4, got %d", len(vec))
	}
}

