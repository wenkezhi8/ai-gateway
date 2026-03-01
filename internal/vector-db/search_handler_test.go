package vectordb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSearchHandler_SearchRoute_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{
		searchResp: []SearchResult{{ID: "vec-1", Score: 0.9, Payload: map[string]any{"title": "doc"}}, {ID: "vec-2", Score: 0.8, Payload: map[string]any{"title": "doc2"}}},
	}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k":      2,
		"min_score":  0.7,
		"vector":     []float32{0.1, 0.2, 0.3},
		"collection": "ignored",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST search status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSearchHandler_RecommendRoute_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{
		searchResp: []SearchResult{{ID: "vec-1", Score: 0.95, Payload: map[string]any{"title": "rec"}}},
	}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k":  1,
		"vector": []float32{0.1, 0.2, 0.3},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/recommend", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST recommend status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSearchHandler_GetVectorByID_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{
		getByIDResp: &SearchResult{ID: "vec-1", Score: 0, Payload: map[string]any{"title": "doc"}},
	}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vector/collections/docs/vectors/vec-1", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET vector status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSearchHandler_SearchRoute_WhenTextQueryProvided_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k": 1,
		"text":  "hello",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST search status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSearchHandler_SearchRoute_WhenBackendUnavailable_ShouldReturn503(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, nil))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k":  1,
		"vector": []float32{0.1, 0.2},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST search status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestSearchHandler_SearchRoute_WhenCollectionNotFound_ShouldReturn404(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{getErr: ErrCollectionNotFound}, &mockSearchBackend{}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k":  1,
		"vector": []float32{0.1, 0.2},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/search", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("POST search status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSearchHandler_RecommendRoute_WhenTextQueryProvided_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, &mockSearchBackend{}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k": 1,
		"text":  "hello",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/recommend", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST recommend status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSearchHandler_RecommendRoute_WhenBackendUnavailable_ShouldReturn503(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, nil))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	body := map[string]any{
		"top_k":  1,
		"vector": []float32{0.1, 0.2},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vector/collections/docs/recommend", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST recommend status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestSearchHandler_GetVectorByID_WhenCollectionNotFound_ShouldReturn404(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{getErr: ErrCollectionNotFound}, &mockSearchBackend{}))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vector/collections/docs/vectors/vec-1", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GET vector status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSearchHandler_GetVectorByID_WhenBackendUnavailable_ShouldReturn503(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSearchHandler(NewServiceWithDeps(&mockRepo{}, nil))

	api := r.Group("/api/v1")
	RegisterVectorSearchRoutes(api, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vector/collections/docs/vectors/vec-1", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET vector status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
