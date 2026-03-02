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

func TestIndexConfigHandler_Routes_ShouldHandleGetAndUpdate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo, initErr := NewSQLiteRepository(setupTestSQLite(t))
	if initErr != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", initErr)
	}

	now := time.Now().UTC()
	if err := repo.Create(t.Context(), &Collection{
		ID:              "col_cfg_1",
		Name:            "docs",
		Dimension:       768,
		DistanceMetric:  "cosine",
		IndexType:       "hnsw",
		HNSWM:           16,
		HNSWEFConstruct: 100,
		IVFNList:        1024,
		StorageBackend:  "qdrant",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       "tester",
	}); err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))
	api := r.Group("/api/admin")
	RegisterIndexConfigRoutes(api, h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/index-config/docs", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", w.Code, http.StatusOK)
	}

	badReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/index-config/docs", bytes.NewBufferString("{invalid"))
	badReq.Header.Set("Content-Type", "application/json")
	badResp := httptest.NewRecorder()
	r.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("PUT invalid body status = %d, want %d", badResp.Code, http.StatusBadRequest)
	}

	updateBody, marshalErr := json.Marshal(map[string]any{"index_type": "ivf", "ivf_nlist": 2048})
	if marshalErr != nil {
		t.Fatalf("json.Marshal(updateBody) error = %v", marshalErr)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/index-config/docs", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	r.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d", updateResp.Code, http.StatusOK)
	}
}

func TestRBACHandler_Routes_ShouldHandleCRUD(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo, initErr := NewSQLiteRepository(setupTestSQLite(t))
	if initErr != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", initErr)
	}

	api := r.Group("/api/admin")
	RegisterRBACRoutes(api, NewRBACService(repo))

	createBody, marshalErr := json.Marshal(map[string]any{"api_key": "rbac-key-1", "role": "reader"})
	if marshalErr != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", marshalErr)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/permissions", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", createResp.Code, http.StatusCreated)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/permissions", http.NoBody)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", listResp.Code, http.StatusOK)
	}
	data := decodeJSONBody(t, listResp)
	ok, okType := data["success"].(bool)
	if !okType || !ok {
		t.Fatalf("GET success = %v, want true", data["success"])
	}

	invalidDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/permissions/abc", http.NoBody)
	invalidDeleteResp := httptest.NewRecorder()
	r.ServeHTTP(invalidDeleteResp, invalidDeleteReq)
	if invalidDeleteResp.Code != http.StatusBadRequest {
		t.Fatalf("DELETE invalid id status = %d, want %d", invalidDeleteResp.Code, http.StatusBadRequest)
	}

	keys, listErr := repo.ListVectorAPIKeys(t.Context())
	if listErr != nil {
		t.Fatalf("repo.ListVectorAPIKeys() error = %v", listErr)
	}
	if len(keys) != 1 {
		t.Fatalf("repo.ListVectorAPIKeys() len=%d, want 1", len(keys))
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/permissions/1", http.NoBody)
	deleteResp := httptest.NewRecorder()
	r.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", deleteResp.Code, http.StatusOK)
	}

	deleteMissingReq := httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/permissions/1", http.NoBody)
	deleteMissingResp := httptest.NewRecorder()
	r.ServeHTTP(deleteMissingResp, deleteMissingReq)
	if deleteMissingResp.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing status = %d, want %d", deleteMissingResp.Code, http.StatusNotFound)
	}
}

func TestRBACHandler_RegisterRoutesWithNilService_ShouldSkipRegistration(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/admin")
	RegisterRBACRoutes(api, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/permissions", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVisualizationHandler_Routes_ShouldHandleSuccessAndValidation(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{getResp: &Collection{Name: "docs", Dimension: 3}}, &mockSearchBackend{
		searchResp: []SearchResult{{ID: "p1", Score: 0.8, Payload: map[string]any{"title": "doc", "x": 0.1, "y": 0.2}}},
	}))
	api := r.Group("/api/admin")
	RegisterVisualizationRoutes(api, h)

	successReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/visualization/scatter?collection_name=docs&sample_size=2", http.NoBody)
	successResp := httptest.NewRecorder()
	r.ServeHTTP(successResp, successReq)
	if successResp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", successResp.Code, http.StatusOK)
	}

	badReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/visualization/scatter", http.NoBody)
	badResp := httptest.NewRecorder()
	r.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("GET missing collection status = %d, want %d", badResp.Code, http.StatusBadRequest)
	}
}

func TestBackupHandler_RestoreAndRetryRoutes_ShouldWork(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo, initErr := NewSQLiteRepository(setupTestSQLite(t))
	if initErr != nil {
		t.Fatalf("NewSQLiteRepository() error = %v", initErr)
	}
	now := time.Now().UTC()
	if err := repo.Create(t.Context(), &Collection{
		ID:             "col_backup_1",
		Name:           "docs",
		Dimension:      384,
		DistanceMetric: "cosine",
		IndexType:      "hnsw",
		StorageBackend: "qdrant",
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      "tester",
	}); err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))
	api := r.Group("/api/admin")
	RegisterBackupRoutes(api, h)

	createBody, marshalErr := json.Marshal(map[string]any{"collection_name": "docs", "snapshot_name": "snap-1"})
	if marshalErr != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", marshalErr)
	}
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/backups", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("POST backup status = %d, want %d", createResp.Code, http.StatusCreated)
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/backups/1/restore", http.NoBody)
	restoreResp := httptest.NewRecorder()
	r.ServeHTTP(restoreResp, restoreReq)
	if restoreResp.Code != http.StatusOK {
		t.Fatalf("POST restore status = %d, want %d", restoreResp.Code, http.StatusOK)
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/backups/1/retry", http.NoBody)
	retryResp := httptest.NewRecorder()
	r.ServeHTTP(retryResp, retryReq)
	if retryResp.Code != http.StatusOK {
		t.Fatalf("POST retry status = %d, want %d", retryResp.Code, http.StatusOK)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/backups/0/retry", http.NoBody)
	badResp := httptest.NewRecorder()
	r.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusBadRequest {
		t.Fatalf("POST retry invalid id status = %d, want %d", badResp.Code, http.StatusBadRequest)
	}
}
