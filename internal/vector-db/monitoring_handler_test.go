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

func TestMonitoringHandler_NotifyAlertChannels_ShouldReturn200(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterMonitoringRoutes(group, h)

	body := map[string]any{
		"rule_name": "high-latency",
		"message":   "search latency exceeded",
		"channels":  []string{"webhook", "email", "console"},
		"operator":  "tester",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/alerts/rules/notify-test", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST notify-test status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeJSONBody(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["sent"] != float64(3) {
		t.Fatalf("notify sent = %v, want 3", data["sent"])
	}
}

func TestMonitoringHandler_InvalidRequests_ShouldReturn400Or404(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCollectionHandler(NewServiceWithDeps(&mockRepo{}, &mockBackend{}))

	group := r.Group("/api/admin")
	RegisterMonitoringRoutes(group, h)

	invalidCreateReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/alerts/rules", bytes.NewBufferString("{bad"))
	invalidCreateReq.Header.Set("Content-Type", "application/json")
	invalidCreateResp := httptest.NewRecorder()
	r.ServeHTTP(invalidCreateResp, invalidCreateReq)
	if invalidCreateResp.Code != http.StatusBadRequest {
		t.Fatalf("POST invalid create status = %d, want %d", invalidCreateResp.Code, http.StatusBadRequest)
	}

	invalidUpdateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/alerts/rules/abc", bytes.NewBufferString("{}"))
	invalidUpdateReq.Header.Set("Content-Type", "application/json")
	invalidUpdateResp := httptest.NewRecorder()
	r.ServeHTTP(invalidUpdateResp, invalidUpdateReq)
	if invalidUpdateResp.Code != http.StatusBadRequest {
		t.Fatalf("PUT invalid id status = %d, want %d", invalidUpdateResp.Code, http.StatusBadRequest)
	}

	missingUpdatePayload, err := json.Marshal(map[string]any{"enabled": false})
	if err != nil {
		t.Fatalf("json.Marshal(missingUpdatePayload) error = %v", err)
	}
	missingUpdateReq := httptest.NewRequest(http.MethodPut, "/api/admin/vector-db/alerts/rules/99", bytes.NewReader(missingUpdatePayload))
	missingUpdateReq.Header.Set("Content-Type", "application/json")
	missingUpdateResp := httptest.NewRecorder()
	r.ServeHTTP(missingUpdateResp, missingUpdateReq)
	if missingUpdateResp.Code != http.StatusNotFound {
		t.Fatalf("PUT missing id status = %d, want %d", missingUpdateResp.Code, http.StatusNotFound)
	}

	invalidDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/alerts/rules/abc", http.NoBody)
	invalidDeleteResp := httptest.NewRecorder()
	r.ServeHTTP(invalidDeleteResp, invalidDeleteReq)
	if invalidDeleteResp.Code != http.StatusBadRequest {
		t.Fatalf("DELETE invalid id status = %d, want %d", invalidDeleteResp.Code, http.StatusBadRequest)
	}

	missingDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/vector-db/alerts/rules/99", http.NoBody)
	missingDeleteResp := httptest.NewRecorder()
	r.ServeHTTP(missingDeleteResp, missingDeleteReq)
	if missingDeleteResp.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing id status = %d, want %d", missingDeleteResp.Code, http.StatusNotFound)
	}

	invalidNotifyReq := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/alerts/rules/notify-test", bytes.NewBufferString("{bad"))
	invalidNotifyReq.Header.Set("Content-Type", "application/json")
	invalidNotifyResp := httptest.NewRecorder()
	r.ServeHTTP(invalidNotifyResp, invalidNotifyReq)
	if invalidNotifyResp.Code != http.StatusBadRequest {
		t.Fatalf("POST notify invalid body status = %d, want %d", invalidNotifyResp.Code, http.StatusBadRequest)
	}
}
