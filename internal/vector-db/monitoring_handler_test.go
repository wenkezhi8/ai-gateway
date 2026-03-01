//nolint:errcheck // Type assertions in tests intentionally validate response shape.
package vectordb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMonitoringHandler_GetMetrics_ShouldReturnSummary(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{
		importJobs: map[string]*ImportJob{
			"job_1": {ID: "job_1", Status: ImportJobStatusPending},
			"job_2": {ID: "job_2", Status: ImportJobStatusFailed},
		},
		alertRules: map[int64]*AlertRule{
			1: {ID: 1, Name: "r1", Enabled: true},
		},
	}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterMonitoringRoutes(group, h)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/metrics/summary", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET summary status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["alert_rules_total"] != float64(1) {
		t.Fatalf("summary alert_rules_total = %v, want 1", data["alert_rules_total"])
	}
}

func TestMonitoringHandler_AlertRulesCRUD(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterMonitoringRoutes(group, h)

	createBody := map[string]any{
		"name":      "high-latency",
		"metric":    "search_p95_ms",
		"operator":  "gt",
		"threshold": 500,
		"duration":  "5m",
		"channels":  []string{"webhook"},
	}
	payload, err := json.Marshal(createBody)
	if err != nil {
		t.Fatalf("json.Marshal(createBody) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/alerts/rules", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST rules status = %d, want %d", w.Code, http.StatusCreated)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	id, _ := data["id"].(float64)
	if id <= 0 {
		t.Fatalf("POST rules id = %v, want >0", data["id"])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/vector-db/alerts/rules", http.NoBody)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("GET rules status = %d, want %d", listW.Code, http.StatusOK)
	}

	updateBody := map[string]any{"enabled": false}
	updatePayload, err := json.Marshal(updateBody)
	if err != nil {
		t.Fatalf("json.Marshal(updateBody) error = %v", err)
	}
	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/alerts/rules/1", bytes.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("PUT rules status = %d, want %d", updateW.Code, http.StatusOK)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/alerts/rules/1", http.NoBody)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("DELETE rules status = %d, want %d", deleteW.Code, http.StatusOK)
	}
}
