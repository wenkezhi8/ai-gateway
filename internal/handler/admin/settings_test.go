package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-gateway/internal/constants"

	"github.com/gin-gonic/gin"
)

func TestSettingsUI_GetDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSettingsHandler(filepath.Join(t.TempDir(), "ui-settings.json"))
	router := gin.New()
	router.GET("/api/admin/settings/ui", handler.GetUISettings)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings/ui", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Success bool       `json:"success"`
		Data    UISettings `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success = false, want true")
	}
	if resp.Data.Settings == nil {
		t.Fatalf("settings should not be nil")
	}
}

func TestSettingsUI_PutAndPersist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	filePath := filepath.Join(t.TempDir(), "ui-settings.json")
	handler := NewSettingsHandler(filePath)
	router := gin.New()
	router.PUT("/api/admin/settings/ui", handler.UpdateUISettings)
	router.GET("/api/admin/settings/ui", handler.GetUISettings)

	reqBody := map[string]any{
		"routing": map[string]any{
			"auto_save_enabled": true,
			"last_saved_at":     "2026-02-28T08:00:00Z",
		},
		"model_management": map[string]any{
			"last_saved_at": "2026-02-28T08:01:00Z",
		},
		"settings": map[string]any{
			"theme": "light",
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request body failed: %v", err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/admin/settings/ui", bytes.NewReader(payload))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	router.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d", putW.Code, http.StatusOK)
	}

	// Re-create handler from the same file path to verify persistence.
	handler2 := NewSettingsHandler(filePath)
	router2 := gin.New()
	router2.GET("/api/admin/settings/ui", handler2.GetUISettings)

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/settings/ui", http.NoBody)
	getW := httptest.NewRecorder()
	router2.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getW.Code, http.StatusOK)
	}

	var resp struct {
		Success bool       `json:"success"`
		Data    UISettings `json:"data"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success = false, want true")
	}
	if !resp.Data.Routing.AutoSaveEnabled {
		t.Fatalf("routing.auto_save_enabled = false, want true")
	}
	if resp.Data.Routing.LastSavedAt != "2026-02-28T08:00:00Z" {
		t.Fatalf("routing.last_saved_at = %q", resp.Data.Routing.LastSavedAt)
	}
	if resp.Data.ModelManagement.LastSavedAt != "2026-02-28T08:01:00Z" {
		t.Fatalf("model_management.last_saved_at = %q", resp.Data.ModelManagement.LastSavedAt)
	}
	if got := resp.Data.Settings["theme"]; got != "light" {
		t.Fatalf("settings.theme = %v, want light", got)
	}
}

func TestSettingsUI_InvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSettingsHandler(filepath.Join(t.TempDir(), "ui-settings.json"))
	router := gin.New()
	router.PUT("/api/admin/settings/ui", handler.UpdateUISettings)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings/ui", bytes.NewReader([]byte(`["invalid"]`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Success {
		t.Fatalf("success = true, want false")
	}
	if resp.Error.Code == "" {
		t.Fatalf("error.code should not be empty")
	}
}

func TestSettingsDefaults_Get_ShouldReturnContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSettingsHandler(filepath.Join(t.TempDir(), "ui-settings.json"))
	router := gin.New()
	router.GET("/api/admin/settings/defaults", handler.GetSettingsDefaults)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings/defaults", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	for _, key := range []string{"gateway", "cache", "logging", "security"} {
		if _, ok := resp.Data[key]; !ok {
			t.Fatalf("data.%s missing", key)
		}
	}
}

func TestRegisterRoutes_ShouldExposeSettingsDefaultsEndpoint(t *testing.T) {
	if constants.AdminSettingsDefaults != "/api/admin/settings/defaults" {
		t.Fatalf("AdminSettingsDefaults = %q", constants.AdminSettingsDefaults)
	}

	adminFile := filepath.Join("..", "..", "handler", "admin", "admin.go")
	content, err := os.ReadFile(adminFile)
	if err != nil {
		t.Fatalf("read admin.go failed: %v", err)
	}

	if !strings.Contains(string(content), "settings.GET(\"/defaults\", handlers.Settings.GetSettingsDefaults)") {
		t.Fatalf("defaults route registration missing in RegisterRoutes")
	}
}
