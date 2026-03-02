package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-gateway/internal/config"

	"github.com/gin-gonic/gin"
)

func TestEditionAPI_GetEdition(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	cfg.Edition.Type = string(config.EditionBasic)
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config failed: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	orig := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", orig)
	os.Setenv("CONFIG_PATH", configPath)

	h := NewEditionHandler()
	r := gin.New()
	r.GET("/api/admin/edition", h.GetEdition)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/edition", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestEditionAPI_UpdateEdition(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config failed: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	orig := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", orig)
	os.Setenv("CONFIG_PATH", configPath)

	origDeps := dependencyStatusProvider
	defer func() { dependencyStatusProvider = origDeps }()
	dependencyStatusProvider = func(_ *config.Config) map[string]DependencyStatus {
		return map[string]DependencyStatus{
			"redis":  {Healthy: true},
			"ollama": {Healthy: true},
			"qdrant": {Healthy: true},
		}
	}

	h := NewEditionHandler()
	r := gin.New()
	r.PUT("/api/admin/edition", h.UpdateEdition)

	body := []byte(`{"type":"standard"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/edition", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}
