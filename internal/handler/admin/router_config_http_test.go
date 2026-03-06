package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"ai-gateway/internal/routing"

	"github.com/gin-gonic/gin"
)

func TestRouterHandler_GetRouterConfig_ShouldReturnModeContractAndMigrationNotice(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	persistedConfig = &PersistedRouterConfig{
		UseAutoMode:     "latest",
		DefaultStrategy: "auto",
		DefaultModel:    "qwen-plus",
		Classifier:      routing.DefaultClassifierConfig(),
	}
	t.Cleanup(func() {
		persistedConfig = nil
	})

	handler := &RouterHandler{router: routing.NewSmartRouter()}
	router := gin.New()
	router.GET("/api/admin/router/config", handler.GetRouterConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/router/config", http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is not object: %#v", payload["data"])
	}

	if got := data["use_auto_mode"]; got != "auto" {
		t.Fatalf("expected use_auto_mode auto, got %#v", got)
	}
	if got := data["migration_notice"]; got == "" {
		t.Fatalf("expected migration_notice, got %#v", got)
	}
	contract, ok := data["use_auto_mode_contract"].(map[string]any)
	if !ok {
		t.Fatalf("expected use_auto_mode_contract object, got %#v", data["use_auto_mode_contract"])
	}
	allowedModes, ok := contract["allowed_modes"].([]any)
	if !ok || len(allowedModes) != 3 {
		t.Fatalf("expected 3 allowed modes, got %#v", contract["allowed_modes"])
	}
}

func TestRouterHandler_UpdateRouterConfig_ShouldReturnModeMigration(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	persistedConfig = &PersistedRouterConfig{
		UseAutoMode:     "auto",
		DefaultStrategy: "auto",
		DefaultModel:    "qwen-plus",
		Classifier:      routing.DefaultClassifierConfig(),
	}
	t.Cleanup(func() {
		persistedConfig = nil
	})

	handler := &RouterHandler{router: routing.NewSmartRouter()}
	router := gin.New()
	router.PUT("/api/admin/router/config", handler.UpdateRouterConfig)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/router/config", bytes.NewBufferString(`{"use_auto_mode":"latest"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if got := payload["use_auto_mode"]; got != "auto" {
		t.Fatalf("expected use_auto_mode auto, got %#v", got)
	}
	migration, ok := payload["mode_migration"].(map[string]any)
	if !ok {
		t.Fatalf("expected mode_migration object, got %#v", payload["mode_migration"])
	}
	if migration["from"] != "latest" || migration["to"] != "auto" {
		t.Fatalf("unexpected mode_migration: %#v", migration)
	}
}
