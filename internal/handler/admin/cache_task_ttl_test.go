package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/cache"
	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes_ShouldExposeCacheTaskTTLEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	cacheHandler := NewCacheHandler(manager)

	engine := gin.New()
	adminGroup := engine.Group("/api/admin")

	RegisterRoutes(adminGroup, &Handlers{
		Account:     &AccountHandler{},
		Provider:    &ProviderHandler{},
		Routing:     &RoutingHandler{},
		Cache:       cacheHandler,
		Knowledge:   &KnowledgeHandler{},
		Dashboard:   &DashboardHandler{},
		SmartRouter: &RouterHandler{},
		APIKey:      &APIKeyHandler{},
		Upload:      &UploadHandler{},
		Alert:       &AlertHandler{},
		Feedback:    &FeedbackHandler{},
		Ops:         &OpsHandler{},
		Usage:       &UsageHandler{},
		Settings:    &SettingsHandler{},
		Trace:       &TraceHandler{},
		Edition:     &EditionHandler{},
		VectorDB:    &vectordb.CollectionHandler{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/cache/task-ttl", http.NoBody)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("expected /api/admin/cache/task-ttl to be registered, got 404")
	}
}

func TestCacheHandler_GetCacheTaskTTL_ShouldReturnTaskTypesContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := NewCacheHandler(manager)

	router := gin.New()
	router.GET("/api/admin/cache/task-ttl", handler.GetCacheTaskTTL)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/cache/task-ttl", http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TaskTypes []map[string]interface{} `json:"task_types"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if len(resp.Data.TaskTypes) == 0 {
		t.Fatalf("expected non-empty task_types")
	}

	required := map[string]struct{}{
		"key":         {},
		"label":       {},
		"description": {},
		"default_ttl": {},
		"ttl_unit":    {},
	}
	for _, item := range resp.Data.TaskTypes {
		for field := range required {
			if _, ok := item[field]; !ok {
				t.Fatalf("expected task_type field %s", field)
			}
		}
		if item["ttl_unit"] != "hours" {
			t.Fatalf("expected ttl_unit=hours, got %#v", item["ttl_unit"])
		}
	}
}

func TestCacheHandler_GetCacheTaskTTL_ShouldReturnModelOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := NewCacheHandler(manager)

	router := gin.New()
	router.GET("/api/admin/cache/task-ttl", handler.GetCacheTaskTTL)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/cache/task-ttl", http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ModelOptions []struct {
				ProviderID    string   `json:"provider_id"`
				ProviderLabel string   `json:"provider_label"`
				Models        []string `json:"models"`
			} `json:"model_options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if len(resp.Data.ModelOptions) == 0 {
		t.Fatalf("expected non-empty model_options")
	}

	for _, group := range resp.Data.ModelOptions {
		if group.ProviderID == "" {
			t.Fatalf("expected provider_id")
		}
		if group.ProviderLabel == "" {
			t.Fatalf("expected provider_label")
		}
		if len(group.Models) == 0 {
			t.Fatalf("expected non-empty models for provider %s", group.ProviderID)
		}
	}
}
