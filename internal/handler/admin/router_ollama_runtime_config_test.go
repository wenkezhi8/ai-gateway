package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

func TestRouterHandler_GetOllamaRuntimeConfig_ShouldReturnConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defaultCfg := service.DefaultOllamaServiceConfig()
	h := &RouterHandler{ollamaService: service.NewOllamaService(&defaultCfg)}
	r := gin.New()
	r.GET("/api/admin/router/ollama/runtime-config", h.GetOllamaRuntimeConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/router/ollama/runtime-config", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", body["data"])
	}
	cfg, ok := data["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config object, got %#v", data["config"])
	}
	if cfg["startup_mode"] != "auto" {
		t.Fatalf("startup_mode = %#v, want auto", cfg["startup_mode"])
	}
	preload, ok := cfg["preload"].(map[string]any)
	if !ok {
		t.Fatalf("expected preload object, got %#v", cfg["preload"])
	}
	if preload["timeout_seconds"] != float64(180) {
		t.Fatalf("preload.timeout_seconds = %#v, want 180", preload["timeout_seconds"])
	}
}

func TestRouterHandler_UpdateOllamaRuntimeConfig_InvalidMode_ShouldFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defaultCfg := service.DefaultOllamaServiceConfig()
	h := &RouterHandler{ollamaService: service.NewOllamaService(&defaultCfg)}
	r := gin.New()
	r.PUT("/api/admin/router/ollama/runtime-config", h.UpdateOllamaRuntimeConfig)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/ollama/runtime-config", bytes.NewBufferString(`{"startup_mode":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRouterHandler_UpdateOllamaRuntimeConfig_ShouldPersistPreloadFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defaultCfg := service.DefaultOllamaServiceConfig()
	h := &RouterHandler{ollamaService: service.NewOllamaService(&defaultCfg)}
	r := gin.New()
	r.PUT("/api/admin/router/ollama/runtime-config", h.UpdateOllamaRuntimeConfig)
	r.GET("/api/admin/router/ollama/runtime-config", h.GetOllamaRuntimeConfig)

	putReq := httptest.NewRequest(http.MethodPut, "/api/admin/router/ollama/runtime-config", bytes.NewBufferString(`{
		"preload":{
			"auto_on_startup":true,
			"targets":["intent","embedding"],
			"timeout_seconds":180
		}
	}`))
	putReq.Header.Set("Content-Type", "application/json")
	putRes := httptest.NewRecorder()
	r.ServeHTTP(putRes, putReq)
	if putRes.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d", putRes.Code, http.StatusOK)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/router/ollama/runtime-config", http.NoBody)
	getRes := httptest.NewRecorder()
	r.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getRes.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(getRes.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", body["data"])
	}
	cfg, ok := data["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config object, got %#v", data["config"])
	}
	preload, ok := cfg["preload"].(map[string]any)
	if !ok {
		t.Fatalf("expected preload object, got %#v", cfg["preload"])
	}

	if preload["auto_on_startup"] != true {
		t.Fatalf("preload.auto_on_startup = %#v, want true", preload["auto_on_startup"])
	}
	targets, ok := preload["targets"].([]any)
	if !ok || len(targets) != 2 {
		t.Fatalf("preload.targets = %#v, want 2 items", preload["targets"])
	}
	if preload["timeout_seconds"] != float64(180) {
		t.Fatalf("preload.timeout_seconds = %#v, want 180", preload["timeout_seconds"])
	}
}
