package vectordb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestImportJobHandler_Routes_ShouldHandleCRUDAndActions(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	now := time.Now().UTC()
	repo := &mockRepo{getResp: &Collection{Name: "docs", ID: "col_docs", Dimension: 3}}
	backend := &mockBackend{}
	h := NewCollectionHandler(NewServiceWithDeps(repo, backend))

	api := r.Group("/api/admin")
	RegisterImportJobRoutes(api, h)

	filePath := filepath.Join(t.TempDir(), "vectors.json")
	content := []byte(`[{"id":"v1","vector":[0.1,0.2,0.3],"payload":{"title":"doc1"}}]`)
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	createBody, err := json.Marshal(map[string]any{
		"collection_name": "docs",
		"file_name":       "vectors.json",
		"file_path":       filePath,
		"file_size":       len(content),
		"total_records":   1,
		"max_retries":     2,
		"created_by":      "tester",
	})
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", createResp.Code, http.StatusCreated)
	}

	created := decodeJSONBody(t, createResp)
	data, ok := created["data"].(map[string]any)
	if !ok {
		t.Fatalf("create response data type = %T, want map[string]any", created["data"])
	}
	jobID, ok := data["id"].(string)
	if !ok {
		t.Fatalf("create response id type = %T, want string", data["id"])
	}
	if jobID == "" {
		t.Fatal("create response missing id")
	}

	summaryReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/summary?collection_name=docs", http.NoBody)
	summaryResp := httptest.NewRecorder()
	r.ServeHTTP(summaryResp, summaryReq)
	if summaryResp.Code != http.StatusOK {
		t.Fatalf("GET summary status = %d, want %d", summaryResp.Code, http.StatusOK)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs?collection_name=docs&status=pending", http.NoBody)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, want %d", listResp.Code, http.StatusOK)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/"+jobID, http.NoBody)
	getResp := httptest.NewRecorder()
	r.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("GET detail status = %d, want %d", getResp.Code, http.StatusOK)
	}

	updatePayload, err := json.Marshal(map[string]any{"status": "running", "processed_records": 1})
	if err != nil {
		t.Fatalf("json.Marshal(updatePayload) error = %v", err)
	}
	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/import-jobs/"+jobID+"/status", bytes.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	r.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d", updateResp.Code, http.StatusOK)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/"+jobID+"/run", http.NoBody)
	runResp := httptest.NewRecorder()
	r.ServeHTTP(runResp, runReq)
	if runResp.Code != http.StatusOK {
		t.Fatalf("POST run status = %d, want %d", runResp.Code, http.StatusOK)
	}

	if err := repo.UpdateImportJobStatus(t.Context(), jobID, &UpdateImportJobStatusRequest{Status: ImportJobStatusFailed}); err != nil {
		t.Fatalf("repo.UpdateImportJobStatus(failed) error = %v", err)
	}
	retryReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/"+jobID+"/retry", http.NoBody)
	retryResp := httptest.NewRecorder()
	r.ServeHTTP(retryResp, retryReq)
	if retryResp.Code != http.StatusOK {
		t.Fatalf("POST retry status = %d, want %d", retryResp.Code, http.StatusOK)
	}

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/"+jobID+"/cancel", http.NoBody)
	cancelResp := httptest.NewRecorder()
	r.ServeHTTP(cancelResp, cancelReq)
	if cancelResp.Code != http.StatusBadRequest {
		t.Fatalf("POST cancel status = %d, want %d", cancelResp.Code, http.StatusBadRequest)
	}

	if err := repo.CreateAuditLog(t.Context(), &AuditLog{UserID: "u", Action: "import_run_failed", ResourceType: "import_job", ResourceID: jobID, CreatedAt: now}); err != nil {
		t.Fatalf("repo.CreateAuditLog() error = %v", err)
	}
	errorsReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/"+jobID+"/errors?limit=-1&offset=-1", http.NoBody)
	errorsResp := httptest.NewRecorder()
	r.ServeHTTP(errorsResp, errorsReq)
	if errorsResp.Code != http.StatusOK {
		t.Fatalf("GET errors status = %d, want %d", errorsResp.Code, http.StatusOK)
	}

	retryFailedReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/retry-failed?limit=-2", http.NoBody)
	retryFailedResp := httptest.NewRecorder()
	r.ServeHTTP(retryFailedResp, retryFailedReq)
	if retryFailedResp.Code != http.StatusOK {
		t.Fatalf("POST retry-failed status = %d, want %d", retryFailedResp.Code, http.StatusOK)
	}
}

func TestImportJobHandler_InvalidRequests_ShouldReturnBadRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))
	api := r.Group("/api/admin")
	RegisterImportJobRoutes(api, h)

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs", bytes.NewBufferString("{bad"))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusBadRequest {
		t.Fatalf("POST invalid create status = %d, want %d", createResp.Code, http.StatusBadRequest)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/import-jobs/1/status", bytes.NewBufferString("{bad"))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	r.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusBadRequest {
		t.Fatalf("PUT invalid status payload code = %d, want %d", updateResp.Code, http.StatusBadRequest)
	}
}
