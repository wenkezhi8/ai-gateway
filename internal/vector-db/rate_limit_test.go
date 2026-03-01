package vectordb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSearchHandler_WhenRateLimited_ShouldReturn429(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{
		searchResp: []SearchResult{{ID: "vec-1", Score: 0.9, Payload: map[string]any{"title": "doc"}}},
	}))

	api := r.Group("/api/v1")
	limiter := NewVectorSearchRateLimiter(1, time.Minute)
	RegisterVectorSearchRoutesWithRBAC(api, h, limiter.Middleware())

	body := map[string]any{
		"top_k":  1,
		"vector": []float32{0.1, 0.2},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(payload))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-API-Key", "test-key")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", w1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-API-Key", "test-key")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}
