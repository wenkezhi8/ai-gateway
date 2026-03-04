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
	"time"

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

func TestEditionAPI_SetupEdition_InvalidRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetEditionSetupTasksForTest()

	h := NewEditionHandler()
	r := gin.New()
	r.POST("/api/admin/edition/setup", h.SetupEditionEnvironment)

	body := []byte(`{"edition":"standard","runtime":"invalid","apply_config":false,"pull_embedding_model":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/edition/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestEditionAPI_SetupEdition_Accepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetEditionSetupTasksForTest()

	origExecutor := editionSetupExecutor
	defer func() { editionSetupExecutor = origExecutor }()
	editionSetupExecutor = func(_ string, _ EditionSetupRequest, appendLog EditionSetupLogAppender) (string, error) {
		if appendLog != nil {
			appendLog("setup success")
		}
		return "installed redis, ollama", nil
	}

	h := NewEditionHandler()
	r := gin.New()
	r.POST("/api/admin/edition/setup", h.SetupEditionEnvironment)

	body := []byte(`{"edition":"standard","runtime":"docker","apply_config":true,"pull_embedding_model":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/edition/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusAccepted, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success = false")
	}
	if resp.Data.TaskID == "" {
		t.Fatalf("task id should not be empty")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		task, ok := getEditionSetupTask(resp.Data.TaskID)
		if ok && (task.Status == EditionSetupStatusSuccess || task.Status == EditionSetupStatusFailed) {
			if task.Status != EditionSetupStatusSuccess {
				t.Fatalf("task status = %s, want %s, message=%s", task.Status, EditionSetupStatusSuccess, task.Message)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("setup task %s did not finish before timeout", resp.Data.TaskID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditionAPI_GetSetupTask_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetEditionSetupTasksForTest()

	h := NewEditionHandler()
	r := gin.New()
	r.GET("/api/admin/edition/setup/tasks/:taskId", h.GetSetupTask)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/edition/setup/tasks/not-exist", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestEditionAPI_SetupEditionTask_RunningStatusShowsIncrementalLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetEditionSetupTasksForTest()

	origExecutor := editionSetupExecutor
	defer func() { editionSetupExecutor = origExecutor }()
	editionSetupExecutor = func(_ string, _ EditionSetupRequest, appendLog EditionSetupLogAppender) (string, error) {
		if appendLog != nil {
			appendLog("[setup-edition] step-1")
		}
		time.Sleep(300 * time.Millisecond)
		if appendLog != nil {
			appendLog("[setup-edition] step-2")
		}
		return "streamed logs", nil
	}

	h := NewEditionHandler()
	r := gin.New()
	r.POST("/api/admin/edition/setup", h.SetupEditionEnvironment)

	body := []byte(`{"edition":"standard","runtime":"docker","apply_config":false,"pull_embedding_model":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/edition/setup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusAccepted, w.Body.String())
	}

	var resp struct {
		Data struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		task, ok := getEditionSetupTask(resp.Data.TaskID)
		if ok && task.Status == EditionSetupStatusRunning {
			if !strings.Contains(task.Logs, "step-1") {
				t.Fatalf("running task logs should include incremental output, got=%q", task.Logs)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s did not enter running status before timeout", resp.Data.TaskID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
