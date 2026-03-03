package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/routing"
	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes_ModelScoreManagementEndpoints_ShouldReturn404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	adminGroup := engine.Group("/api/admin")

	RegisterRoutes(adminGroup, &Handlers{
		Account:     &AccountHandler{},
		Provider:    &ProviderHandler{},
		Routing:     &RoutingHandler{},
		Cache:       &CacheHandler{},
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
		VectorDB:    &vectordb.CollectionHandler{},
	})

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "get model scores",
			method: http.MethodGet,
			path:   "/api/admin/router/models",
		},
		{
			name:   "update model score",
			method: http.MethodPut,
			path:   "/api/admin/router/models/gpt-4o",
		},
		{
			name:   "delete model score",
			method: http.MethodDelete,
			path:   "/api/admin/router/models/gpt-4o",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRegisterRoutes_ModelRegistryEndpoints_ShouldBeRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	adminGroup := engine.Group("/api/admin")

	smartRouter := routing.NewSmartRouter()

	RegisterRoutes(adminGroup, &Handlers{
		Account:     &AccountHandler{},
		Provider:    &ProviderHandler{},
		Routing:     &RoutingHandler{},
		Cache:       &CacheHandler{},
		Knowledge:   &KnowledgeHandler{},
		Dashboard:   &DashboardHandler{},
		SmartRouter: NewRouterHandler(smartRouter, nil),
		APIKey:      &APIKeyHandler{},
		Upload:      &UploadHandler{},
		Alert:       &AlertHandler{},
		Feedback:    &FeedbackHandler{},
		Ops:         &OpsHandler{},
		Usage:       &UsageHandler{},
		Settings:    &SettingsHandler{},
		Trace:       &TraceHandler{},
		VectorDB:    &vectordb.CollectionHandler{},
	})

	t.Run("get model registry", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/router/model-registry", http.NoBody)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("expected non-404 for model-registry endpoint")
		}
	})

	t.Run("upsert model registry", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/admin/router/model-registry/gpt-4o", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("expected non-404 for model-registry endpoint")
		}
	})

	t.Run("delete model registry", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/admin/router/model-registry/gpt-4o", http.NoBody)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("expected non-404 for model-registry endpoint")
		}
	})
}
