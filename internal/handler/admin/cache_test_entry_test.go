package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/cache"

	"github.com/gin-gonic/gin"
)

func TestCacheHandler_AddTestCacheEntry_ShouldRejectInReleaseMode(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	t.Cleanup(func() {
		gin.SetMode(previousMode)
	})

	handler := NewCacheHandler(cache.NewManagerWithCache(cache.NewMemoryCache()))
	router := gin.New()
	router.POST("/api/admin/cache/test-entry", handler.AddTestCacheEntry)

	body := []byte(`{"task_type":"chat","user_message":"hello","ai_response":"world"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/cache/test-entry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCacheHandler_AddTestCacheEntry_ShouldPersistReadableMetadata(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(previousMode)
	})

	handler := NewCacheHandler(cache.NewManagerWithCache(cache.NewMemoryCache()))
	router := gin.New()
	router.POST("/api/admin/cache/test-entry", handler.AddTestCacheEntry)
	router.GET("/api/admin/cache/entries", handler.GetCacheEntries)

	body := []byte(`{"task_type":"chat","user_message":"缓存测试问题","ai_response":"缓存测试回答","model":"debug-model","provider":"debug-provider","ttl":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/cache/test-entry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("add test entry status=%d body=%s", rec.Code, rec.Body.String())
	}

	entriesReq := httptest.NewRequest(http.MethodGet, "/api/admin/cache/entries?type=response&aggregate=1&readable_only=1&page=1&page_size=20", http.NoBody)
	entriesRec := httptest.NewRecorder()
	router.ServeHTTP(entriesRec, entriesReq)

	if entriesRec.Code != http.StatusOK {
		t.Fatalf("entries status=%d body=%s", entriesRec.Code, entriesRec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total   int `json:"total"`
			Entries []struct {
				Key            string `json:"key"`
				TaskType       string `json:"task_type"`
				TaskTypeSource string `json:"task_type_source"`
				UserMessage    string `json:"user_message"`
				AIResponse     string `json:"ai_response"`
				Model          string `json:"model"`
				Provider       string `json:"provider"`
			} `json:"entries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(entriesRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode entries response failed: %v", err)
	}

	if !resp.Success {
		t.Fatalf("success=false body=%s", entriesRec.Body.String())
	}
	if resp.Data.Total != 1 || len(resp.Data.Entries) != 1 {
		t.Fatalf("entries total=%d len=%d want=1 body=%s", resp.Data.Total, len(resp.Data.Entries), entriesRec.Body.String())
	}

	entry := resp.Data.Entries[0]
	if entry.UserMessage == "" {
		t.Fatalf("user_message should not be empty: %+v", entry)
	}
	if entry.AIResponse == "" {
		t.Fatalf("ai_response should not be empty: %+v", entry)
	}
	if entry.TaskType != "chat" {
		t.Fatalf("task_type=%q want=chat", entry.TaskType)
	}
	if entry.TaskTypeSource != "manual" {
		t.Fatalf("task_type_source=%q want=manual", entry.TaskTypeSource)
	}
}
