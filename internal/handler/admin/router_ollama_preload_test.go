package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/routing"
	"ai-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

func TestRouterHandler_PreloadOllamaModels_ShouldUseConfiguredTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"models":[{"name":"qwen2.5:0.5b-instruct"},{"name":"nomic-embed-text"}]}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		case "/api/chat":
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"message":{"content":"ok"}}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		case "/api/embed":
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	routerCore := routing.NewSmartRouter()
	classifierCfg := routerCore.GetClassifierConfig()
	classifierCfg.BaseURL = server.URL
	classifierCfg.ActiveModel = "qwen2.5:0.5b-instruct"
	routerCore.SetClassifierConfig(classifierCfg)

	cacheManager, err := cache.NewManager(cache.DefaultManagerConfig())
	if err != nil {
		t.Fatalf("create cache manager failed: %v", err)
	}
	settings := cacheManager.GetSettings()
	settings.VectorOllamaBaseURL = server.URL
	settings.VectorOllamaEmbeddingModel = "nomic-embed-text"
	settings.VectorOllamaEndpointMode = cache.OllamaEndpointModeEmbed
	cacheManager.UpdateSettings(settings)

	svcCfg := service.DefaultOllamaServiceConfig()
	h := &RouterHandler{router: routerCore, cacheManager: cacheManager, ollamaService: service.NewOllamaService(&svcCfg)}

	r := gin.New()
	r.POST("/api/admin/router/ollama/preload", h.PreloadOllamaModels)

	reqBody := []byte(`{"targets":["intent","embedding"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/router/ollama/preload", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", body["data"])
	}
	results, ok := data["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("results = %#v, want 2 items", data["results"])
	}
}

func TestRouterHandler_PreloadOllamaModels_ShouldDeduplicateSameModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	chatCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"models":[{"name":"qwen2.5:0.5b-instruct"}]}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		if r.URL.Path == "/api/chat" {
			chatCalls++
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"message":{"content":"ok"}}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		if r.URL.Path == "/api/embed" {
			chatCalls++
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	routerCore := routing.NewSmartRouter()
	classifierCfg := routerCore.GetClassifierConfig()
	classifierCfg.BaseURL = server.URL
	classifierCfg.ActiveModel = "qwen2.5:0.5b-instruct"
	routerCore.SetClassifierConfig(classifierCfg)

	cacheManager, err := cache.NewManager(cache.DefaultManagerConfig())
	if err != nil {
		t.Fatalf("create cache manager failed: %v", err)
	}
	settings := cacheManager.GetSettings()
	settings.VectorOllamaBaseURL = server.URL
	settings.VectorOllamaEmbeddingModel = "qwen2.5:0.5b-instruct"
	settings.VectorOllamaEndpointMode = cache.OllamaEndpointModeEmbed
	cacheManager.UpdateSettings(settings)

	svcCfg := service.DefaultOllamaServiceConfig()
	h := &RouterHandler{router: routerCore, cacheManager: cacheManager, ollamaService: service.NewOllamaService(&svcCfg)}

	r := gin.New()
	r.POST("/api/admin/router/ollama/preload", h.PreloadOllamaModels)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/router/ollama/preload", bytes.NewBufferString(`{"targets":["intent","embedding"]}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if chatCalls != 1 {
		t.Fatalf("preload calls = %d, want 1 (deduplicated)", chatCalls)
	}
}
