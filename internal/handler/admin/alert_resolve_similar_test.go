package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAlertHandler_ResolveSimilarAlerts_ShouldResolvePendingGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)
	handler.alerts = []AlertRecord{
		{ID: "a1", Time: time.Now().Add(-2 * time.Hour).Format(time.RFC3339), Level: "warning", Source: "system", Message: "cpu high", Status: "pending"},
		{ID: "a2", Time: time.Now().Add(-1 * time.Hour).Format(time.RFC3339), Level: "warning", Source: "system", Message: "cpu high", Status: "pending"},
		{ID: "a3", Time: time.Now().Add(-30 * time.Minute).Format(time.RFC3339), Level: "warning", Source: "system", Message: "cpu high", Status: "resolved", ResolvedAt: time.Now().Add(-20 * time.Minute).Format(time.RFC3339)},
		{ID: "a4", Time: time.Now().Add(-30 * time.Minute).Format(time.RFC3339), Level: "critical", Source: "system", Message: "cpu high", Status: "pending"},
	}

	router := gin.New()
	router.POST("/api/admin/alerts/resolve-similar", handler.ResolveSimilarAlerts)

	body := map[string]string{"level": "warning", "source": "system", "message": "cpu high"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/alerts/resolve-similar", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Affected int `json:"affected"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if resp.Data.Affected != 2 {
		t.Fatalf("expected affected=2, got %d", resp.Data.Affected)
	}

	if handler.alerts[0].Status != "resolved" || handler.alerts[0].ResolvedAt == "" {
		t.Fatalf("expected first alert resolved with resolvedAt")
	}
	if handler.alerts[1].Status != "resolved" || handler.alerts[1].ResolvedAt == "" {
		t.Fatalf("expected second alert resolved with resolvedAt")
	}
	if handler.alerts[2].Status != "resolved" {
		t.Fatalf("expected third alert remain resolved")
	}
	if handler.alerts[3].Status != "pending" {
		t.Fatalf("expected different-level alert untouched")
	}
}

func TestAlertHandler_ResolveSimilarAlerts_ShouldValidateRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)
	router := gin.New()
	router.POST("/api/admin/alerts/resolve-similar", handler.ResolveSimilarAlerts)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/alerts/resolve-similar", bytes.NewBufferString(`{"level":"warning","source":"system"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAlertHandler_ResolveSimilarAlerts_ShouldResolveByDedupKeyAcrossDifferentMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTestAlertHandler(t)
	handler.alerts = []AlertRecord{
		{ID: "a1", Time: time.Now().Add(-4 * time.Hour).Format(time.RFC3339), Level: "warning", Source: "system", Message: "内存使用率偏高: 93.2%", Status: "pending", DedupKey: "memory_warning"},
		{ID: "a2", Time: time.Now().Add(-3 * time.Hour).Format(time.RFC3339), Level: "warning", Source: "system", Message: "内存使用率偏高: 93.1%", Status: "pending", DedupKey: "memory_warning"},
		{ID: "a3", Time: time.Now().Add(-2 * time.Hour).Format(time.RFC3339), Level: "warning", Source: "system", Message: "内存使用率偏高: 92.9%", Status: "pending", DedupKey: "memory_warning"},
		{ID: "b1", Time: time.Now().Add(-2 * time.Hour).Format(time.RFC3339), Level: "warning", Source: "system", Message: "Goroutine 数偏高: 9000", Status: "pending", DedupKey: "goroutine_warning"},
	}

	router := gin.New()
	router.POST("/api/admin/alerts/resolve-similar", handler.ResolveSimilarAlerts)

	body := map[string]string{"level": "warning", "source": "system", "dedup_key": "memory_warning"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/alerts/resolve-similar", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Affected int    `json:"affected"`
			Key      string `json:"key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if resp.Data.Affected != 3 {
		t.Fatalf("expected affected=3, got %d", resp.Data.Affected)
	}
	if resp.Data.Key != "warning|system|memory_warning" {
		t.Fatalf("expected dedup key in response, got %s", resp.Data.Key)
	}

	if handler.alerts[0].Status != "resolved" || handler.alerts[1].Status != "resolved" || handler.alerts[2].Status != "resolved" {
		t.Fatalf("expected memory_warning group resolved")
	}
	if handler.alerts[3].Status != "pending" {
		t.Fatalf("expected goroutine warning untouched")
	}
}
