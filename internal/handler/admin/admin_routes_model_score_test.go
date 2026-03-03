package admin

import (
	"testing"

	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes_ModelScoreManagementEndpoints_ShouldBeRegistered(t *testing.T) {
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

	routes := engine.Routes()
	type routeKey struct {
		method string
		path   string
	}
	registered := make(map[routeKey]struct{}, len(routes))
	for _, route := range routes {
		registered[routeKey{method: route.Method, path: route.Path}] = struct{}{}
	}

	required := []routeKey{
		{method: "GET", path: "/api/admin/router/models"},
		{method: "PUT", path: "/api/admin/router/models/:model"},
		{method: "DELETE", path: "/api/admin/router/models/:model"},
	}
	for _, item := range required {
		if _, ok := registered[item]; !ok {
			t.Fatalf("expected route registered: %s %s", item.method, item.path)
		}
	}
}
