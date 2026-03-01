package vectordb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBackupHandler_Routes_ShouldWork(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))

	api := r.Group("/api/admin")
	RegisterBackupRoutes(api, h)

	createBody, err := json.Marshal(map[string]any{"collection_name": "docs", "snapshot_name": "snapshot-001"})
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/backups", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", createResp.Code, http.StatusCreated)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/backups", http.NoBody)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", listResp.Code, http.StatusOK)
	}
}
