package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"ai-gateway/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageHandler_GetUsageLogs_FilterByExperimentAndDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "usage-handler-test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now().UnixMilli()
	err = store.LogUsage(map[string]interface{}{
		"request_id":     "req-admin-1",
		"timestamp":      now,
		"model":          "gpt-4o-mini",
		"provider":       "openai",
		"tokens":         int64(100),
		"input_tokens":   int64(60),
		"output_tokens":  int64(40),
		"latency_ms":     int64(320),
		"ttft_ms":        int64(120),
		"cache_hit":      true,
		"success":        true,
		"experiment_tag": "exp-a",
		"domain_tag":     "finance",
	})
	require.NoError(t, err)

	err = store.LogUsage(map[string]interface{}{
		"request_id":     "req-admin-2",
		"timestamp":      now - 1000,
		"model":          "gpt-4o-mini",
		"provider":       "openai",
		"tokens":         int64(80),
		"input_tokens":   int64(50),
		"output_tokens":  int64(30),
		"latency_ms":     int64(410),
		"ttft_ms":        int64(150),
		"cache_hit":      false,
		"success":        true,
		"experiment_tag": "exp-b",
		"domain_tag":     "general",
	})
	require.NoError(t, err)

	handler := NewUsageHandler(store)
	r := gin.New()
	r.GET("/admin/usage/logs", handler.GetUsageLogs)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/logs?experiment_tag=exp-a&domain_tag=finance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool               `json:"success"`
		Data    []UsageLogResponse `json:"data"`
		Total   int                `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "exp-a", resp.Data[0].ExperimentTag)
	assert.Equal(t, "finance", resp.Data[0].DomainTag)
}
