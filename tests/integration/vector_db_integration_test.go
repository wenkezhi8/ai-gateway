package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

func TestIntegration_VectorDBAlertNotifyRoute_ShouldWork(t *testing.T) {
	setGinTestMode()

	r := gin.New()
	service := vectordb.NewService()
	handler := vectordb.NewCollectionHandler(service)
	admin := r.Group("/api/admin")
	vectordb.RegisterMonitoringRoutes(admin, handler)

	body := map[string]any{
		"rule_name": "high-latency",
		"message":   "search latency high",
		"channels":  []string{"webhook", "email"},
		"operator":  "integration",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/vector-db/alerts/rules/notify-test", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", w.Code, http.StatusOK)
	}
}
