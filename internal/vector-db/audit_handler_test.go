package vectordb

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuditHandler_ListAuditLogs_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockRepo{auditLogs: []AuditLog{{
		ID:           1,
		UserID:       "system",
		Action:       "import_run_failed",
		ResourceType: "import_job",
		ResourceID:   "job_1",
		Details:      "failed",
		CreatedAt:    time.Now().UTC(),
	}}}
	h := NewCollectionHandler(NewServiceWithDeps(repo, &mockBackend{}))

	api := r.Group("/api/admin")
	RegisterAuditRoutes(api, h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/audit/logs?resource_type=import_job", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", w.Code, http.StatusOK)
	}
}
