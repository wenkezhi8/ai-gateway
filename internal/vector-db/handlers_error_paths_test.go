package vectordb

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCollectionAndImportHandlers_ServiceErrorPaths_ShouldReturnExpectedCodes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(nil, nil))

	api := r.Group("/api/admin")
	RegisterCollectionRoutes(api, h)
	RegisterImportJobRoutes(api, h)

	tests := []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{method: http.MethodPost, path: "/api/admin/vector-db/collections", body: `{"name":"docs","dimension":3}`, code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/collections?is_public=bad", code: http.StatusBadRequest},
		{method: http.MethodGet, path: "/api/admin/vector-db/collections", code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/collections/docs", code: http.StatusInternalServerError},
		{method: http.MethodPut, path: "/api/admin/vector-db/collections/docs", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPut, path: "/api/admin/vector-db/collections/docs", body: `{}`, code: http.StatusInternalServerError},
		{method: http.MethodDelete, path: "/api/admin/vector-db/collections/docs", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/admin/vector-db/collections/docs", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/collections/docs/empty", code: http.StatusInternalServerError},

		{method: http.MethodGet, path: "/api/admin/vector-db/import-jobs/summary", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/import-jobs", body: `{"collection_name":"docs","file_name":"f.json","file_path":"/tmp/f.json","file_size":1,"total_records":1}`, code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/import-jobs", code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/import-jobs/job1", code: http.StatusInternalServerError},
		{method: http.MethodPut, path: "/api/admin/vector-db/import-jobs/job1/status", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPut, path: "/api/admin/vector-db/import-jobs/job1/status", body: `{"status":"running"}`, code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/import-jobs/job1/run", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/import-jobs/job1/retry", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/import-jobs/job1/cancel", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/import-jobs/retry-failed", code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/import-jobs/job1/errors", code: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body == "" {
				body = bytes.NewBuffer(nil)
			} else {
				body = bytes.NewBufferString(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("status=%d, want %d", w.Code, tc.code)
			}
		})
	}
}

func TestBackupAndMonitoringAndRBACHandlers_ErrorPaths_ShouldReturnExpectedCodes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(nil, nil))
	api := r.Group("/api/admin")
	RegisterBackupRoutes(api, h)
	RegisterMonitoringRoutes(api, h)
	RegisterRBACRoutes(api, NewRBACService(nil))

	tests := []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{method: http.MethodPost, path: "/api/admin/vector-db/backups", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups", body: `{"collection_name":"docs"}`, code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/backups", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups/policy/run", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups/policy/run", body: `{"collection_name":"docs"}`, code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups/a/restore", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups/1/restore", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups/a/retry", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/backups/1/retry", code: http.StatusInternalServerError},

		{method: http.MethodGet, path: "/api/admin/vector-db/alerts/rules", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/alerts/rules", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/alerts/rules", body: `{"name":"r","metric":"m","operator":"gt","threshold":1,"duration":"1m"}`, code: http.StatusInternalServerError},
		{method: http.MethodPut, path: "/api/admin/vector-db/alerts/rules/a", body: `{}`, code: http.StatusBadRequest},
		{method: http.MethodPut, path: "/api/admin/vector-db/alerts/rules/1", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPut, path: "/api/admin/vector-db/alerts/rules/1", body: `{"enabled":true}`, code: http.StatusInternalServerError},
		{method: http.MethodDelete, path: "/api/admin/vector-db/alerts/rules/a", code: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/admin/vector-db/alerts/rules/1", code: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/api/admin/vector-db/metrics/summary", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/alerts/rules/notify-test", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/alerts/rules/notify-test", body: `{"rule_name":"r","message":"m","channels":["console"]}`, code: http.StatusInternalServerError},

		{method: http.MethodGet, path: "/api/admin/vector-db/permissions", code: http.StatusInternalServerError},
		{method: http.MethodPost, path: "/api/admin/vector-db/permissions", body: "{bad", code: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/admin/vector-db/permissions", body: `{"api_key":"k","role":"reader"}`, code: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/admin/vector-db/permissions/a", code: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/admin/vector-db/permissions/1", code: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body == "" {
				body = bytes.NewBuffer(nil)
			} else {
				body = bytes.NewBufferString(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("status=%d, want %d", w.Code, tc.code)
			}
		})
	}
}
