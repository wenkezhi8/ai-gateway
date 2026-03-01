//nolint:errcheck // Type assertions in tests intentionally validate response shape.
package vectordb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestVectorDBService_CreateImportJob_ShouldPersist(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		getResp: &Collection{ID: "col_1", Name: "docs"},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	job, err := svc.CreateImportJob(context.Background(), &CreateImportJobRequest{
		CollectionName: "docs",
		FileName:       "docs.json",
		FilePath:       "/tmp/docs.json",
		FileSize:       1024,
		TotalRecords:   100,
		CreatedBy:      "tester",
	})
	if err != nil {
		t.Fatalf("CreateImportJob() error = %v", err)
	}
	if job.CollectionID != "col_1" || job.Status != ImportJobStatusPending {
		t.Fatalf("CreateImportJob() job=%+v", job)
	}
}

func TestVectorDBService_GetImportJob_WhenMissing_ShouldReturnNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{getImportJobErr: ErrImportJobNotFound}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	_, err := svc.GetImportJob(context.Background(), "job_1")
	if !errors.Is(err, ErrImportJobNotFound) {
		t.Fatalf("GetImportJob() err=%v, want ErrImportJobNotFound", err)
	}
}

func TestVectorDBService_GetImportJobSummary_ShouldAggregateByStatus(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", Status: ImportJobStatusPending},
			"job_2": {ID: "job_2", Status: ImportJobStatusRunning},
			"job_3": {ID: "job_3", Status: ImportJobStatusRetrying},
			"job_4": {ID: "job_4", Status: ImportJobStatusCompleted},
			"job_5": {ID: "job_5", Status: ImportJobStatusFailed},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	summary, err := svc.GetImportJobSummary(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetImportJobSummary() error = %v", err)
	}
	if summary.Total != 5 || summary.Pending != 1 || summary.Running != 1 || summary.Retrying != 1 || summary.Completed != 1 || summary.Failed != 1 {
		t.Fatalf("GetImportJobSummary() summary=%+v", summary)
	}
}

func TestVectorDBService_RunImportJob_ShouldTransitToCompleted(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "json", `[ {"id":1}, {"id":2}, {"id":3}, {"id":4}, {"id":5}, {"id":6}, {"id":7}, {"id":8}, {"id":9}, {"id":10} ]`)

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				FilePath:       filePath,
				TotalRecords:   10,
				Status:         ImportJobStatusPending,
			},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	job, err := svc.RunImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusCompleted || job.ProcessedRecords != 10 {
		t.Fatalf("RunImportJob() job=%+v", job)
	}
}

func TestVectorDBService_RunImportJob_WhenParseFailed_ShouldTransitToFailed(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "json", `{ invalid json }`)

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				FilePath:       filePath,
				TotalRecords:   4,
				Status:         ImportJobStatusPending,
			},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	job, err := svc.RunImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusFailed || job.FailedRecords != 4 {
		t.Fatalf("RunImportJob() job=%+v", job)
	}
	if job.ErrorMessage == "" {
		t.Fatalf("RunImportJob() error message should not be empty")
	}
	if len(repo.auditLogs) == 0 {
		t.Fatalf("RunImportJob() should write audit logs on failure")
	}
}

func TestVectorDBService_RunImportJob_WithCSV_ShouldCountProcessedAndFailed(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "csv", "id,text\n1,hello\n,\n2,world\n")

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				FilePath:       filePath,
				TotalRecords:   3,
				Status:         ImportJobStatusPending,
			},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	job, err := svc.RunImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusCompleted {
		t.Fatalf("RunImportJob() status=%s, want completed", job.Status)
	}
	if job.ProcessedRecords != 2 || job.FailedRecords != 1 {
		t.Fatalf("RunImportJob() processed=%d failed=%d, want 2/1", job.ProcessedRecords, job.FailedRecords)
	}
}

func TestVectorDBService_RunImportJob_WithPDF_ShouldBeSupported(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "pdf", "first line\nsecond line\n")

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				FilePath:       filePath,
				TotalRecords:   2,
				Status:         ImportJobStatusPending,
			},
		},
	}
	backend := &mockBackend{}
	svc := NewServiceWithDeps(repo, backend)

	job, err := svc.RunImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusCompleted {
		t.Fatalf("RunImportJob() status=%s, want completed", job.Status)
	}
	if backend.upsertCalls == 0 {
		t.Fatalf("RunImportJob() should call backend upsert for pdf")
	}
}

func TestVectorDBService_RunImportJob_WithJSONVectors_ShouldUpsertBackend(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "json", `[ {"id":"a","vector":[0.1,0.2],"text":"hello"}, {"id":"b","vector":[0.2,0.3],"text":"world"} ]`)

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				FilePath:       filePath,
				TotalRecords:   2,
				Status:         ImportJobStatusPending,
			},
		},
	}
	backend := &mockBackend{}
	svc := NewServiceWithDeps(repo, backend)

	job, err := svc.RunImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusCompleted || job.ProcessedRecords != 2 || job.FailedRecords != 0 {
		t.Fatalf("RunImportJob() job=%+v", job)
	}
	if backend.upsertCalls == 0 {
		t.Fatalf("RunImportJob() should call backend upsert")
	}
}

