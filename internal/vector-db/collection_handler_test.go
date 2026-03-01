//nolint:errcheck // Type assertions in tests intentionally validate shape.
package vectordb

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCollectionHandler_BasicCRUDRoutes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterCollectionRoutes(group, h)

	createBody := map[string]any{
		"name":            "team-docs",
		"description":     "Team documents",
		"dimension":       1536,
		"distance_metric": "cosine",
	}
	createPayload, err := json.Marshal(createBody)
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/collections", bytes.NewReader(createPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", w.Code, http.StatusCreated)
	}
	createResp := decodeJSONBody(t, w)
	if ok, _ := createResp["success"].(bool); !ok {
		t.Fatalf("POST success = %v, want true", createResp["success"])
	}
	data, _ := createResp["data"].(map[string]any)
	if data["name"] != "team-docs" {
		t.Fatalf("POST data.name = %v, want team-docs", data["name"])
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/collections", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, want %d", w.Code, http.StatusOK)
	}
	listResp := decodeJSONBody(t, w)
	if ok, _ := listResp["success"].(bool); !ok {
		t.Fatalf("GET list success = %v, want true", listResp["success"])
	}
	listData, _ := listResp["data"].(map[string]any)
	if _, ok := listData["collections"].([]any); !ok {
		t.Fatalf("GET list data.collections missing or invalid: %T", listData["collections"])
	}
	if _, ok := listData["total"].(float64); !ok {
		t.Fatalf("GET list data.total missing or invalid: %T", listData["total"])
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/collections/team-docs", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET detail status = %d, want %d", w.Code, http.StatusOK)
	}

	updateBody := map[string]any{"status": "inactive"}
	updatePayload, err := json.Marshal(updateBody)
	if err != nil {
		t.Fatalf("json.Marshal(updateBody) error = %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/collections/team-docs", bytes.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d", w.Code, http.StatusOK)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/collections/team-docs", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", w.Code, http.StatusOK)
	}
	deleteResp := decodeJSONBody(t, w)
	if ok, _ := deleteResp["success"].(bool); !ok {
		t.Fatalf("DELETE success = %v, want true", deleteResp["success"])
	}
}

func TestCollectionHandler_Create_WhenBackendUnavailable_ShouldReturn503(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{createErr: errors.New("qdrant down")}))

	group := r.Group("/api/admin")
	RegisterCollectionRoutes(group, h)

	createBody := map[string]any{
		"name":      "team-docs",
		"dimension": 1536,
	}
	payload, err := json.Marshal(createBody)
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/collections", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	errResp := decodeJSONBody(t, w)
	if ok, _ := errResp["success"].(bool); ok {
		t.Fatalf("POST success = %v, want false", errResp["success"])
	}
	if msg, _ := errResp["error"].(string); msg == "" {
		t.Fatalf("POST error message should not be empty")
	}
}

func TestCollectionHandler_Create_WhenCollectionExists_ShouldReturn409(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{createErr: errors.New("already exists")}))

	group := r.Group("/api/admin")
	RegisterCollectionRoutes(group, h)

	createBody := map[string]any{
		"name":      "team-docs",
		"dimension": 1536,
	}
	payload, err := json.Marshal(createBody)
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/collections", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("POST status = %d, want %d", w.Code, http.StatusConflict)
	}
	errResp := decodeJSONBody(t, w)
	if ok, _ := errResp["success"].(bool); ok {
		t.Fatalf("POST success = %v, want false", errResp["success"])
	}
	if msg, _ := errResp["error"].(string); msg == "" {
		t.Fatalf("POST error message should not be empty")
	}
}

func TestCollectionHandler_EmptyCollection_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{getResp: &Collection{Name: "team-docs", Dimension: 1536, DistanceMetric: "cosine"}}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterCollectionRoutes(group, h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/collections/team-docs/empty", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST empty status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeJSONBody(t, w)
	if ok, _ := resp["success"].(bool); !ok {
		t.Fatalf("POST empty success = %v, want true", resp["success"])
	}
}

func TestCollectionHandler_EmptyCollection_WhenNotFound_ShouldReturn404(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{getErr: ErrCollectionNotFound}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterCollectionRoutes(group, h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/collections/missing/empty", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("POST empty status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func decodeJSONBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	resp := map[string]any{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v", err)
	}
	return resp
}
