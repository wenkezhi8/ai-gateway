package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-gateway/internal/cache"
	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

func newOllamaTagsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		err := json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"name": "qwen2.5:0.5b-instruct"},
				{"name": "nomic-embed-text"},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
}

func TestRouterHandler_OllamaDualModelConfigEndpoints_ShouldGetAndUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 隔离 router_ui_config / router_config 写入路径，避免污染仓库目录。
	tempDir := t.TempDir()
	originWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originWd); chdirErr != nil {
			t.Fatalf("cleanup chdir failed: %v", chdirErr)
		}
	})

	// 隔离 vector_cache 持久化路径。
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"server":{"port":"8566"},"vector_cache":{"enabled":true}}`), 0o644); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}
	originConfigPath := os.Getenv("CONFIG_PATH")
	t.Cleanup(func() {
		_ = os.Setenv("CONFIG_PATH", originConfigPath)
	})
	_ = os.Setenv("CONFIG_PATH", configPath)

	ollamaServer := newOllamaTagsServer(t)
	defer ollamaServer.Close()

	smartRouter := routing.NewSmartRouter()
	classifierCfg := smartRouter.GetClassifierConfig()
	classifierCfg.BaseURL = ollamaServer.URL
	classifierCfg.ActiveModel = "qwen2.5:0.5b-instruct"
	classifierCfg.CandidateModels = []string{"qwen2.5:0.5b-instruct"}
	classifierCfg.TimeoutMs = 300
	smartRouter.SetClassifierConfig(classifierCfg)

	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	settings := manager.GetSettings()
	settings.VectorPipelineEnabled = true
	settings.VectorOllamaBaseURL = ollamaServer.URL
	settings.VectorOllamaEmbeddingModel = "nomic-embed-text"
	settings.VectorOllamaEmbeddingDimension = 1024
	settings.VectorOllamaEmbeddingTimeoutMs = 1200
	settings.VectorOllamaEndpointMode = cache.OllamaEndpointModeAuto
	settings.VectorWritebackEnabled = true
	manager.UpdateSettings(settings)

	persistedConfig = nil
	handler := &RouterHandler{
		router:       smartRouter,
		cacheManager: manager,
	}

	router := gin.New()
	router.GET("/api/admin/router/ollama/dual-model/config", handler.GetOllamaDualModelConfig)
	router.PUT("/api/admin/router/ollama/dual-model/config", handler.UpdateOllamaDualModelConfig)

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/router/ollama/dual-model/config", http.NoBody)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d body=%s", getW.Code, getW.Body.String())
	}
	var getResp map[string]any
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response failed: %v", err)
	}
	data, ok := getResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("get response data is not object: %#v", getResp["data"])
	}
	if data["classifier_active_model"] != "qwen2.5:0.5b-instruct" {
		t.Fatalf("unexpected classifier_active_model: %#v", data["classifier_active_model"])
	}
	if data["vector_ollama_embedding_model"] != "nomic-embed-text" {
		t.Fatalf("unexpected vector_ollama_embedding_model: %#v", data["vector_ollama_embedding_model"])
	}

	putBody := []byte(`{
		"classifier_active_model":"qwen2.5:1.5b-instruct",
		"classifier_candidate_models":["qwen2.5:1.5b-instruct","qwen2.5:0.5b-instruct"],
		"vector_ollama_embedding_model":"bge-m3",
		"vector_ollama_endpoint_mode":"embed",
		"vector_ollama_embedding_timeout_ms":1800,
		"vector_writeback_enabled":false
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/admin/router/ollama/dual-model/config", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	router.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("expected 200 on put, got %d body=%s", putW.Code, putW.Body.String())
	}

	updatedClassifier := smartRouter.GetClassifierConfig()
	if updatedClassifier.ActiveModel != "qwen2.5:1.5b-instruct" {
		t.Fatalf("classifier active model not updated, got %s", updatedClassifier.ActiveModel)
	}

	updatedSettings := manager.GetSettings()
	if updatedSettings.VectorOllamaEmbeddingModel != "bge-m3" {
		t.Fatalf("vector embedding model not updated, got %s", updatedSettings.VectorOllamaEmbeddingModel)
	}
	if updatedSettings.VectorOllamaEndpointMode != cache.OllamaEndpointModeEmbed {
		t.Fatalf("vector endpoint mode not updated, got %s", updatedSettings.VectorOllamaEndpointMode)
	}
	if updatedSettings.VectorWritebackEnabled {
		t.Fatal("expected vector writeback disabled")
	}
}

func TestRouterHandler_UpdateOllamaDualModelConfig_ShouldRejectInvalidEndpointMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := cache.NewManagerWithCache(cache.NewMemoryCache())
	handler := &RouterHandler{
		router:       routing.NewSmartRouter(),
		cacheManager: manager,
	}

	router := gin.New()
	router.PUT("/api/admin/router/ollama/dual-model/config", handler.UpdateOllamaDualModelConfig)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/ollama/dual-model/config", bytes.NewBufferString(`{"vector_ollama_endpoint_mode":"invalid-mode"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRouterHandler_IntentEngineEndpoints_ShouldBe404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &RouterHandler{
		router:       routing.NewSmartRouter(),
		cacheManager: cache.NewManagerWithCache(cache.NewMemoryCache()),
	}

	router := gin.New()
	router.GET("/api/admin/router/ollama/dual-model/config", handler.GetOllamaDualModelConfig)
	router.PUT("/api/admin/router/ollama/dual-model/config", handler.UpdateOllamaDualModelConfig)

	for _, path := range []string{
		"/api/admin/router/intent-engine/config",
		"/api/admin/router/intent-engine/health",
	} {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for %s, got %d", path, w.Code)
		}
	}
}