func TestVectorDBService_RunImportJob_WhenBackendUpsertFailed_ShouldMarkFailed(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "json", `[ {"id":"a","vector":[0.1,0.2],"text":"hello"}, {"id":"b","vector":[0.2,0.3],"text":"world"} ]`)

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				FilePath:       filePath,
				TotalRecords:   2,
				Status:         ImportJobStatusPending,
			},
		},
	}
	backend := &mockBackend{upsertErr: errors.New("qdrant down")}
	svc := NewServiceWithDeps(repo, backend)

	job, err := svc.RunImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RunImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusFailed || job.FailedRecords != 2 {
		t.Fatalf("RunImportJob() job=%+v", job)
	}
	if job.ErrorMessage == "" {
		t.Fatalf("RunImportJob() error message should not be empty")
	}
}

func TestVectorDBService_RetryImportJob_ShouldResetAndRun(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "json", `[ {"id":"1","vector":[0.1,0.2]}, {"id":"2","vector":[0.1,0.2]}, {"id":"3","vector":[0.1,0.2]}, {"id":"4","vector":[0.1,0.2]}, {"id":"5","vector":[0.1,0.2]}, {"id":"6","vector":[0.1,0.2]}, {"id":"7","vector":[0.1,0.2]}, {"id":"8","vector":[0.1,0.2]}, {"id":"9","vector":[0.1,0.2]}, {"id":"10","vector":[0.1,0.2]}, {"id":"11","vector":[0.1,0.2]}, {"id":"12","vector":[0.1,0.2]} ]`)

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:               "job_1",
				CollectionID:     "col_1",
				CollectionName:   "docs",
				FilePath:         filePath,
				TotalRecords:     12,
				ProcessedRecords: 2,
				FailedRecords:    10,
				RetryCount:       0,
				MaxRetries:       2,
				Status:           ImportJobStatusFailed,
			},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	job, err := svc.RetryImportJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("RetryImportJob() error = %v", err)
	}
	if job.Status != ImportJobStatusCompleted || job.FailedRecords != 0 || job.ProcessedRecords != 12 {
		t.Fatalf("RetryImportJob() job=%+v", job)
	}
	if job.RetryCount != 1 {
		t.Fatalf("RetryImportJob() retry_count = %d, want 1", job.RetryCount)
	}
}

func TestVectorDBService_RetryImportJob_WhenExceeded_ShouldFail(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:           "job_1",
				CollectionID: "col_1",
				TotalRecords: 12,
				RetryCount:   2,
				MaxRetries:   2,
				Status:       ImportJobStatusFailed,
			},
		},
	}
	svc := NewServiceWithDeps(repo, &mockBackend{})

	_, err := svc.RetryImportJob(context.Background(), "job_1")
	if !errors.Is(err, ErrImportJobRetryExceeded) {
		t.Fatalf("RetryImportJob() err=%v, want ErrImportJobRetryExceeded", err)
	}
}

func TestCollectionHandler_ImportJobsRoutes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{getResp: &Collection{ID: "col_1", Name: "docs"}}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterImportJobRoutes(group, h)

	createBody := map[string]any{
		"collection_name": "docs",
		"file_name":       "docs.json",
		"file_path":       "/tmp/docs.json",
		"file_size":       1024,
		"total_records":   100,
		"created_by":      "tester",
	}
	payload, err := json.Marshal(createBody)
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", w.Code, http.StatusCreated)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["status"] != string(ImportJobStatusPending) {
		t.Fatalf("POST data.status = %v, want %s", data["status"], ImportJobStatusPending)
	}

	jobID, _ := data["id"].(string)
	if jobID == "" {
		t.Fatalf("POST data.id should not be empty")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs", http.NoBody)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, want %d", listW.Code, http.StatusOK)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/"+jobID, http.NoBody)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET detail status = %d, want %d", getW.Code, http.StatusOK)
	}

	updateBody := map[string]any{
		"status":            string(ImportJobStatusCompleted),
		"processed_records": 100,
		"failed_records":    0,
		"completed_at":      time.Now().UTC().Format(time.RFC3339),
	}
	updatePayload, err := json.Marshal(updateBody)
	if err != nil {
		t.Fatalf("json.Marshal(updateBody) error = %v", err)
	}
	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/import-jobs/"+jobID+"/status", bytes.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d", updateW.Code, http.StatusOK)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/"+jobID+"/run", http.NoBody)
	runW := httptest.NewRecorder()
	r.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("POST run status = %d, want %d", runW.Code, http.StatusOK)
	}

	failBody := map[string]any{
		"status":            string(ImportJobStatusFailed),
		"processed_records": 20,
		"failed_records":    80,
		"error_message":     "temporary error",
	}
	failPayload, err := json.Marshal(failBody)
	if err != nil {
		t.Fatalf("json.Marshal(failBody) error = %v", err)
	}
	failReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/import-jobs/"+jobID+"/status", bytes.NewReader(failPayload))
	failReq.Header.Set("Content-Type", "application/json")
	failW := httptest.NewRecorder()
	r.ServeHTTP(failW, failReq)
	if failW.Code != http.StatusOK {
		t.Fatalf("PUT failed status = %d, want %d", failW.Code, http.StatusOK)
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/"+jobID+"/retry", http.NoBody)
	retryW := httptest.NewRecorder()
	r.ServeHTTP(retryW, retryReq)
	if retryW.Code != http.StatusOK {
		t.Fatalf("POST retry status = %d, want %d", retryW.Code, http.StatusOK)
	}
}

