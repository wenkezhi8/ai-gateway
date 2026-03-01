package vectordb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCollectionHandler_BasicCRUDRoutes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewService())

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

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/collections", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, want %d", w.Code, http.StatusOK)
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
}
