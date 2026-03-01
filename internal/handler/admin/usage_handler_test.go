package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ai-gateway/internal/storage"
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
		"usage_source":   "actual",
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
		"usage_source":   "estimated",
	})
	require.NoError(t, err)

	handler := NewUsageHandler(store)
	r := gin.New()
	r.GET("/admin/usage/logs", handler.GetUsageLogs)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/logs?experiment_tag=exp-a&domain_tag=finance", http.NoBody)
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
	assert.Equal(t, "actual", resp.Data[0].UsageSource)
}

func TestUsageHandler_GetUsageStats_FilterByRangeModelAndTaskType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "usage-handler-stats-test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now().UnixMilli()
	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-stats-1",
		"timestamp":  now - 15*60*1000,
		"model":      "qwen2.5:3b",
		"provider":   "ollama",
		"tokens":     int64(180),
		"latency_ms": int64(18),
		"cache_hit":  true,
		"success":    true,
		"task_type":  "chat",
	}))
	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-stats-2",
		"timestamp":  now - 30*60*1000,
		"model":      "qwen2.5:3b",
		"provider":   "ollama",
		"tokens":     int64(100),
		"latency_ms": int64(20),
		"cache_hit":  true,
		"success":    false,
		"task_type":  "chat",
	}))
	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-stats-3",
		"timestamp":  now - 12*24*60*60*1000,
		"model":      "qwen2.5:3b",
		"provider":   "ollama",
		"tokens":     int64(220),
		"latency_ms": int64(25),
		"cache_hit":  true,
		"success":    true,
		"task_type":  "qa",
	}))

	handler := NewUsageHandler(store)
	r := gin.New()
	r.GET("/admin/usage/stats", handler.GetUsageStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?range=7d&model=qwen2.5:3b&task_type=chat", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool               `json:"success"`
		Data    UsageStatsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)

	assert.Equal(t, int64(2), resp.Data.TotalRequests)
	assert.Equal(t, int64(280), resp.Data.TotalTokens)
	assert.Equal(t, int64(180), resp.Data.SavedTokens)
	assert.Equal(t, int64(1), resp.Data.SavedRequests)
}

func TestUsageHandler_ClearUsageLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "usage-handler-clear-test.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	now := time.Now().UnixMilli()
	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-clear-admin-1",
		"timestamp":  now,
		"model":      "qwen2.5:3b",
		"provider":   "ollama",
		"tokens":     int64(100),
		"cache_hit":  true,
		"success":    true,
	}))
	require.NoError(t, store.LogUsage(map[string]interface{}{
		"request_id": "req-clear-admin-2",
		"timestamp":  now - 1000,
		"model":      "qwen2.5:3b",
		"provider":   "ollama",
		"tokens":     int64(70),
		"cache_hit":  false,
		"success":    true,
	}))

	handler := NewUsageHandler(store)
	r := gin.New()
	r.DELETE("/admin/usage/logs", handler.ClearUsageLogs)

	req := httptest.NewRequest(http.MethodDelete, "/admin/usage/logs", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Deleted int64 `json:"deleted"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	assert.Equal(t, int64(2), resp.Data.Deleted)

	stats := store.GetUsageStats()
	assert.Equal(t, int64(0), stats["total_requests"])
	assert.Equal(t, int64(0), stats["total_tokens"])
}