func TestCollectionHandler_RetryFailedImportJobsRoute(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {
				ID:             "job_1",
				CollectionID:   "col_1",
				CollectionName: "docs",
				TotalRecords:   10,
				RetryCount:     0,
				MaxRetries:     2,
				Status:         ImportJobStatusFailed,
			},
			"job_2": {
				ID:             "job_2",
				CollectionID:   "col_1",
				CollectionName: "docs",
				TotalRecords:   20,
				RetryCount:     2,
				MaxRetries:     2,
				Status:         ImportJobStatusFailed,
			},
		},
	}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterImportJobRoutes(group, h)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/import-jobs/retry-failed?limit=10", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST retry-failed status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	jobs, _ := data["jobs"].([]any)
	if len(jobs) != 1 {
		t.Fatalf("retry-failed jobs len = %d, want 1", len(jobs))
	}
}

func TestCollectionHandler_GetImportJobErrorsRoute(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", CollectionID: "col_1", CollectionName: "docs", Status: ImportJobStatusFailed},
		},
		auditLogs: []AuditLog{
			{ID: 1, UserID: "tester", Action: "import_run_failed", ResourceType: "import_job", ResourceID: "job_1", Details: "parse failed"},
			{ID: 2, UserID: "tester", Action: "import_retry_exceeded", ResourceType: "import_job", ResourceID: "job_1", Details: "retry exceeded"},
		},
	}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterImportJobRoutes(group, h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/job_1/errors?limit=10", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET errors status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	logs, _ := data["logs"].([]any)
	if len(logs) != 2 {
		t.Fatalf("logs len = %d, want 2", len(logs))
	}
}

func TestCollectionHandler_GetImportJobErrorsRoute_WithActionFilter(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", CollectionID: "col_1", CollectionName: "docs", Status: ImportJobStatusFailed},
		},
		auditLogs: []AuditLog{
			{ID: 1, UserID: "tester", Action: "import_run_failed", ResourceType: "import_job", ResourceID: "job_1", Details: "parse failed"},
			{ID: 2, UserID: "tester", Action: "import_retry_exceeded", ResourceType: "import_job", ResourceID: "job_1", Details: "retry exceeded"},
		},
	}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterImportJobRoutes(group, h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/job_1/errors?action=import_run_failed&limit=10", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET errors status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	logs, _ := data["logs"].([]any)
	if len(logs) != 1 {
		t.Fatalf("logs len = %d, want 1", len(logs))
	}
}

func TestCollectionHandler_GetImportJobErrorsRoute_WithOffset(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", CollectionID: "col_1", CollectionName: "docs", Status: ImportJobStatusFailed},
		},
		auditLogs: []AuditLog{
			{ID: 1, UserID: "tester", Action: "import_run_failed", ResourceType: "import_job", ResourceID: "job_1", Details: "error-1"},
			{ID: 2, UserID: "tester", Action: "import_run_failed", ResourceType: "import_job", ResourceID: "job_1", Details: "error-2"},
		},
	}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterImportJobRoutes(group, h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/job_1/errors?limit=1&offset=1", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET errors status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	logs, _ := data["logs"].([]any)
	if len(logs) != 1 {
		t.Fatalf("logs len = %d, want 1", len(logs))
	}
}

func TestCollectionHandler_GetImportJobSummaryRoute(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", Status: ImportJobStatusPending},
			"job_2": {ID: "job_2", Status: ImportJobStatusFailed},
			"job_3": {ID: "job_3", Status: ImportJobStatusCompleted},
		},
	}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterImportJobRoutes(group, h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/import-jobs/summary", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET summary status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["total"] != float64(3) {
		t.Fatalf("summary total = %v, want 3", data["total"])
	}
	if data["failed"] != float64(1) {
		t.Fatalf("summary failed = %v, want 1", data["failed"])
	}
}

func writeTempFile(t *testing.T, ext, content string) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "import-*."+ext)
	if err != nil {
		file, err = os.CreateTemp(t.TempDir(), "import-*")
	}
	if err != nil {
		t.Fatalf("os.CreateTemp() error = %v", err)
	}
	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file.Close() error = %v", err)
	}
	return file.Name()
}
