package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/limiter"
	"ai-gateway/internal/provider"
	vectordb "ai-gateway/internal/vector-db"

	"github.com/gin-gonic/gin"
)

func TestAdminRoutes_ProviderTypesRoute_ShouldNotBeCapturedByIdRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := provider.NewRegistry()
	registry.Register("openai", &providerDeleteTestProvider{name: "openai", models: []string{"gpt-4o"}, enabled: true})
	manager := limiter.NewAccountManager(nil, nil)
	handler := NewProviderHandler(registry, manager, nil, "")

	engine := newProviderTypesTestRouter(handler, &AccountHandler{manager: manager})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/providers/types", http.NoBody)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	providerReq := httptest.NewRequest(http.MethodGet, "/api/admin/providers/openai", http.NoBody)
	providerRec := httptest.NewRecorder()
	engine.ServeHTTP(providerRec, providerReq)
	if providerRec.Code != http.StatusOK {
		t.Fatalf("expected /providers/:id still available, got %d body=%s", providerRec.Code, providerRec.Body.String())
	}
}

func TestProviderHandler_GetProviderTypes_ShouldIncludeBuiltInAndDynamicProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := provider.NewRegistry()
	registry.Register("acmeai", &providerDeleteTestProvider{name: "acmeai", models: []string{"acme-chat-1"}, enabled: true})
	registry.Register("claude", &providerDeleteTestProvider{name: "claude", models: []string{"claude-3-5-sonnet-20241022"}, enabled: true})

	manager := limiter.NewAccountManager(nil, nil)
	if err := manager.AddAccount(&limiter.AccountConfig{ID: "acc-custom", Name: "custom", Provider: "my-custom", ProviderType: "my-custom", Enabled: true}); err != nil {
		t.Fatalf("add account: %v", err)
	}

	handler := NewProviderHandler(registry, manager, nil, "")
	engine := newProviderTypesTestRouter(handler, &AccountHandler{manager: manager})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/providers/types", http.NoBody)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success=true, body=%s", rec.Body.String())
	}

	providerByID := make(map[string]map[string]interface{}, len(payload.Data))
	for _, item := range payload.Data {
		id, _ := item["id"].(string)
		if id != "" {
			providerByID[id] = item
		}
	}

	if _, ok := providerByID["openai"]; !ok {
		t.Fatalf("expected built-in provider openai")
	}
	if _, ok := providerByID["acmeai"]; !ok {
		t.Fatalf("expected dynamic provider acmeai")
	}
	if _, ok := providerByID["my-custom"]; !ok {
		t.Fatalf("expected account provider my-custom")
	}
	if _, ok := providerByID["anthropic"]; !ok {
		t.Fatalf("expected claude alias normalized to anthropic")
	}
	if _, ok := providerByID["claude"]; ok {
		t.Fatalf("expected claude alias not exposed directly")
	}
}

func TestProviderHandler_GetProviderTypes_ShouldReturnContractFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := provider.NewRegistry()
	manager := limiter.NewAccountManager(nil, nil)
	handler := NewProviderHandler(registry, manager, nil, "")
	engine := newProviderTypesTestRouter(handler, &AccountHandler{manager: manager})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/providers/types", http.NoBody)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Data) == 0 {
		t.Fatalf("expected non-empty data")
	}

	var openai map[string]interface{}
	for _, item := range payload.Data {
		if id, _ := item["id"].(string); id == "openai" {
			openai = item
			break
		}
	}
	if openai == nil {
		t.Fatalf("expected openai item in payload")
	}

	requiredFields := []string{
		"id",
		"label",
		"category",
		"color",
		"logo",
		"default_endpoint",
		"coding_endpoint",
		"supports_coding_plan",
		"models",
	}
	for _, field := range requiredFields {
		if _, ok := openai[field]; !ok {
			t.Fatalf("expected field %s in openai item", field)
		}
	}

	if _, ok := openai["supports_coding_plan"].(bool); !ok {
		t.Fatalf("expected supports_coding_plan to be bool")
	}
	if _, ok := openai["models"].([]interface{}); !ok {
		t.Fatalf("expected models to be array")
	}
}

func newProviderTypesTestRouter(providerHandler *ProviderHandler, accountHandler *AccountHandler) *gin.Engine {
	engine := gin.New()
	adminGroup := engine.Group("/api/admin")

	RegisterRoutes(adminGroup, &Handlers{
		Account:     accountHandler,
		Provider:    providerHandler,
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
		Edition:     &EditionHandler{},
		VectorDB:    &vectordb.CollectionHandler{},
	})

	return engine
}
